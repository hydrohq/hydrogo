package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrCacheUnavailable = errors.New("cache is unavailable")
	ErrCacheLoaderPanic = errors.New("cache loader panicked")
)

type generationContextKey struct{}

// WithGeneration binds a request to the deployment/cache generation selected
// before traffic was handed to its handler.
func WithGeneration(ctx context.Context, generation uint64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, generationContextKey{}, generation)
}

// GenerationFromContext returns the generation captured at request ingress.
func GenerationFromContext(ctx context.Context) (uint64, bool) {
	if ctx == nil {
		return 0, false
	}
	generation, ok := ctx.Value(generationContextKey{}).(uint64)
	return generation, ok
}

type item[T any] struct {
	value     T
	expiresAt time.Time
}

type loadCall[T any] struct {
	done       chan struct{}
	value      T
	err        error
	generation uint64
}

// Cache is a bounded, process-local cache. It never grows beyond maxEntries.
type Cache[T any] struct {
	mu          sync.Mutex
	entries     map[string]item[T]
	maxEntries  int
	ttl         time.Duration
	generation  uint64
	loads       map[string]*loadCall[T]
	hits        atomic.Uint64
	misses      atomic.Uint64
	loadsTotal  atomic.Uint64
	loadErrors  atomic.Uint64
	evictions   atomic.Uint64
	expirations atomic.Uint64
}

type Stats struct {
	Hits        uint64
	Misses      uint64
	Loads       uint64
	LoadErrors  uint64
	Evictions   uint64
	Expirations uint64
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
	if !c.ready() {
		var zero T
		return zero, ErrCacheUnavailable
	}
	c.mu.Lock()
	generation := c.generation
	c.mu.Unlock()
	return c.GetOrLoadAtGeneration(key, generation, loader)
}

// GetOrLoadAtGeneration coalesces a load for a request that entered during a
// specific cache generation. A result from an older generation may be
// returned to that request, but it can never refill the current cache.
func (c *Cache[T]) GetOrLoadAtGeneration(key string, generation uint64, loader func() (T, error)) (T, error) {
	if !c.ready() {
		var zero T
		return zero, ErrCacheUnavailable
	}
	if loader == nil {
		var zero T
		return zero, errors.New("cache loader is required")
	}
	c.mu.Lock()
	if c.generation == generation {
		if entry, ok := c.entries[key]; ok && !time.Now().After(entry.expiresAt) {
			c.hits.Add(1)
			c.mu.Unlock()
			return entry.value, nil
		}
		c.misses.Add(1)
		if _, ok := c.entries[key]; ok {
			c.expirations.Add(1)
		}
		delete(c.entries, key)
	} else {
		c.misses.Add(1)
	}
	if call, ok := c.loads[key]; ok && call.generation == generation {
		c.mu.Unlock()
		<-call.done
		return call.value, call.err
	}
	call := &loadCall[T]{done: make(chan struct{}), generation: generation}
	c.loads[key] = call
	c.mu.Unlock()

	value, err := runLoader(loader)
	c.loadsTotal.Add(1)
	if err != nil {
		c.loadErrors.Add(1)
	}
	c.mu.Lock()
	call.value, call.err = value, err
	if err == nil && c.generation == generation {
		c.setLocked(key, value)
	}
	if current, ok := c.loads[key]; ok && current == call {
		delete(c.loads, key)
	}
	close(call.done)
	c.mu.Unlock()
	return value, err
}

func (c *Cache[T]) Get(key string) (T, bool) {
	if !c.ready() {
		var zero T
		return zero, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		c.misses.Add(1)
		if ok {
			c.expirations.Add(1)
		}
		delete(c.entries, key)
		var zero T
		return zero, false
	}
	c.hits.Add(1)
	return entry.value, true
}

func (c *Cache[T]) Set(key string, value T) {
	if !c.ready() {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setLocked(key, value)
}

func (c *Cache[T]) setLocked(key string, value T) {
	if _, exists := c.entries[key]; !exists && len(c.entries) >= c.maxEntries {
		for oldestKey := range c.entries {
			delete(c.entries, oldestKey)
			c.evictions.Add(1)
			break
		}
	}
	c.entries[key] = item[T]{value: value, expiresAt: time.Now().Add(c.ttl)}
}

func (c *Cache[T]) Stats() Stats {
	if !c.ready() {
		return Stats{}
	}
	return Stats{Hits: c.hits.Load(), Misses: c.misses.Load(), Loads: c.loadsTotal.Load(), LoadErrors: c.loadErrors.Load(), Evictions: c.evictions.Load(), Expirations: c.expirations.Load()}
}

func (c *Cache[T]) Generation() uint64 {
	if !c.ready() {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generation
}

// SetIfGeneration prevents a slow read started before invalidation from
// refilling the cache with a stale public projection.
func (c *Cache[T]) SetIfGeneration(key string, value T, generation uint64) bool {
	if !c.ready() {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.generation != generation {
		return false
	}
	c.setLocked(key, value)
	return true
}

func (c *Cache[T]) Clear() {
	if !c.ready() {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]item[T])
	c.generation++
}

func (c *Cache[T]) ready() bool {
	// entries and loads are protected by mu and are replaced by Clear. Readiness
	// must therefore depend only on immutable construction state; inspecting the
	// maps here would race with an invalidation under concurrent traffic.
	return c != nil && c.maxEntries > 0
}

func runLoader[T any](loader func() (T, error)) (value T, err error) {
	defer func() {
		if recover() != nil {
			var zero T
			value = zero
			err = ErrCacheLoaderPanic
		}
	}()
	return loader()
}
