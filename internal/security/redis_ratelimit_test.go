package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRedisRateLimiterFailsClosedForInvalidInputs(t *testing.T) {
	cases := []RedisRateLimiter{
		{},
		{Limit: 1, Window: time.Minute},
		{Limit: 0, Window: time.Minute},
		{Limit: 1, Window: 0},
	}
	for i, limiter := range cases {
		if limiter.Allow("key") {
			t.Errorf("case %d allowed without a usable Redis client", i)
		}
	}
	limiter := RedisRateLimiter{Limit: 1, Window: time.Minute}
	if limiter.Allow("") {
		t.Fatal("empty rate-limit key was accepted")
	}
}

func TestNewRedisClientRejectsMissingOrMalformedURL(t *testing.T) {
	if _, err := NewRedisClient(context.Background(), ""); err == nil {
		t.Fatal("empty Redis URL accepted")
	}
	if _, err := NewRedisClient(context.Background(), "not-a-redis-url"); err == nil {
		t.Fatal("malformed Redis URL accepted")
	}
}

func TestRedisRateLimiterAtomicLimitAndTTL(t *testing.T) {
	rawURL := strings.TrimSpace(os.Getenv("HYDRO_TEST_REDIS_URL"))
	if rawURL == "" {
		t.Skip("HYDRO_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	client, err := NewRedisClient(ctx, rawURL)
	if err != nil {
		t.Fatalf("connect to Redis: %v", err)
	}
	defer client.Close()

	key := "integration-" + time.Now().UTC().Format("20060102150405.000000000")
	prefix := "hydro:test:" + key + ":"
	limiter := RedisRateLimiter{Client: client, Prefix: prefix, Limit: 1, Window: 30 * time.Second}
	if !limiter.Allow(key) {
		t.Fatal("first request should be allowed")
	}
	if limiter.Allow(key) {
		t.Fatal("second request exceeded the configured limit")
	}

	digest := sha256.Sum256([]byte(key))
	redisKey := prefix + hex.EncodeToString(digest[:])
	ttl, err := client.TTL(ctx, redisKey).Result()
	if err != nil {
		t.Fatalf("read limiter TTL: %v", err)
	}
	if ttl <= 0 || ttl > 30*time.Second {
		t.Fatalf("unexpected limiter TTL %s", ttl)
	}
	if err := client.Del(ctx, redisKey).Err(); err != nil {
		t.Fatalf("cleanup limiter key: %v", err)
	}
}
