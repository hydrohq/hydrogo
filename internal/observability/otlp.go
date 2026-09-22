package observability

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// InstallOTLPProvider installs an optional HTTP OTLP exporter selected by the
// deployment profile. Compact callers can pass a disabled profile and receive
// a no-op shutdown function with no collector dependency.
func InstallOTLPProvider(ctx context.Context, profile TracingProfile) (func(context.Context) error, error) {
	if !profile.Enabled || profile.Endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}
	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(profile.Endpoint))
	if err != nil {
		return nil, err
	}
	provider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(profile.Sampling))))
	otel.SetTracerProvider(provider)
	return provider.Shutdown, nil
}

var ErrOTLPUnavailable = errors.New("otlp exporter unavailable")
