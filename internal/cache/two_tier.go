package cache

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

var ErrTwoTierUnavailable = errors.New("two-tier cache is unavailable")

type RemoteStore[T any] interface {
	Get(string) (T, bool, error)
	Set(string, T) error
}

// TwoTier combines bounded local L1 with an optional remote L2. L2 failures
// fall back to the local loader and never make the request path unbounded.
type TwoTier[T any] struct {
	local      *Cache[T]
	remote     RemoteStore[T]
	generation atomic.Uint64
}

func NewTwoTier[T any](local *Cache[T], remote RemoteStore[T]) (*TwoTier[T], error) {
	if local == nil {
		return nil, errors.New("local cache is required")
	}
	return &TwoTier[T]{local: local, remote: remote}, nil
}

func (c *TwoTier[T]) remoteKey(generation uint64, key string) string {
	return fmt.Sprintf("g%d:%s", generation, key)
}

func (c *TwoTier[T]) GetOrLoad(key string, loader func() (T, error)) (T, error) {
	if c == nil || c.local == nil {
		var zero T
		return zero, ErrTwoTierUnavailable
	}
	if value, ok := c.local.Get(key); ok {
		return value, nil
	}
	generation := c.generation.Load()
	if !guards.IsNil(c.remote) {
		if value, ok, err := c.remote.Get(c.remoteKey(generation, key)); err == nil && ok {
			c.local.Set(key, value)
			return value, nil
		}
	}
	value, err := c.local.GetOrLoad(key, loader)
	if err == nil && !guards.IsNil(c.remote) {
		_ = c.remote.Set(c.remoteKey(generation, key), value)
	}
	return value, err
}

// GetOrLoadScoped requires an explicit tenant scope and applies it to both L1
// and L2 keys, preventing cross-tenant cache hits in shared deployments.
func (c *TwoTier[T]) GetOrLoadScoped(ctx context.Context, module, key string, loader func() (T, error)) (T, error) {
	ctx, span := observability.StartComponentSpan(ctx, "cache", "get_or_load")
	defer span.End()
	scopedKey, err := tenant.CacheKey(ctx, module, key)
	if err != nil {
		var zero T
		return zero, err
	}
	return c.GetOrLoad(scopedKey, loader)
}

func (c *TwoTier[T]) GetOrLoadAtGeneration(key string, generation uint64, loader func() (T, error)) (T, error) {
	if c == nil || c.local == nil {
		var zero T
		return zero, ErrTwoTierUnavailable
	}
	if value, ok := c.local.Get(key); ok {
		return value, nil
	}
	if !guards.IsNil(c.remote) {
		if value, ok, err := c.remote.Get(c.remoteKey(generation, key)); err == nil && ok {
			c.local.SetIfGeneration(key, value, generation)
			return value, nil
		}
	}
	value, err := c.local.GetOrLoadAtGeneration(key, generation, loader)
	if err == nil && !guards.IsNil(c.remote) {
		_ = c.remote.Set(c.remoteKey(generation, key), value)
	}
	return value, err
}

func (c *TwoTier[T]) Clear() {
	if c == nil || c.local == nil {
		return
	}
	c.local.Clear()
	c.generation.Store(c.local.Generation())
}

func (c *TwoTier[T]) Stats() Stats {
	if c == nil || c.local == nil {
		return Stats{}
	}
	return c.local.Stats()
}
