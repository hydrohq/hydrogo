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
