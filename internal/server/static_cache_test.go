package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
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

func TestPurgeStaticDiskCachePathRemovesEveryHostVariant(t *testing.T) {
	root := t.TempDir()
	for _, host := range []string{"localhost", "example.test"} {
		if err := writeCachedFileForHostLimit(root, host, publicPagePath, []byte("cached"), 10, 1<<20); err != nil {
			t.Fatal(err)
		}
	}
	if err := PurgeStaticDiskCachePath(root, publicPagePath); err != nil {
		t.Fatal(err)
	}
	for _, host := range []string{"localhost", "example.test"} {
		path, err := staticCachePathForHost(root, host, publicPagePath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("host cache %q remains: %v", host, err)
		}
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

// TestStaticDiskCacheWritesBrotliSibling proves a cache write also produces a
// "<file>.html.br" sibling that Apache can serve directly (see
// deploy/hosting/.htaccess) whose decompressed content matches the plain
// ".html" cache file byte for byte.
func TestStaticDiskCacheWritesBrotliSibling(t *testing.T) {
	root := t.TempDir()
	body := "<html><body>" + strings.Repeat("Hello Public Page. ", 64) + "</body></html>"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(body))
	})
	requestCache(t, StaticDiskCacheMiddleware(StaticCacheOptions{BaseDir: root})(handler), http.MethodGet, publicPagePath, nil)

	htmlFile := filepath.Join(root, "localhost", "public", "pages", "en", "about", "html.html")
	brFile := htmlFile + ".br"
	compressed, err := os.ReadFile(brFile)
	if err != nil {
		t.Fatalf("expected a .br sibling to be written: %v", err)
	}
	decoded, err := io.ReadAll(brotli.NewReader(bytes.NewReader(compressed)))
	if err != nil {
		t.Fatalf("brotli sibling did not decode: %v", err)
	}
	if string(decoded) != body {
		t.Fatalf("brotli sibling content mismatch:\n got  %q\n want %q", decoded, body)
	}
}

// TestStaticDiskCacheBrotliSiblingRemovedOnPurge proves PurgeStaticDiskCachePath
// also removes the .br sibling, so a stale compressed copy can never outlive
// (and so shadow) a purged/updated cache entry.
func TestStaticDiskCacheBrotliSiblingRemovedOnPurge(t *testing.T) {
	root := t.TempDir()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>content</body></html>"))
	})
	requestCache(t, StaticDiskCacheMiddleware(StaticCacheOptions{BaseDir: root})(handler), http.MethodGet, publicPagePath, nil)
	brFile := filepath.Join(root, "localhost", "public", "pages", "en", "about", "html.html.br")
	if _, err := os.Stat(brFile); err != nil {
		t.Fatalf("precondition: expected .br sibling to exist before purge: %v", err)
	}
	if err := PurgeStaticDiskCachePath(root, publicPagePath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(brFile); !os.IsNotExist(err) {
		t.Fatalf("expected .br sibling removed alongside the purged .html file, stat err = %v", err)
	}
}

// TestStaticDiskCacheSetsLiteSpeedCacheControlExactlyWhenDiskCacheWould proves
// X-LiteSpeed-Cache-Control tracks the same eligibility as the disk cache:
// present on a response that gets cached, absent on one that doesn't (here, a
// response carrying a session cookie, which StaticDiskCacheMiddleware's own
// sameCookieHeaders check also refuses to cache).
func TestStaticDiskCacheSetsLiteSpeedCacheControlExactlyWhenDiskCacheWould(t *testing.T) {
	root := t.TempDir()

	cacheableHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>content</body></html>"))
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, publicPagePath, nil)
	req.Host = "localhost"
	StaticDiskCacheMiddleware(StaticCacheOptions{BaseDir: root})(cacheableHandler).ServeHTTP(rec, req)
	if got := rec.Header().Get("X-LiteSpeed-Cache-Control"); got != "public,max-age=30" {
		t.Fatalf("expected X-LiteSpeed-Cache-Control on a disk-cache-eligible response, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "localhost", "public", "pages", "en", "about", "html.html")); err != nil {
		t.Fatalf("expected the same response to also be disk-cached: %v", err)
	}

	cookieHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "hydro_session", Value: "x"})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>private</body></html>"))
	})
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/public/pages/en/other/html", nil)
	req2.Host = "localhost"
	StaticDiskCacheMiddleware(StaticCacheOptions{BaseDir: root, SessionCookie: "hydro_session"})(cookieHandler).ServeHTTP(rec2, req2)
	if got := rec2.Header().Get("X-LiteSpeed-Cache-Control"); got != "" {
		t.Fatalf("expected no X-LiteSpeed-Cache-Control on a Set-Cookie response, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "localhost", "public", "pages", "en", "other", "html.html")); !os.IsNotExist(err) {
		t.Fatalf("expected the Set-Cookie response to not be disk-cached either, stat err = %v", err)
	}
}
