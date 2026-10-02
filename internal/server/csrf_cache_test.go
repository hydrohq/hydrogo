package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/security"
)

// TestCSRFCookieDeadlockedStaticCacheWithoutSkip reproduces the bug: CSRFMiddleware
// with no skip predicate always sets hydro_csrf on a cookie-less visitor's GET,
// which StaticDiskCacheMiddleware's sameCookieHeaders check then refuses to cache.
// A genuinely anonymous visitor (the only kind that can ever hit the static disk
// cache per deploy/hosting/.htaccess, which only serves it when %{HTTP_COOKIE} is
// empty) can therefore never populate the cache.
func TestCSRFCookieDeadlockedStaticCacheWithoutSkip(t *testing.T) {
	root := t.TempDir()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>News list</body></html>"))
	})
	handler := StaticDiskCacheMiddleware(StaticCacheOptions{BaseDir: root})(security.CSRFMiddleware(false, nil)(inner))

	req := httptest.NewRequest(http.MethodGet, "/public/news/en/html", nil)
	req.Host = "localhost"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Set-Cookie") == "" {
		t.Fatal("expected the baseline (no skip predicate) request to mint a CSRF cookie")
	}
	cacheFile := filepath.Join(root, "localhost", "public", "news", "en", "html.html")
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Fatalf("deadlock reproduction failed: cache file should not exist when every anonymous GET sets a cookie, stat err = %v", err)
	}
}

// TestCSRFSkipUnblocksStaticDiskCacheForPublicContent proves the fix: wiring
// publicContentCacheEligible as CSRFMiddleware's skip predicate (exactly as
// internal/server/server.go does) lets a cookie-less request to any of the
// public content HTML routes populate the static disk cache, because no
// Set-Cookie header is written.
func TestCSRFSkipUnblocksStaticDiskCacheForPublicContent(t *testing.T) {
	for _, tc := range []struct {
		path      string
		cacheFile []string
	}{
		{"/public/news/en/html", []string{"localhost", "public", "news", "en", "html.html"}},
		{"/public/pages/en/about/html", []string{"localhost", "public", "pages", "en", "about", "html.html"}},
		{"/public/news/en/my-article/html", []string{"localhost", "public", "news", "en", "my-article", "html.html"}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			root := t.TempDir()
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte("<html><body>content</body></html>"))
			})
			handler := StaticDiskCacheMiddleware(StaticCacheOptions{BaseDir: root})(security.CSRFMiddleware(false, publicContentCacheEligible)(inner))

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Host = "localhost"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if got := rec.Header().Get("Set-Cookie"); got != "" {
				t.Fatalf("expected no Set-Cookie for the cache-eligible route %s, got %q", tc.path, got)
			}
			cacheFile := filepath.Join(append([]string{root}, tc.cacheFile...)...)
			data, err := os.ReadFile(cacheFile)
			if err != nil || string(data) != "<html><body>content</body></html>" {
				t.Fatalf("expected the disk cache write to succeed for %s, data=%q err=%v", tc.path, data, err)
			}
		})
	}
}

// TestCSRFSkipDoesNotApplyOutsidePublicContentRoutes guards against widening
// publicContentCacheEligible beyond the exact route set staticCacheURLPath
// recognizes: anything else (admin, auth, member, API routes) must keep
// receiving the cookie exactly as before.
func TestCSRFSkipDoesNotApplyOutsidePublicContentRoutes(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := security.CSRFMiddleware(false, publicContentCacheEligible)(inner)

	for _, path := range []string{
		"/admin/pages",
		"/auth/login",
		"/member/login",
		"/public/pages/en/about", // missing the trailing /html segment
		"/csrf-token",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Header().Get("Set-Cookie") == "" {
			t.Fatalf("%s: expected the CSRF cookie to still be set for a non-cache-eligible route", path)
		}
	}
}

// TestCSRFSkipRouteStillRejectsForgedMutation proves the skip predicate does not
// weaken protection for the routes it touches: even though GET no longer sets
// a cookie, a POST to the same paths with no cookie/token is still rejected.
func TestCSRFSkipRouteStillRejectsForgedMutation(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := security.CSRFMiddleware(false, publicContentCacheEligible)(inner)

	for _, path := range []string{
		"/public/news/en/html",
		"/public/pages/en/about/html",
		"/public/news/en/my-article/html",
	} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s: expected forged mutation on the skip-eligible route to be rejected, got status %d", path, rec.Code)
		}
	}
}
