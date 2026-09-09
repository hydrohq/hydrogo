package security

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestWriteRateLimitedResponseSetsRetryAfter(t *testing.T) {
	response := httptest.NewRecorder()
	WriteRateLimitedResponse(response, "too many", 1500*time.Millisecond)
	if response.Code != 429 {
		t.Fatalf("status = %d", response.Code)
	}
	if got := response.Header().Get("Retry-After"); got != "2" {
		t.Fatalf("Retry-After = %q, want 2", got)
	}
}

func TestWriteRateLimitedResponseClampsShortDuration(t *testing.T) {
	response := httptest.NewRecorder()
	WriteRateLimitedResponse(response, "too many", 0)
	if got := response.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("Retry-After = %q, want 1", got)
	}
}
