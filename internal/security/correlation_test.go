package security

import (
	"context"
	"testing"
)

func TestJobCorrelationContextIsNilSafe(t *testing.T) {
	if JobIDFromContext(nil) != "" {
		t.Fatal("nil context unexpectedly contained a job id")
	}
	if got := JobIDFromContext(WithJobID(nil, "job-1")); got != "job-1" {
		t.Fatalf("job id = %q", got)
	}
	if got := JobIDFromContext(context.Background()); got != "" {
		t.Fatalf("background context job id = %q", got)
	}
}
