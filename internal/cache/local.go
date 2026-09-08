package cache

import (
	"sync"
	"time"
)

type item[T any] struct {
	value     T
	expiresAt time.Time
}

// Cache is a bounded, process-local cache. It never grows beyond maxEntries.
type Cache[T any] struct {
	mu         sync.Mutex
	entries    map[string]item[T]
	maxEntries int
	ttl        time.Duration
}

func New[T any](maxEntries int, ttl time.Duration) *Cache[T] {
	if maxEntries < 1 {
		maxEntries = 1
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	return &Cache[T]{entries: make(map[string]item[T]), maxEntries: maxEntries, ttl: ttl}
}

func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		delete(c.entries, key)
		var zero T
		return zero, false
	}
	return entry.value, true
}

func (c *Cache[T]) Set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.entries[key]; !exists && len(c.entries) >= c.maxEntries {
		for oldestKey := range c.entries {
			delete(c.entries, oldestKey)
			break
		}
	}
	c.entries[key] = item[T]{value: value, expiresAt: time.Now().Add(c.ttl)}
}

func (c *Cache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]item[T])
}
