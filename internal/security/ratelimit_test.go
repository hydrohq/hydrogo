package security

import (
	"context"
	"errors"
	"testing"
	"time"
)

type distributedStore struct {
	ok  bool
	err error
}

func (s distributedStore) Allow(context.Context, string) (bool, error) { return s.ok, s.err }

func TestDistributedLimiterFailsClosedOnStoreFailure(t *testing.T) {
	if (DistributedLimiter{Store: distributedStore{ok: true}}).Allow(context.Background(), "login") != true {
		t.Fatal("available shared store rejected request")
	}
	if (DistributedLimiter{Store: distributedStore{ok: true, err: errors.New("backend down")}}).Allow(context.Background(), "login") {
		t.Fatal("shared store failure opened limiter")
	}
	if (DistributedLimiter{}).Allow(context.Background(), "login") {
		t.Fatal("missing shared store opened limiter")
	}
}

func TestAttemptLimiterBoundsAttemptsAndMemory(t *testing.T) {
	limiter := NewAttemptLimiter(1, 2, time.Hour)
	if !limiter.Allow("first") || !limiter.Allow("first") || limiter.Allow("first") {
		t.Fatal("attempt limit was not enforced")
	}
	if !limiter.Allow("second") {
		t.Fatal("bounded limiter rejected a fresh key")
	}
}

func TestAttemptLimiterRejectsEmptyKeysAndEvictsExpiredEntriesFirst(t *testing.T) {
	limiter := NewAttemptLimiter(2, 1, 5*time.Millisecond)
	if limiter.Allow("") {
		t.Fatal("empty limiter key was accepted")
	}
	if !limiter.Allow("expired") || !limiter.Allow("active") {
		t.Fatal("initial limiter keys were rejected")
	}
	time.Sleep(10 * time.Millisecond)
	if !limiter.Allow("fresh") {
		t.Fatal("fresh key was rejected after expired entry cleanup")
	}
	if _, ok := limiter.entries["expired"]; ok {
		t.Fatal("expired entry was not evicted")
	}
}
