package cache

import (
	"errors"
	"sync"
	"time"
)

type item[T any] struct {
	value     T
	expiresAt time.Time
}

type loadCall[T any] struct {
	done  chan struct{}
	value T
	err   error
}

// Cache is a bounded, process-local cache. It never grows beyond maxEntries.
type Cache[T any] struct {
	mu         sync.Mutex
	entries    map[string]item[T]
	maxEntries int
	ttl        time.Duration
	generation uint64
	loads      map[string]*loadCall[T]
}

func New[T any](maxEntries int, ttl time.Duration) *Cache[T] {
	if maxEntries < 1 {
		maxEntries = 1
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	return &Cache[T]{entries: make(map[string]item[T]), loads: make(map[string]*loadCall[T]), maxEntries: maxEntries, ttl: ttl}
}

// GetOrLoad coalesces concurrent misses for the same key into one bounded
// loader call. A loader started before Clear may still return to its caller,
// but its value cannot repopulate the cache after the generation changes.
func (c *Cache[T]) GetOrLoad(key string, loader func() (T, error)) (T, error) {
	if loader == nil {
		var zero T
		return zero, errors.New("cache loader is required")
	}
	c.mu.Lock()
	if entry, ok := c.entries[key]; ok && !time.Now().After(entry.expiresAt) {
		c.mu.Unlock()
		return entry.value, nil
	}
	delete(c.entries, key)
	if call, ok := c.loads[key]; ok {
		c.mu.Unlock()
		<-call.done
		return call.value, call.err
	}
	call := &loadCall[T]{done: make(chan struct{})}
	c.loads[key] = call
	generation := c.generation
	c.mu.Unlock()

	value, err := loader()
	c.mu.Lock()
	call.value, call.err = value, err
	if err == nil && c.generation == generation {
		c.setLocked(key, value)
	}
	delete(c.loads, key)
	close(call.done)
	c.mu.Unlock()
	return value, err
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
	c.setLocked(key, value)
}

func (c *Cache[T]) setLocked(key string, value T) {
	if _, exists := c.entries[key]; !exists && len(c.entries) >= c.maxEntries {
		for oldestKey := range c.entries {
			delete(c.entries, oldestKey)
			break
		}
	}
	c.entries[key] = item[T]{value: value, expiresAt: time.Now().Add(c.ttl)}
}

func (c *Cache[T]) Generation() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generation
}

// SetIfGeneration prevents a slow read started before invalidation from
// refilling the cache with a stale public projection.
func (c *Cache[T]) SetIfGeneration(key string, value T, generation uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.generation != generation {
		return false
	}
	c.setLocked(key, value)
	return true
}

func (c *Cache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]item[T])
	c.generation++
}
