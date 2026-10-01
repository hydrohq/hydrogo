//go:build !hydro_no_redis

package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter is an optional distributed limiter. Redis/Valkey is never
// required by Compact; callers select it only when a shared URL is configured.
type RedisRateLimiter struct {
	Client    *redis.Client
	Prefix    string
	Limit     int
	Window    time.Duration
	Extractor interface {
		Extract(*http.Request) string
	}
}

func NewRedisClient(ctx context.Context, rawURL string) (*redis.Client, error) {
	if rawURL == "" {
		return nil, errors.New("redis URL is required")
	}
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(options)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func (l RedisRateLimiter) Allow(key string) bool {
	return l.AllowContext(context.Background(), key)
}

// AllowContext evaluates a limit using the caller's cancellation and deadline.
// This prevents an abandoned request from keeping a Redis operation alive
// until the fixed safety timeout.
func (l RedisRateLimiter) AllowContext(parent context.Context, key string) bool {
	if l.Client == nil || key == "" || l.Limit < 1 || l.Window <= 0 {
		return false
	}
	if parent == nil {
		parent = context.Background()
	}
	digest := sha256.Sum256([]byte(key))
	redisKey := l.Prefix + hex.EncodeToString(digest[:])
	const script = `local n = redis.call('INCR', KEYS[1]); if n == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end; return n <= tonumber(ARGV[2])`
	ctx, cancel := context.WithTimeout(parent, 250*time.Millisecond)
	defer cancel()
	result, err := l.Client.Eval(ctx, script, []string{redisKey}, l.Window.Milliseconds(), l.Limit).Bool()
	return err == nil && result
}

func (l RedisRateLimiter) Close() error {
	if l.Client == nil {
		return nil
	}
	return l.Client.Close()
}
