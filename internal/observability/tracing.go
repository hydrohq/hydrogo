package observability

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ardaozyildirim/hydrogo/internal/security"
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
		ctx, span := tracer.Start(r.Context(), r.Method+" "+boundedPath(r.URL.Path), trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		span.SetAttributes(attribute.String("http.request.method", r.Method), attribute.String("url.path", boundedPath(r.URL.Path)))
		if requestID := security.RequestIDFromContext(ctx); requestID != "" {
			span.SetAttributes(attribute.String("hydro.request_id", requestID))
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
