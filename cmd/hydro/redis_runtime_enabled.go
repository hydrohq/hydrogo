//go:build !hydro_no_redis

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/news"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	"github.com/redis/go-redis/v9"
)

type redisRuntime struct{ client *redis.Client }

func newRedisRuntime(ctx context.Context, rawURL string) (redisRuntime, error) {
	if rawURL == "" {
		return redisRuntime{}, nil
	}
	client, err := security.NewRedisClient(ctx, rawURL)
	if err != nil {
		return redisRuntime{}, err
	}
	return redisRuntime{client: client}, nil
}

func (r redisRuntime) Close() error {
	if r.client == nil {
		return nil
	}
	return r.client.Close()
}

func newRuntimeRateLimiter(r redisRuntime, keyPrefix string, limit int, window time.Duration, requireTenantScope bool) security.RateLimiter {
	var limiter security.RateLimiter
	if r.client != nil {
		limiter = security.RedisRateLimiter{Client: r.client, Prefix: keyPrefix + "hydro:ratelimit:", Limit: limit, Window: window}
	} else {
		limiter = security.NewAttemptLimiter(10000, limit, window)
	}
	if requireTenantScope {
		return security.TenantScopedRateLimiter{Inner: limiter}
	}
	return limiter
}

func newPageCache(enabled bool, budget capacity.Budget, r redisRuntime, prefix string) (pageCache, error) {
	if !enabled {
		return nil, nil
	}
	local := cache.New[pages.Page](budget.CacheEntries, budget.CacheTTL)
	if r.client == nil {
		return local, nil
	}
	remote := cache.RedisJSONRemote[pages.Page]{Client: r.client, Prefix: prefix + "hydro:cache:page:", TTL: budget.CacheTTL}
	twoTier, err := cache.NewTwoTier(local, remote)
	if err != nil {
		return nil, fmt.Errorf("configure two-tier page cache: %w", err)
	}
	return twoTier, nil
}

func newNewsCache(enabled bool, budget capacity.Budget, r redisRuntime, prefix string) (newsCache, error) {
	if !enabled {
		return nil, nil
	}
	local := cache.New[news.Item](budget.CacheEntries, budget.CacheTTL)
	if r.client == nil {
		return local, nil
	}
	remote := cache.RedisJSONRemote[news.Item]{Client: r.client, Prefix: prefix + "hydro:cache:news:", TTL: budget.CacheTTL}
	twoTier, err := cache.NewTwoTier(local, remote)
	if err != nil {
		return nil, fmt.Errorf("configure two-tier news cache: %w", err)
	}
	return twoTier, nil
}

func configureRealtimeHub(hub *realtime.Hub, r redisRuntime, keyPrefix string) error {
	if r.client != nil {
		bus := realtime.NewRedisMessageBus(r.client, keyPrefix)
		return hub.SetBus(bus)
	}
	return nil
}
