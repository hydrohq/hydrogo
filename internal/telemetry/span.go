// Package telemetry contains dependency-light tracing primitives usable by
// low-level packages without importing HTTP/auth observability middleware.
package telemetry

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func StartSpan(ctx context.Context, component, operation string) (context.Context, trace.Span) {
	return otel.Tracer("hydrogo/"+component).Start(ctx, operation)
}
