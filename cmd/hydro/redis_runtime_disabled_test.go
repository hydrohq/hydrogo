//go:build hydro_no_redis

package main

import (
	"context"
	"testing"
	"time"
)

func TestNoRedisRuntimeKeepsBoundedLocalLimiter(t *testing.T) {
	runtime, err := newRedisRuntime(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	limiter := newRuntimeRateLimiter(runtime, "", 1, time.Minute, false)
	if !limiter.Allow("request") || limiter.Allow("request") {
		t.Fatal("local fallback limiter did not enforce its configured bound")
	}
	if _, err := newRedisRuntime(context.Background(), "redis://127.0.0.1:6379/0"); err == nil {
		t.Fatal("no-Redis build accepted a configured Redis endpoint")
	}
}
