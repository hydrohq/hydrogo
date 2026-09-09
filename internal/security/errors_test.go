package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorMiddlewareHidesInternalServerDetails(t *testing.T) {
	handler := ErrorMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "database password=super-secret connection detail", http.StatusInternalServerError)
	}))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusInternalServerError)
	}
	if strings.Contains(res.Body.String(), "super-secret") || strings.Contains(res.Body.String(), "connection detail") {
		t.Fatalf("internal error details leaked: %q", res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `"message":"internal server error"`) {
		t.Fatalf("generic internal error message missing: %q", res.Body.String())
	}
}

func TestErrorMiddlewarePreservesSafeServiceUnavailableMessage(t *testing.T) {
	handler := ErrorMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "database is not ready", http.StatusServiceUnavailable)
	}))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))

	if res.Code != http.StatusServiceUnavailable || !strings.Contains(res.Body.String(), `"message":"service unavailable"`) {
		t.Fatalf("unexpected service unavailable response: %d %q", res.Code, res.Body.String())
	}
}
