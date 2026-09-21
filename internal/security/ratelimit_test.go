package security

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
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

func TestDistributedLimiterDoesNotForwardRawKeyToSharedStore(t *testing.T) {
	const rawKey = "203.0.113.10:person@example.test"
	var received string
	store := distributedStoreFunc(func(_ context.Context, key string) (bool, error) {
		received = key
		return true, nil
	})
	if !(DistributedLimiter{Store: store}).Allow(context.Background(), rawKey) {
		t.Fatal("shared store rejected valid key")
	}
	if received == rawKey || strings.Contains(received, "person@example.test") || strings.Contains(received, "203.0.113.10") {
		t.Fatalf("raw limiter key reached shared store: %q", received)
	}
	want := sha256.Sum256([]byte(rawKey))
	if received != fmt.Sprintf("%x", want) {
		t.Fatalf("shared store key = %q, want sha256 digest %x", received, want)
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

func TestAttemptLimiterFailsClosedWhenZeroValue(t *testing.T) {
	var nilLimiter *AttemptLimiter
	if nilLimiter.Allow("login") {
		t.Fatal("nil limiter accepted an attempt")
	}
	zeroLimiter := AttemptLimiter{}
	if zeroLimiter.Allow("login") {
		t.Fatal("zero-value limiter accepted an attempt")
	}
}

func TestTenantScopedRateLimiterSeparatesBucketsAndFailsClosed(t *testing.T) {
	var received []string
	inner := rateLimiterFunc(func(key string) bool {
		received = append(received, key)
		return true
	})
	limiter := TenantScopedRateLimiter{Inner: inner}
	alpha, err := tenant.NewScope("tenant-alpha", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	beta, err := tenant.NewScope("tenant-beta", "beta")
	if err != nil {
		t.Fatal(err)
	}
	if !limiter.AllowContext(tenant.WithScope(context.Background(), alpha), "198.51.100.10:member@example.test") || !limiter.AllowContext(tenant.WithScope(context.Background(), beta), "198.51.100.10:member@example.test") {
		t.Fatal("scoped limiter rejected valid requests")
	}
	if limiter.AllowContext(context.Background(), "198.51.100.10:member@example.test") {
		t.Fatal("missing tenant scope was accepted")
	}
	if len(received) != 2 || received[0] == received[1] || strings.Contains(received[0], "member@example.test") || strings.Contains(received[0], "198.51.100.10") {
		t.Fatalf("tenant limiter keys = %#v", received)
	}
}

type rateLimiterFunc func(string) bool

func (f rateLimiterFunc) Allow(key string) bool { return f(key) }
