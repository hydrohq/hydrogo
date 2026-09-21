package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/ardaozyildirim/hydrogo/internal/security"
)

func TestTracingMiddlewarePreservesRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/public/pages?secret=no", nil)
	res := httptest.NewRecorder()
	TracingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "secret=no" {
			t.Fatal("request changed")
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d", res.Code)
	}
}

func TestTracingMiddlewareDoesNotExportRequestSecrets(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		_ = provider.Shutdown(t.Context())
		otel.SetTracerProvider(previous)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/media?access_token=secret-query", nil)
	req.Header.Set("Authorization", "Bearer secret-header")
	req.AddCookie(&http.Cookie{Name: "hydro_session", Value: "secret-cookie"})
	req = req.WithContext(security.WithRequestID(req.Context(), "request-safe-1"))
	TracingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(httptest.NewRecorder(), req)
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("spans = %d", len(spans))
	}
	serialized := spans[0].Name
	for _, attribute := range spans[0].Attributes {
		serialized += " " + string(attribute.Key) + "=" + attribute.Value.AsString()
	}
	for _, secret := range []string{"secret-query", "secret-header", "secret-cookie", "access_token", "Authorization", "hydro_session"} {
		if strings.Contains(serialized, secret) {
			t.Fatalf("secret exported in span: %s", serialized)
		}
	}
	if !strings.Contains(serialized, "url.path=/api/media") || !strings.Contains(serialized, "hydro.request_id=request-safe-1") {
		t.Fatalf("expected bounded trace attributes: %s", serialized)
	}
}
