package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOperationalRoutesAreExplicit(t *testing.T) {
	handler := newHandler(nil)

	for _, route := range []struct {
		path string
		want int
	}{
		{path: "/healthz", want: http.StatusOK},
		{path: "/readyz", want: http.StatusOK},
		{path: "/unregistered", want: http.StatusNotFound},
	} {
		req := httptest.NewRequest(http.MethodGet, route.path, nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if got := res.Code; got != route.want {
			t.Errorf("GET %s status = %d, want %d", route.path, got, route.want)
		}
	}
}

func TestReadinessFailureDoesNotLeakDetails(t *testing.T) {
	handler := newHandler(func(context.Context) error { return errors.New("secret database detail") })
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
	if strings.Contains(res.Body.String(), "secret") {
		t.Fatalf("readiness leaked internal error: %q", res.Body.String())
	}
}
