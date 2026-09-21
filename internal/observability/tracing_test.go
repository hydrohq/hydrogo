package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
