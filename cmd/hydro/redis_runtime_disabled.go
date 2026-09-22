//go:build hydro_no_redis

package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

type redisRuntime struct{}

func newRedisRuntime(_ context.Context, rawURL string) (redisRuntime, error) {
	if strings.TrimSpace(rawURL) != "" {
		return redisRuntime{}, errors.New("redis is not included in this Edition build")
	}
	return redisRuntime{}, nil
}

func (redisRuntime) Close() error { return nil }

func newRuntimeRateLimiter(_ redisRuntime, _ string, limit int, window time.Duration, requireTenantScope bool) security.RateLimiter {
	var limiter security.RateLimiter = security.NewAttemptLimiter(10000, limit, window)
	if requireTenantScope {
		return security.TenantScopedRateLimiter{Inner: limiter}
	}
	return limiter
}

func newPageCache(enabled bool, budget capacity.Budget, _ redisRuntime, _ string) (pageCache, error) {
	if !enabled {
		return nil, nil
	}
	return cache.New[pages.Page](budget.CacheEntries, budget.CacheTTL), nil
}
