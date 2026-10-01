package security

import (
	"context"
	"crypto/sha256"
	"net/http"
	"sync"
	"time"
)

type failureWindow struct {
	count    uint8
	lastSeen time.Time
}

// Tarpit adds a bounded delay after failed authentication attempts. Keys are
// digested before storage so account names and IP addresses are not retained.
type Tarpit struct {
	mu         sync.Mutex
	entries    map[[32]byte]failureWindow
	maxEntries int
	expires    time.Duration
	baseDelay  time.Duration
	maxDelay   time.Duration
	now        func() time.Time
	Extractor  interface {
		Extract(*http.Request) string
	}
}

func NewTarpit(maxEntries int, expires, baseDelay, maxDelay time.Duration) *Tarpit {
	if maxEntries < 1 {
		maxEntries = 1
	}
	if expires <= 0 {
		expires = 15 * time.Minute
	}
	if baseDelay <= 0 {
		baseDelay = time.Second
	}
	if maxDelay <= 0 {
		maxDelay = 8 * time.Second
	}
	if maxDelay < baseDelay {
		maxDelay = baseDelay
	}
	return &Tarpit{entries: make(map[[32]byte]failureWindow), maxEntries: maxEntries, expires: expires, baseDelay: baseDelay, maxDelay: maxDelay, now: time.Now}
}

// Failure records an unsuccessful attempt and returns its bounded delay.
func (t *Tarpit) Failure(key string) time.Duration {
	if t == nil || key == "" || t.maxEntries < 1 || t.expires <= 0 || t.baseDelay <= 0 {
		return 0
	}
	digest := sha256.Sum256([]byte(key))
	now := time.Now()
	if t.now != nil {
		now = t.now()
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.evict(now)
	entry, exists := t.entries[digest]
	if !exists && len(t.entries) >= t.maxEntries {
		t.evictOldest()
	}
	if !exists || now.Sub(entry.lastSeen) >= t.expires {
		entry = failureWindow{}
	}
	if entry.count < 255 {
		entry.count++
	}
	entry.lastSeen = now
	t.entries[digest] = entry
	delay := t.baseDelay
	for i := uint8(1); i < entry.count && delay < t.maxDelay; i++ {
		delay *= 2
	}
	if delay > t.maxDelay {
		return t.maxDelay
	}
	return delay
}

// Reset forgets the failure history after successful authentication.
func (t *Tarpit) Reset(key string) {
	if t == nil || key == "" {
		return
	}
	digest := sha256.Sum256([]byte(key))
	t.mu.Lock()
	delete(t.entries, digest)
	t.mu.Unlock()
}

// Wait records a failure and waits without ignoring request cancellation.
func (t *Tarpit) Wait(ctx context.Context, key string) error {
	delay := t.Failure(key)
	if delay <= 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (t *Tarpit) evict(now time.Time) {
	for key, entry := range t.entries {
		if now.Sub(entry.lastSeen) >= t.expires {
			delete(t.entries, key)
		}
	}
}

func (t *Tarpit) evictOldest() {
	var oldestKey [32]byte
	var oldest time.Time
	for key, entry := range t.entries {
		if oldest.IsZero() || entry.lastSeen.Before(oldest) {
			oldest, oldestKey = entry.lastSeen, key
		}
	}
	delete(t.entries, oldestKey)
}
