package news

import (
	"testing"
	"time"
)

func TestRetryDelayIsBoundedExponential(t *testing.T) {
	base := 10 * time.Second
	if got := retryDelay(base, 1); got != 10*time.Second {
		t.Fatalf("first retry = %s", got)
	}
	if got := retryDelay(base, 3); got != 40*time.Second {
		t.Fatalf("third retry = %s", got)
	}
	if got := retryDelay(base, 20); got != time.Hour {
		t.Fatalf("capped retry = %s", got)
	}
}
