package observability

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// TracingMiddleware uses the global OpenTelemetry provider. With no exporter
// configured it is a no-op, keeping Compact free from mandatory telemetry
// infrastructure while preserving trace-context propagation for other
// profiles. It deliberately records neither query/body/cookie data nor actor
// and tenant identifiers, which would create sensitive/high-cardinality data.
func TracingMiddleware(next http.Handler) http.Handler {
	if next == nil {
		return http.NotFoundHandler()
	}
	tracer := otel.Tracer("hydrogo/http")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		ctx, span := tracer.Start(r.Context(), r.Method+" "+boundedPath(r.URL.Path), trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		span.SetAttributes(attribute.String("http.request.method", r.Method), attribute.String("url.path", boundedPath(r.URL.Path)), attribute.String("hydro.route", NormalizeRoute(r.URL.Path)))
		if requestID := security.RequestIDFromContext(ctx); requestID != "" {
			span.SetAttributes(attribute.String("hydro.request_id", requestID))
		}
		if scope, err := tenant.FromContext(ctx); err == nil {
			span.SetAttributes(attribute.String("hydro.tenant_id", scope.TenantID))
		}
		next.ServeHTTP(recorder, r.WithContext(ctx))
		span.SetAttributes(attribute.Int("http.response.status_code", recorder.status), attribute.Int64("http.response.body.size", int64(recorder.bytes)), attribute.Int64("hydro.duration_ms", time.Since(started).Milliseconds()))
	})
}

// StartComponentSpan gives repositories, cache adapters, workers, webhooks
// and backup operators one provider-neutral span boundary. No exporter is
// required; the active global provider decides whether it is recorded.
func StartComponentSpan(ctx context.Context, component, operation string) (context.Context, trace.Span) {
	return otel.Tracer("hydrogo/"+boundedPath(component)).Start(ctx, operation)
}
