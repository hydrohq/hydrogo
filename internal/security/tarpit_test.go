package security

import (
	"context"
	"testing"
	"time"
)

func TestTarpitProgressionResetAndExpiry(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	tarpit := NewTarpit(4, time.Minute, time.Second, 8*time.Second)
	tarpit.now = func() time.Time { return now }
	for i, want := range []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 8 * time.Second} {
		if got := tarpit.Failure("ip:account"); got != want {
			t.Fatalf("failure %d delay = %v, want %v", i+1, got, want)
		}
	}
	tarpit.Reset("ip:account")
	if got := tarpit.Failure("ip:account"); got != time.Second {
		t.Fatalf("reset delay = %v", got)
	}
	now = now.Add(time.Minute)
	if got := tarpit.Failure("ip:account"); got != time.Second {
		t.Fatalf("expired delay = %v", got)
	}
}

func TestTarpitWaitHonorsCancellationAndBoundsEntries(t *testing.T) {
	tarpit := NewTarpit(1, time.Minute, time.Hour, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := tarpit.Wait(ctx, "first"); err != context.Canceled {
		t.Fatalf("cancelled wait = %v", err)
	}
	tarpit.Failure("second")
	if len(tarpit.entries) > 1 {
		t.Fatalf("failure bucket count = %d", len(tarpit.entries))
	}
}
