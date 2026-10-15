package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type nilHandler struct{}

func (*nilHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}

func TestNewRejectsTypedNilHandler(t *testing.T) {
	var handler *nilHandler
	if _, err := New([]Route{{Pattern: "GET /nil", Access: Public, Handler: handler}}, nil); err == nil {
		t.Fatal("typed nil handler was accepted")
	}
}

func TestNewRejectsUnclassifiedRoute(t *testing.T) {
	_, err := New([]Route{{Pattern: "GET /unsafe", Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}}, nil)
	if err == nil {
		t.Fatal("unclassified route was accepted")
	}
}

func TestNewRejectsDuplicateExpandedRoutePatterns(t *testing.T) {
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	_, err := New([]Route{
		{Pattern: "GET /api/pages", Access: Public, Handler: handler},
		{Pattern: "GET /api/v1/pages", Access: Public, Handler: handler},
	}, nil)
	if err == nil {
		t.Fatal("duplicate API alias route was accepted")
	}
	if got := err.Error(); got != `duplicate route pattern "GET /api/v1/pages"` {
		t.Fatalf("duplicate route error = %q", got)
	}
}

func TestProtectedRouteRequiresAuthorization(t *testing.T) {
	handler, err := New([]Route{{
		Pattern: "GET /admin",
		Access:  Protected,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}}, func(r *http.Request) bool { return r.Header.Get("Authorization") == "test" })
	if err != nil {
		t.Fatal(err)
	}

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorized.Code)
	}
	var envelope struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(unauthorized.Body.Bytes(), &envelope); err != nil || envelope.Code != "unauthorized" || envelope.Message == "" {
		t.Fatalf("unauthorized body = %q err=%v", unauthorized.Body.String(), err)
	}
	if got := unauthorized.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("unauthorized content type = %q", got)
	}

	// Browser navigations get the same JSON 401: the SPA handles redirects.
	browserRequest := httptest.NewRequest(http.MethodGet, "/admin?tab=drafts", nil)
	browserRequest.Header.Set("Accept", "text/html")
	browser := httptest.NewRecorder()
	handler.ServeHTTP(browser, browserRequest)
	if browser.Code != http.StatusUnauthorized || browser.Header().Get("Location") != "" {
		t.Fatalf("browser response = %d %q", browser.Code, browser.Header().Get("Location"))
	}

	request := httptest.NewRequest(http.MethodGet, "/admin", nil)
	request.Header.Set("Authorization", "test")
	authorized := httptest.NewRecorder()
	handler.ServeHTTP(authorized, request)
	if authorized.Code != http.StatusNoContent {
		t.Fatalf("authorized status = %d", authorized.Code)
	}
}

func TestAPIV1AliasPreservesAuthorizationAndDeprecatesLegacyPath(t *testing.T) {
	handler, err := New([]Route{{
		Pattern: "GET /api/pages/{id}",
		Access:  Protected,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}}, func(r *http.Request) bool { return r.Header.Get("Authorization") == "test" })
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"/api/pages/one", "/api/v1/pages/one"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "test")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusNoContent {
			t.Fatalf("GET %s status = %d, want %d", path, res.Code, http.StatusNoContent)
		}
		legacy := path == "/api/pages/one"
		if got := res.Header().Get("Deprecation"); legacy && got != "true" {
			t.Fatalf("legacy deprecation header = %q", got)
		} else if !legacy && got != "" {
			t.Fatalf("v1 deprecation header = %q", got)
		}
	}
}
