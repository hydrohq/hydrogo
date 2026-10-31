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

func TestErrorMiddlewareDoesNotFlushInternalErrorDetails(t *testing.T) {
	handler := ErrorMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("secret database details"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
	if strings.Contains(res.Body.String(), "secret database details") || !strings.Contains(res.Body.String(), "internal server error") {
		t.Fatalf("internal error details leaked after flush: %q", res.Body.String())
	}
}

func TestErrorMiddlewareUnwrapsResponseWriter(t *testing.T) {
	underlying := httptest.NewRecorder()
	seen := false
	handler := ErrorMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		unwrapper, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok || unwrapper.Unwrap() != underlying {
			t.Error("error middleware did not expose the underlying response writer")
			return
		}
		seen = true
	}))
	handler.ServeHTTP(underlying, httptest.NewRequest(http.MethodGet, "/", nil))
	if !seen {
		t.Fatal("underlying response writer was not unwrapped")
	}
}

func TestErrorMiddlewareBrowserHTMLWithCSP(t *testing.T) {
	handler := ErrorMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "page not found", http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/some-page", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", res.Code)
	}
	contentType := res.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Fatalf("expected text/html content-type for browser request, got %q", contentType)
	}
	csp := res.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "'unsafe-inline'") {
		t.Fatalf("expected CSP to allow unsafe-inline styles for error page, got %q", csp)
	}
	body := res.Body.String()
	if !strings.Contains(body, "Sayfa Bulunamadı") {
		t.Fatalf("expected styled HTML error page with Turkish title, got %q", body)
	}
}
