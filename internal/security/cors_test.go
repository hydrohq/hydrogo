package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublicReadCORSAllowsOnlyConfiguredPublishedGET(t *testing.T) {
	middleware, err := NewPublicReadCORSMiddleware([]string{"https://store.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	next := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	for _, path := range []string{"/api/v1/news?locale=tr", "/api/v1/pages?locale=tr"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Origin", "https://store.example.test")
		recorder := httptest.NewRecorder()
		next.ServeHTTP(recorder, request)
		if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://store.example.test" {
			t.Fatalf("%s allow origin = %q", path, got)
		}
		if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "" {
			t.Fatalf("%s credentialed CORS unexpectedly enabled: %q", path, got)
		}
		if got := recorder.Header().Get("Vary"); got != "Origin" {
			t.Fatalf("%s vary = %q", path, got)
		}
	}
}

func TestPublicReadCORSDeniesUnknownOriginAndWrites(t *testing.T) {
	middleware, err := NewPublicReadCORSMiddleware([]string{"https://store.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	next := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	for name, request := range map[string]*http.Request{
		"unknown":             httptest.NewRequest(http.MethodGet, "/api/v1/news?locale=tr", nil),
		"write":               httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", nil),
		"member-mobile-write": httptest.NewRequest(http.MethodPost, "/api/v1/mobile/member/commerce/cart/items", nil),
		"options":             httptest.NewRequest(http.MethodOptions, "/api/v1/news", nil),
	} {
		request.Header.Set("Origin", "https://evil.example.test")
		recorder := httptest.NewRecorder()
		next.ServeHTTP(recorder, request)
		if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("%s unexpectedly allowed origin %q", name, got)
		}
	}
}

func TestPublicReadCORSRejectsUnsafeOriginConfiguration(t *testing.T) {
	for _, origin := range []string{"*", "https://store.example.test/path", "https://user:pass@store.example.test", "ftp://store.example.test"} {
		if _, err := NewPublicReadCORSMiddleware([]string{origin}); err == nil {
			t.Errorf("unsafe origin %q accepted", origin)
		}
	}
}
