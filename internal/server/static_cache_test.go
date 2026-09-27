package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const publicPagePath = "/public/pages/en/about/html"

func TestStaticDiskCacheMiddleware(t *testing.T) {
	root := t.TempDir()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>Hello Public Page</body></html>"))
	})
	cachedHandler := StaticDiskCacheMiddleware(StaticCacheOptions{BaseDir: root})(handler)
	requestCache(t, cachedHandler, http.MethodGet, publicPagePath, nil)
	cacheFile := filepath.Join(root, "localhost", "public", "pages", "en", "about", "html.html")
	data, err := os.ReadFile(cacheFile)
	if err != nil || string(data) != "<html><body>Hello Public Page</body></html>" {
		t.Fatalf("cache file content = %q, err = %v", data, err)
	}
	if err := PurgeStaticDiskCachePath(root, publicPagePath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Fatalf("expected cache file removed, stat err = %v", err)
	}
}

func TestStaticDiskCacheRejectsUnsafeOrVariantRequests(t *testing.T) {
	root := t.TempDir()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("public"))
	})
	cachedHandler := StaticDiskCacheMiddleware(StaticCacheOptions{BaseDir: root})(handler)
	for _, tc := range []struct {
		name   string
		path   string
		method string
		cookie *http.Cookie
		header string
	}{
		{name: "unknown path", path: "/about", method: http.MethodGet},
		{name: "query", path: publicPagePath + "?preview=1", method: http.MethodGet},
		{name: "session", path: publicPagePath, method: http.MethodGet, cookie: &http.Cookie{Name: "hydro_session", Value: "secret"}},
		{name: "htmx", path: publicPagePath, method: http.MethodGet, header: "HX-Request"},
		{name: "post", path: publicPagePath, method: http.MethodPost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Host = "localhost"
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			if tc.header != "" {
				req.Header.Set(tc.header, "true")
			}
			cachedHandler.ServeHTTP(httptest.NewRecorder(), req)
			if _, err := os.Stat(filepath.Join(root, "localhost", "public", "pages", "en", "about", "html.html")); !os.IsNotExist(err) {
				t.Fatalf("variant request wrote cache file, stat err = %v", err)
			}
		})
	}
}

func TestStaticDiskCacheSkipsPrivateNonceAndOversizedResponses(t *testing.T) {
	for _, tc := range []struct {
		name      string
		configure func(http.ResponseWriter)
		body      string
	}{
		{name: "private", configure: func(w http.ResponseWriter) { w.Header().Set("Cache-Control", "private") }, body: "html"},
		{name: "nonce", body: `<script nonce="random">x</script>`},
		{name: "oversized", body: string(make([]byte, staticCacheBodyLimit+1))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				if tc.configure != nil {
					tc.configure(w)
				}
				_, _ = w.Write([]byte(tc.body))
			})
			req := httptest.NewRequest(http.MethodGet, publicPagePath, nil)
			req.Host = "localhost"
			StaticDiskCacheMiddleware(StaticCacheOptions{BaseDir: root})(handler).ServeHTTP(httptest.NewRecorder(), req)
			if _, err := os.Stat(filepath.Join(root, "localhost", "public", "pages", "en", "about", "html.html")); !os.IsNotExist(err) {
				t.Fatalf("non-cacheable response was stored, stat err = %v", err)
			}
		})
	}
}

func requestCache(t *testing.T, handler http.Handler, method, path string, cookie *http.Cookie) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Host = "localhost"
	if cookie != nil {
		req.AddCookie(cookie)
	}
	handler.ServeHTTP(httptest.NewRecorder(), req)
}
