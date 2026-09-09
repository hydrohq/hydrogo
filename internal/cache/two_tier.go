package cache

import "errors"

import "github.com/ardaozyildirim/hydrogo/internal/guards"

type RemoteStore[T any] interface {
	Get(string) (T, bool, error)
	Set(string, T) error
}

// TwoTier combines bounded local L1 with an optional remote L2. L2 failures
// fall back to the local loader and never make the request path unbounded.
type TwoTier[T any] struct {
	local  *Cache[T]
	remote RemoteStore[T]
}

func NewTwoTier[T any](local *Cache[T], remote RemoteStore[T]) (*TwoTier[T], error) {
	if local == nil {
		return nil, errors.New("local cache is required")
	}
	return &TwoTier[T]{local: local, remote: remote}, nil
}

func (c *TwoTier[T]) GetOrLoad(key string, loader func() (T, error)) (T, error) {
	if value, ok := c.local.Get(key); ok {
		return value, nil
	}
	if !guards.IsNil(c.remote) {
		if value, ok, err := c.remote.Get(key); err == nil && ok {
			c.local.Set(key, value)
			return value, nil
		}
	}
	value, err := c.local.GetOrLoad(key, loader)
	if err == nil && !guards.IsNil(c.remote) {
		_ = c.remote.Set(key, value)
	}
	return value, err
}
