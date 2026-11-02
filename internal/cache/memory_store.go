package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MemoryItem represents a cached item with an expiration time.
type MemoryItem struct {
	Value     any
	ExpiresAt time.Time // Zero value means item never expires
}

// MemoryStoreOptions configures the in-memory cache engine.
type MemoryStoreOptions struct {
	MaxEntries      int
	JanitorInterval time.Duration
}

// MemoryStore is an embedded, thread-safe, pure Go in-memory key-value store
// with per-key TTL, background janitor cleanup, atomic counters, and pub/sub.
type MemoryStore struct {
	mu              sync.RWMutex
	items           map[string]*MemoryItem
	maxEntries      int
	janitorInterval time.Duration
	stopJanitor     chan struct{}
	janitorRunning  bool
	closed          bool

	hits        atomic.Uint64
	misses      atomic.Uint64
	evictions   atomic.Uint64
	expirations atomic.Uint64
}

// NewMemoryStore initializes a new MemoryStore with a background janitor.
func NewMemoryStore(opts MemoryStoreOptions) *MemoryStore {
	maxEntries := opts.MaxEntries
	if maxEntries < 1 {
		maxEntries = 10000
	}
	janitorInterval := opts.JanitorInterval
	if janitorInterval <= 0 {
		janitorInterval = time.Minute
	}

	s := &MemoryStore{
		items:           make(map[string]*MemoryItem, 128),
		maxEntries:      maxEntries,
		janitorInterval: janitorInterval,
		stopJanitor:     make(chan struct{}),
	}
	s.startJanitor()
	return s
}

func (s *MemoryStore) startJanitor() {
	s.mu.Lock()
	if s.janitorRunning || s.closed {
		s.mu.Unlock()
		return
	}
	s.janitorRunning = true
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(s.janitorInterval)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopJanitor:
				return
			case now := <-ticker.C:
				s.deleteExpired(now)
			}
		}
	}()
}

// deleteExpired scans and purges all expired keys.
func (s *MemoryStore) deleteExpired(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	purged := 0
	for k, item := range s.items {
		if !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
			delete(s.items, k)
			s.expirations.Add(1)
			purged++
		}
	}
	return purged
}

// Set stores a key-value pair with an optional TTL. If ttl <= 0, the item never expires.
func (s *MemoryStore) Set(key string, value any, ttl time.Duration) {
	if s == nil || key == "" {
		return
	}
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	// Evict if at capacity and key does not exist
	if _, exists := s.items[key]; !exists && len(s.items) >= s.maxEntries {
		now := time.Now()
		evicted := false
		// First try evicting an already expired item
		for k, item := range s.items {
			if !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
				delete(s.items, k)
				s.expirations.Add(1)
				evicted = true
				break
			}
		}
		// If none expired, evict an arbitrary oldest key
		if !evicted {
			for k := range s.items {
				delete(s.items, k)
				s.evictions.Add(1)
				break
			}
		}
	}

	s.items[key] = &MemoryItem{
		Value:     value,
		ExpiresAt: expiresAt,
	}
}

// Get retrieves a value by key. Returns (nil, false) if key does not exist or is expired.
func (s *MemoryStore) Get(key string) (any, bool) {
	if s == nil || key == "" {
		return nil, false
	}

	s.mu.RLock()
	item, ok := s.items[key]
	if !ok {
		s.mu.RUnlock()
		s.misses.Add(1)
		return nil, false
	}

	now := time.Now()
	if !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
		s.mu.RUnlock()
		s.misses.Add(1)
		// Active cleanup on read
		s.Delete(key)
		return nil, false
	}

	val := item.Value
	s.mu.RUnlock()
	s.hits.Add(1)
	return val, true
}

