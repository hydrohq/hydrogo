//go:build hydro_no_otlp

package observability

import (
	"context"
	"errors"
	"testing"
)

func TestNoOTLPBuildFailsClosedWhenTracingRequested(t *testing.T) {
	if _, err := InstallOTLPProvider(context.Background(), TracingProfile{Endpoint: "https://collector.example.test", Enabled: true}); !errors.Is(err, ErrOTLPUnavailable) {
		t.Fatalf("OTLP-enabled profile error = %v, want ErrOTLPUnavailable", err)
	}
}

func TestNoOTLPBuildAcceptsDisabledTracing(t *testing.T) {
	shutdown, err := InstallOTLPProvider(context.Background(), TracingProfile{})
	if err != nil || shutdown == nil {
		t.Fatalf("disabled tracing shutdown_present=%t err=%v", shutdown != nil, err)
	}
}
