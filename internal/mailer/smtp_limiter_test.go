package mailer

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTokenBucketReservesBurstAndPacesConcurrentWork(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bucket := newTokenBucket(2, 2)
	for i := 0; i < 2; i++ {
		if wait := bucket.reserve(start); wait != 0 {
			t.Fatalf("burst reservation %d waited %s", i, wait)
		}
	}
	if wait := bucket.reserve(start); wait != 500*time.Millisecond {
		t.Fatalf("third send wait = %s, want 500ms", wait)
	}
	if wait := bucket.reserve(start); wait != time.Second {
		t.Fatalf("fourth send wait = %s, want 1s", wait)
	}
}

func TestHourlyBucketSpacesSendQuota(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bucket := newTokenBucket(100.0/3600, 1)
	if wait := bucket.reserve(start); wait != 0 {
		t.Fatalf("first send wait = %s", wait)
	}
	if wait := bucket.reserve(start); wait != 36*time.Second {
		t.Fatalf("hourly quota wait = %s, want 36s", wait)
	}
}

func TestSendLimiterHonorsCancellationWhileWaiting(t *testing.T) {
	limiter := sendLimiter{perSecond: newTokenBucket(1, 1), perHour: newTokenBucket(1, 1)}
	if err := limiter.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := limiter.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled wait = %v", err)
	}
}
