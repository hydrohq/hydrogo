package security

import (
	"crypto/sha256"
	"sync"
	"time"
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
	if l == nil || key == "" {
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
