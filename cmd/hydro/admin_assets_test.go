package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminAssetsRouteServesEmbeddedMotionAssets(t *testing.T) {
	route := adminAssetsRoute()
	mux := http.NewServeMux()
	mux.Handle(route.Pattern, route.Handler)
	for path, contentType := range map[string]string{"/admin-assets/admin.css": "text/css", "/admin-assets/motion.js": "javascript"} {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Header().Get("Content-Type"), contentType) {
			t.Fatalf("%s: status=%d content-type=%q", path, recorder.Code, recorder.Header().Get("Content-Type"))
		}
		if recorder.Header().Get("Cache-Control") != "no-cache" {
			t.Fatalf("%s: Cache-Control=%q", path, recorder.Header().Get("Cache-Control"))
		}
	}

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/admin-assets/admin.css", nil))
	etag := recorder.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}
	conditional := httptest.NewRequest(http.MethodGet, "/admin-assets/admin.css", nil)
	conditional.Header.Set("If-None-Match", etag)
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, conditional)
	if recorder.Code != http.StatusNotModified {
		t.Fatalf("conditional request status=%d, want 304", recorder.Code)
	}
}
