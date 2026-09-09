package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRejectsUnclassifiedRoute(t *testing.T) {
	_, err := New([]Route{{Pattern: "GET /unsafe", Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}}, nil)
	if err == nil {
		t.Fatal("unclassified route was accepted")
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
