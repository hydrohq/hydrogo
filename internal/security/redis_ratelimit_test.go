package security

import (
	"context"
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
