//go:build hydro_no_otlp

package observability

import (
	"context"
	"fmt"
)

func InstallOTLPProvider(_ context.Context, profile TracingProfile) (func(context.Context) error, error) {
	if profile.Enabled || profile.Endpoint != "" {
		return nil, fmt.Errorf("%w: this Edition was built without the OTLP exporter", ErrOTLPUnavailable)
	}
	return func(context.Context) error { return nil }, nil
}
