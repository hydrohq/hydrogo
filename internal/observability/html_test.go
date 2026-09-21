package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTMLCommentMiddlewareRemovesCommentsFromHTML(t *testing.T) {
	handler := HTMLCommentMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<main><!-- internal note --><p>visible</p><!--[if IE]>legacy<![endif]--></main>`))
	}))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
	body := res.Body.String()
	if strings.Contains(body, "<!--") || strings.Contains(body, "internal note") || strings.Contains(body, "legacy") {
		t.Fatalf("HTML comment leaked: %q", body)
	}
	if !strings.Contains(body, "<p>visible</p>") {
		t.Fatalf("visible HTML changed: %q", body)
	}
}

func TestHTMLCommentMiddlewareLeavesJSONUnchanged(t *testing.T) {
	handler := HTMLCommentMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":"<!-- not html -->"}`))
	}))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api", nil))
	if got := res.Body.String(); !strings.Contains(got, "<!-- not html -->") {
		t.Fatalf("JSON body changed: %q", got)
	}
}