// GetString retrieves a string value by key.
func (s *MemoryStore) GetString(key string) (string, bool) {
	val, ok := s.Get(key)
	if !ok || val == nil {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetInt64 retrieves an int64 value by key.
func (s *MemoryStore) GetInt64(key string) (int64, bool) {
	val, ok := s.Get(key)
	if !ok || val == nil {
		return 0, false
	}
	switch v := val.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case uint64:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

// GetBool retrieves a boolean value by key.
func (s *MemoryStore) GetBool(key string) (bool, bool) {
	val, ok := s.Get(key)
	if !ok || val == nil {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

// Delete removes a key from the store. Returns true if key was present.
func (s *MemoryStore) Delete(key string) bool {
	if s == nil || key == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[key]; ok {
		delete(s.items, key)
		return true
	}
	return false
}

// Has checks if an active key exists without updating hit/miss statistics.
func (s *MemoryStore) Has(key string) bool {
	if s == nil || key == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[key]
	if !ok {
		return false
	}
	return item.ExpiresAt.IsZero() || time.Now().Before(item.ExpiresAt)
}

// Clear flushes all entries in the store.
func (s *MemoryStore) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[string]*MemoryItem, 128)
}

// Len returns the current number of cached items (including uncollected expired items).
func (s *MemoryStore) Len() int {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// Incr atomically increments the integer value of key by delta.
// If the key does not exist or has expired, it is initialized to delta with the given ttl.
func (s *MemoryStore) Incr(key string, delta int64, ttl time.Duration) (int64, error) {
	if s == nil || key == "" {
		return 0, fmt.Errorf("invalid memory store or key")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, fmt.Errorf("memory store is closed")
	}

	now := time.Now()
	item, ok := s.items[key]
	if !ok || (!item.ExpiresAt.IsZero() && now.After(item.ExpiresAt)) {
		var expiresAt time.Time
		if ttl > 0 {
			expiresAt = now.Add(ttl)
		}
		s.items[key] = &MemoryItem{
			Value:     delta,
			ExpiresAt: expiresAt,
		}
		return delta, nil
	}

	var current int64
	switch v := item.Value.(type) {
	case int64:
		current = v
	case int:
		current = int64(v)
	default:
		return 0, fmt.Errorf("key %q value is not an integer", key)
	}

	newVal := current + delta
	item.Value = newVal
	// If item had no expiration but a ttl is supplied, attach it
	if item.ExpiresAt.IsZero() && ttl > 0 {
		item.ExpiresAt = now.Add(ttl)
	}
	return newVal, nil
}

// RateLimit checks and increments a rate limit counter using a fixed window.
// Returns allowed (true if current <= limit), current count, and duration until reset.
func (s *MemoryStore) RateLimit(key string, limit int64, window time.Duration) (bool, int64, time.Duration) {
	if s == nil || key == "" || limit < 1 || window <= 0 {
		return false, 0, 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	item, ok := s.items[key]
	if !ok || (!item.ExpiresAt.IsZero() && now.After(item.ExpiresAt)) {
		if !ok && len(s.items) >= s.maxEntries {
			evicted := false
			for k, it := range s.items {
				if !it.ExpiresAt.IsZero() && now.After(it.ExpiresAt) {
					delete(s.items, k)
					s.expirations.Add(1)
					evicted = true
					break
				}
			}
			if !evicted {
				for k := range s.items {
					delete(s.items, k)
					s.evictions.Add(1)
					break
				}
			}
		}
		expiresAt := now.Add(window)
		s.items[key] = &MemoryItem{
			Value:     int64(1),
			ExpiresAt: expiresAt,
		}
		return true, 1, window
	}

	var current int64
	switch v := item.Value.(type) {
	case int64:
		current = v
	case int:
		current = int64(v)
	default:
		current = 0
	}

	current++
	item.Value = current

	resetIn := time.Duration(0)
	if !item.ExpiresAt.IsZero() {
		resetIn = time.Until(item.ExpiresAt)
		if resetIn < 0 {
			resetIn = 0
		}
	}

	allowed := current <= limit
	return allowed, current, resetIn
}

// Close gracefully stops the background janitor and marks the store closed.
func (s *MemoryStore) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	if s.janitorRunning {
		close(s.stopJanitor)
		s.janitorRunning = false
	}
	s.mu.Unlock()
	return nil
}

// MemoryRateLimiter implements security.RateLimiter backed by MemoryStore.
type MemoryRateLimiter struct {
	store  *MemoryStore
	limit  int64
	window time.Duration
	prefix string
}

var limiterSeq atomic.Uint64

// NewMemoryRateLimiter creates an in-memory rate limiter backed by store.
func NewMemoryRateLimiter(store *MemoryStore, prefix string, limit int, window time.Duration) *MemoryRateLimiter {
	if store == nil {
		store = NewMemoryStore(MemoryStoreOptions{MaxEntries: 10000})
	}
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	id := limiterSeq.Add(1)
	if prefix == "" {
		prefix = fmt.Sprintf("rl:%d:", id)
	} else {
		prefix = fmt.Sprintf("%s%d:", prefix, id)
	}
	return &MemoryRateLimiter{
		store:  store,
		limit:  int64(limit),
		window: window,
		prefix: prefix,
	}
}

// Allow reports whether an action under key is permitted by the rate limit.
func (r *MemoryRateLimiter) Allow(key string) bool {
	if r == nil || r.store == nil || key == "" {
		return false
	}
	keyDigest := sha256.Sum256([]byte(key))
	cacheKey := r.prefix + hex.EncodeToString(keyDigest[:])
	allowed, _, _ := r.store.RateLimit(cacheKey, r.limit, r.window)
	return allowed
}

// AllowContext reports whether an action under key is permitted, accepting context.
func (r *MemoryRateLimiter) AllowContext(ctx context.Context, key string) bool {
	if ctx != nil && ctx.Err() != nil {
		return false
	}
	return r.Allow(key)
}

// MemoryPubSub is a lightweight, zero-dependency in-process pub/sub event bus.
type MemoryPubSub struct {
	mu          sync.RWMutex
	subscribers map[string]map[uint64]chan any
	nextID      atomic.Uint64
}

// NewMemoryPubSub initializes an in-memory pub/sub broker.
func NewMemoryPubSub() *MemoryPubSub {
	return &MemoryPubSub{
		subscribers: make(map[string]map[uint64]chan any),
	}
}

// Subscribe listens to a topic and returns a receive channel along with an unsubscribe func.
func (ps *MemoryPubSub) Subscribe(topic string, bufferSize int) (<-chan any, func()) {
	if bufferSize < 1 {
		bufferSize = 32
	}
	ch := make(chan any, bufferSize)
	id := ps.nextID.Add(1)

	ps.mu.Lock()
	if ps.subscribers[topic] == nil {
		ps.subscribers[topic] = make(map[uint64]chan any)
	}
	ps.subscribers[topic][id] = ch
	ps.mu.Unlock()

	unsubscribe := func() {
		ps.mu.Lock()
		defer ps.mu.Unlock()
		if subs, ok := ps.subscribers[topic]; ok {
			if existing, found := subs[id]; found {
				delete(subs, id)
				close(existing)
			}
			if len(subs) == 0 {
				delete(ps.subscribers, topic)
			}
		}
	}

	return ch, unsubscribe
}

// Publish non-blockingly fans out payload to all topic subscribers.
// Returns the number of active subscribers who received the message.
func (ps *MemoryPubSub) Publish(topic string, payload any) int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	subs, ok := ps.subscribers[topic]
	if !ok || len(subs) == 0 {
		return 0
	}

	delivered := 0
	for _, ch := range subs {
		select {
		case ch <- payload:
			delivered++
		default:
			// Non-blocking drop if consumer buffer is full to prevent pipeline stall
		}
	}
	return delivered
}

// MemoryWorkerPool is an in-memory concurrent worker pool for async jobs.
type MemoryWorkerPool struct {
	jobs    chan func(context.Context)
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	stopped atomic.Bool
}

// NewMemoryWorkerPool creates a worker pool with workers count and queue capacity.
func NewMemoryWorkerPool(workers, queueSize int) *MemoryWorkerPool {
	if workers < 1 {
		workers = 1
	}
	if queueSize < 1 {
		queueSize = 100
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &MemoryWorkerPool{
		jobs:   make(chan func(context.Context), queueSize),
		ctx:    ctx,
		cancel: cancel,
	}

	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-p.ctx.Done():
					return
				case job, ok := <-p.jobs:
					if !ok {
						return
					}
					if job != nil {
						job(p.ctx)
					}
				}
			}
		}()
	}

	return p
}

// Submit enqueues a job. Returns false if the pool is stopped or buffer is full.
func (p *MemoryWorkerPool) Submit(job func(context.Context)) bool {
	if p == nil || p.stopped.Load() || job == nil {
		return false
	}
	select {
	case p.jobs <- job:
		return true
	default:
		return false
	}
}

// Stop gracefully waits for all queued jobs to finish or cancels on context expiration.
func (p *MemoryWorkerPool) Stop() {
	if p == nil || p.stopped.Swap(true) {
		return
	}
	close(p.jobs)
	p.wg.Wait()
	p.cancel()
}
