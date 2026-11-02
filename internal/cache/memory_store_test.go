package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryStore_BasicKVAndTTL(t *testing.T) {
	store := NewMemoryStore(MemoryStoreOptions{MaxEntries: 10, JanitorInterval: 50 * time.Millisecond})
	defer store.Close()

	store.Set("key1", "val1", 0)
	if val, ok := store.GetString("key1"); !ok || val != "val1" {
		t.Fatalf("expected val1, got %v (ok=%v)", val, ok)
	}

	// Short TTL
	store.Set("temp", "expiring", 30*time.Millisecond)
	if val, ok := store.GetString("temp"); !ok || val != "expiring" {
		t.Fatalf("expected expiring before TTL, got %v", val)
	}

	time.Sleep(50 * time.Millisecond)
	if _, ok := store.Get("temp"); ok {
		t.Fatal("expected temp key to be expired and unavailable")
	}

	// Delete
	if !store.Delete("key1") {
		t.Fatal("expected Delete to return true for existing key")
	}
	if store.Has("key1") {
		t.Fatal("expected key1 to be deleted")
	}
}

func TestMemoryStore_JanitorCleanup(t *testing.T) {
	store := NewMemoryStore(MemoryStoreOptions{MaxEntries: 50, JanitorInterval: 20 * time.Millisecond})
	defer store.Close()

	store.Set("short1", "data1", 10*time.Millisecond)
	store.Set("short2", "data2", 10*time.Millisecond)
	store.Set("persist", "keep", 0)

	if store.Len() != 3 {
		t.Fatalf("expected 3 items before expiration, got %d", store.Len())
	}

	// Wait for janitor pass
	time.Sleep(60 * time.Millisecond)

	// Janitor should have purged short1 and short2
	if store.Len() != 1 {
		t.Fatalf("expected 1 item remaining after janitor pass, got %d", store.Len())
	}
	if !store.Has("persist") {
		t.Fatal("expected persist key to remain active")
	}
}

func TestMemoryStore_AtomicIncrAndRateLimit(t *testing.T) {
	store := NewMemoryStore(MemoryStoreOptions{MaxEntries: 100})
	defer store.Close()

	// Incr from non-existent
	n, err := store.Incr("counter", 5, time.Minute)
	if err != nil || n != 5 {
		t.Fatalf("expected 5, got %d (err: %v)", n, err)
	}

	n, err = store.Incr("counter", 3, time.Minute)
	if err != nil || n != 8 {
		t.Fatalf("expected 8, got %d (err: %v)", n, err)
	}

	// RateLimit check (limit 3 within window)
	rlKey := "user_ip_123"
	window := 100 * time.Millisecond

	allowed1, c1, _ := store.RateLimit(rlKey, 3, window)
	if !allowed1 || c1 != 1 {
		t.Fatalf("expected attempt 1 to be allowed, got allowed=%v, count=%d", allowed1, c1)
	}

	allowed2, c2, _ := store.RateLimit(rlKey, 3, window)
	if !allowed2 || c2 != 2 {
		t.Fatalf("expected attempt 2 to be allowed, got allowed=%v, count=%d", allowed2, c2)
	}

	allowed3, c3, _ := store.RateLimit(rlKey, 3, window)
	if !allowed3 || c3 != 3 {
		t.Fatalf("expected attempt 3 to be allowed, got allowed=%v, count=%d", allowed3, c3)
	}

	// 4th attempt should be blocked
	allowed4, c4, _ := store.RateLimit(rlKey, 3, window)
	if allowed4 || c4 != 4 {
		t.Fatalf("expected attempt 4 to be rejected, got allowed=%v, count=%d", allowed4, c4)
	}

	// After window expires, should allow again
	time.Sleep(120 * time.Millisecond)
	allowed5, c5, _ := store.RateLimit(rlKey, 3, window)
	if !allowed5 || c5 != 1 {
		t.Fatalf("expected new window attempt to be allowed with count 1, got allowed=%v, count=%d", allowed5, c5)
	}
}

func TestMemoryRateLimiter_Allow(t *testing.T) {
	store := NewMemoryStore(MemoryStoreOptions{MaxEntries: 1000})
	defer store.Close()

	limiter := NewMemoryRateLimiter(store, "api:", 2, 80*time.Millisecond)

	if !limiter.Allow("client-a") {
		t.Fatal("expected client-a attempt 1 allowed")
	}
	if !limiter.Allow("client-a") {
		t.Fatal("expected client-a attempt 2 allowed")
	}
	if limiter.Allow("client-a") {
		t.Fatal("expected client-a attempt 3 to be rate-limited")
	}

	// Different client should not be affected
	if !limiter.Allow("client-b") {
		t.Fatal("expected client-b attempt 1 allowed")
	}

	// Wait for window reset
	time.Sleep(100 * time.Millisecond)
	if !limiter.Allow("client-a") {
		t.Fatal("expected client-a to be allowed after window reset")
	}
}

func TestMemoryPubSub(t *testing.T) {
	ps := NewMemoryPubSub()

	sub1, unsub1 := ps.Subscribe("orders", 10)
	sub2, unsub2 := ps.Subscribe("orders", 10)
	defer unsub1()
	defer unsub2()

	delivered := ps.Publish("orders", "order_created_123")
	if delivered != 2 {
		t.Fatalf("expected 2 subscribers to receive, got %d", delivered)
	}

	msg1 := <-sub1
	msg2 := <-sub2
	if msg1 != "order_created_123" || msg2 != "order_created_123" {
		t.Fatalf("expected payload match, got %v and %v", msg1, msg2)
	}

	// Unsubscribe sub1
	unsub1()
	delivered2 := ps.Publish("orders", "order_updated_456")
	if delivered2 != 1 {
		t.Fatalf("expected 1 subscriber to receive after unsub, got %d", delivered2)
	}
}

func TestMemoryWorkerPool(t *testing.T) {
	pool := NewMemoryWorkerPool(3, 20)
	defer pool.Stop()

	var counter atomic.Int64
	var wg sync.WaitGroup

	taskCount := 10
	wg.Add(taskCount)

	for i := 0; i < taskCount; i++ {
		submitted := pool.Submit(func(ctx context.Context) {
			defer wg.Done()
			counter.Add(1)
		})
		if !submitted {
			t.Fatalf("failed to submit task %d", i)
		}
	}

	wg.Wait()
	if counter.Load() != int64(taskCount) {
		t.Fatalf("expected %d completed tasks, got %d", taskCount, counter.Load())
	}
}
