package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type attempt struct {
	count   int
	resetAt time.Time
}

// AttemptLimiter bounds authentication work per key and bounds its own memory.
type AttemptLimiter struct {
	mu         sync.Mutex
	entries    map[[32]byte]attempt
	maxEntries int
	limit      int
	window     time.Duration
}

type RateLimiter interface {
	Allow(string) bool
}

// ContextRateLimiter receives the request lifetime when a limiter supports it.
type ContextRateLimiter interface {
	AllowContext(context.Context, string) bool
}

// AllowForRequest preserves the basic limiter contract while forwarding request
// cancellation and tenant context to implementations that support it.
func AllowForRequest(l RateLimiter, ctx context.Context, key string) bool {
	if l == nil {
		return false
	}
	if contextual, ok := l.(ContextRateLimiter); ok {
		return contextual.AllowContext(ctx, key)
	}
	return l.Allow(key)
}

// TenantScopedRateLimiter isolates each tenant's buckets. It fails closed when
// a shared-mode request has no validated tenant scope and never exposes the raw
// caller key to the underlying limiter.
type TenantScopedRateLimiter struct{ Inner RateLimiter }

func (l TenantScopedRateLimiter) Allow(string) bool { return false }

func (l TenantScopedRateLimiter) AllowContext(ctx context.Context, key string) bool {
	if l.Inner == nil {
		return false
	}
	scope, err := tenant.Require(ctx)
	if err != nil || key == "" {
		return false
	}
	digest := sha256.Sum256([]byte(key))
	return AllowForRequest(l.Inner, ctx, "tenant:"+scope.TenantID+":"+hex.EncodeToString(digest[:]))
}

func NewAttemptLimiter(maxEntries, limit int, window time.Duration) *AttemptLimiter {
	if maxEntries < 1 {
		maxEntries = 1
	}
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &AttemptLimiter{entries: make(map[[32]byte]attempt), maxEntries: maxEntries, limit: limit, window: window}
}

func (l *AttemptLimiter) Allow(key string) bool {
	if l == nil || l.entries == nil || l.maxEntries < 1 || l.limit < 1 || l.window <= 0 || key == "" {
		return false
	}
	// Keep attacker-controlled and potentially sensitive key material out of the
	// bounded map. A fixed-size digest also makes the memory bound meaningful.
	keyDigest := sha256.Sum256([]byte(key))
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	value, ok := l.entries[keyDigest]
	if !ok || !now.Before(value.resetAt) {
		if !ok && len(l.entries) >= l.maxEntries {
			l.evict(now)
		}
		l.entries[keyDigest] = attempt{count: 1, resetAt: now.Add(l.window)}
		return true
	}
	if value.count >= l.limit {
		return false
	}
	value.count++
	l.entries[keyDigest] = value
	return true
}

func (l *AttemptLimiter) evict(now time.Time) {
	for key, value := range l.entries {
		if !now.Before(value.resetAt) {
			delete(l.entries, key)
		}
	}
	if len(l.entries) < l.maxEntries {
		return
	}
	var oldestKey [32]byte
	var foundOldest bool
	var oldestReset time.Time
	for key, value := range l.entries {
		if !foundOldest || value.resetAt.Before(oldestReset) {
			oldestKey = key
			oldestReset = value.resetAt
			foundOldest = true
		}
	}
	if foundOldest {
		delete(l.entries, oldestKey)
	}
}
