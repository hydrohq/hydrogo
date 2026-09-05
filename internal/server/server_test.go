package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOperationalRoutesAreExplicit(t *testing.T) {
	handler := newHandler()

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
