package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/router"
)

func TestAdminUIRoutes(t *testing.T) {
	routes := Routes()
	handler, err := router.New(routes, func(r *http.Request) bool { return true })
	if err != nil {
		t.Fatalf("router.New failed: %v", err)
	}

	t.Run("GET /admin serves index.html", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
			t.Errorf("expected text/html, got %q", rec.Header().Get("Content-Type"))
		}
		if etag := rec.Header().Get("ETag"); etag == "" {
			t.Errorf("expected ETag header to be set")
		}
		if !strings.Contains(rec.Body.String(), "HydroGo Admin") {
			t.Errorf("expected body to contain HydroGo Admin, got %q", rec.Body.String())
		}
	})

	t.Run("GET /admin/login serves index.html fallback", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin/login", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
			t.Errorf("expected text/html, got %q", rec.Header().Get("Content-Type"))
		}
	})

	t.Run("ETag 304 response on If-None-Match", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		etag := rec.Header().Get("ETag")
		req2 := httptest.NewRequest("GET", "/admin", nil)
		req2.Header.Set("If-None-Match", etag)
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusNotModified {
			t.Fatalf("expected 304, got %d", rec2.Code)
		}
	})
}
