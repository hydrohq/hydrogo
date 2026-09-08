package security

import (
	"testing"
	"time"
)

func TestAttemptLimiterBoundsAttemptsAndMemory(t *testing.T) {
	limiter := NewAttemptLimiter(1, 2, time.Hour)
	if !limiter.Allow("first") || !limiter.Allow("first") || limiter.Allow("first") {
		t.Fatal("attempt limit was not enforced")
	}
	if !limiter.Allow("second") {
		t.Fatal("bounded limiter rejected a fresh key")
	}
}
