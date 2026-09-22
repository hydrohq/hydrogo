//go:build !hydro_no_otlp

package observability

import (
	"context"
	"testing"
)

func TestOTLPBuildAcceptsDisabledTracing(t *testing.T) {
	shutdown, err := InstallOTLPProvider(context.Background(), TracingProfile{})
	if err != nil || shutdown == nil {
		t.Fatalf("disabled tracing shutdown_present=%t err=%v", shutdown != nil, err)
	}
}
