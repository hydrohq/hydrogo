package security

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"
)

type distributedStore struct {
	ok  bool
	err error
}

func (s distributedStore) Allow(context.Context, string) (bool, error) { return s.ok, s.err }

func TestDistributedLimiterFailsClosedOnStoreFailure(t *testing.T) {
	if (DistributedLimiter{Store: distributedStore{ok: true}}).Allow(context.Background(), "login") != true {
		t.Fatal("available shared store rejected request")
	}
	if (DistributedLimiter{Store: distributedStore{ok: true, err: errors.New("backend down")}}).Allow(context.Background(), "login") {
		t.Fatal("shared store failure opened limiter")
	}
	if (DistributedLimiter{}).Allow(context.Background(), "login") {
		t.Fatal("missing shared store opened limiter")
	}
}

func TestDistributedLimiterRejectsEmptyKeysBeforeStore(t *testing.T) {
	calls := 0
	store := distributedStoreFunc(func(context.Context, string) (bool, error) {
		calls++
		return true, nil
	})
	if (DistributedLimiter{Store: store}).Allow(context.Background(), "") {
		t.Fatal("empty distributed limiter key was accepted")
	}
	if calls != 0 {
		t.Fatalf("empty key reached shared store %d times", calls)
	}
}

type distributedStoreFunc func(context.Context, string) (bool, error)

func (f distributedStoreFunc) Allow(ctx context.Context, key string) (bool, error) {
	return f(ctx, key)
}

func TestAttemptLimiterBoundsAttemptsAndMemory(t *testing.T) {
	limiter := NewAttemptLimiter(1, 2, time.Hour)
	if !limiter.Allow("first") || !limiter.Allow("first") || limiter.Allow("first") {
		t.Fatal("attempt limit was not enforced")
	}
	if !limiter.Allow("second") {
		t.Fatal("bounded limiter rejected a fresh key")
	}
}

func TestAttemptLimiterRejectsEmptyKeysAndEvictsExpiredEntriesFirst(t *testing.T) {
	limiter := NewAttemptLimiter(2, 1, 5*time.Millisecond)
	if limiter.Allow("") {
		t.Fatal("empty limiter key was accepted")
	}
	if !limiter.Allow("expired") || !limiter.Allow("active") {
		t.Fatal("initial limiter keys were rejected")
	}
	time.Sleep(10 * time.Millisecond)
	if !limiter.Allow("fresh") {
		t.Fatal("fresh key was rejected after expired entry cleanup")
	}
	if _, ok := limiter.entries[sha256.Sum256([]byte("expired"))]; ok {
		t.Fatal("expired entry was not evicted")
	}
}

func TestAttemptLimiterStoresFixedSizeDigests(t *testing.T) {
	limiter := NewAttemptLimiter(2, 1, time.Hour)
	first := strings.Repeat("a", 1<<20)
	second := strings.Repeat("b", 1<<20)
	if !limiter.Allow(first) || !limiter.Allow(second) {
		t.Fatal("long limiter keys were rejected")
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if len(limiter.entries) != 2 {
		t.Fatalf("stored %d limiter entries, want 2", len(limiter.entries))
	}
	if _, ok := limiter.entries[sha256.Sum256([]byte(first))]; !ok {
		t.Fatal("first key was not represented by its digest")
	}
	if _, ok := limiter.entries[sha256.Sum256([]byte(second))]; !ok {
		t.Fatal("second key was not represented by its digest")
	}
}
