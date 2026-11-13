package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hydrohq/hydrogo/internal/config"
	"github.com/hydrohq/hydrogo/internal/router"
)

func TestServeApp_ExternalStorefrontRoutes(t *testing.T) {
	// Create mock upstream SSR server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "sveltekit-ssr")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>Storefront SSR Page</html>"))
	}))
	defer upstream.Close()

	// Create temp static dir
	staticDir := t.TempDir()
	faviconPath := filepath.Join(staticDir, "favicon.svg")
	if err := os.WriteFile(faviconPath, []byte("<svg>icon</svg>"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := &serveApp{
		cfg: config.Config{
			Storefront: config.StorefrontConfig{
				External: config.ExternalStorefrontConfig{
					Enabled:   true,
					Upstream:  upstream.URL,
					StaticDir: staticDir,
				},
			},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	if err := app.initStorefront(); err != nil {
		t.Fatalf("initStorefront failed: %v", err)
	}
	if app.externalStorefront == nil {
		t.Fatal("expected externalStorefront to be initialized")
	}

	routes := app.storefrontRoutes()
	if len(routes) == 0 {
		t.Fatal("expected non-empty storefront routes")
	}

	// Verify route has Pattern "/"
	foundRoot := false
	for _, r := range routes {
		if r.Pattern == "/" {
			foundRoot = true
			break
		}
	}
	if !foundRoot {
		t.Fatal("expected route with Pattern '/'")
	}

	handler, err := router.New(routes, func(r *http.Request) bool { return true })
	if err != nil {
		t.Fatalf("router.New failed: %v", err)
	}

	// Test 1: Static file is served directly
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/favicon.svg", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "<svg>icon</svg>" {
		t.Fatalf("expected '<svg>icon</svg>', got %q", rec.Body.String())
	}

	// Test 2: Dynamic page is proxied to upstream SSR
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/koleksiyonlar", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Upstream") != "sveltekit-ssr" {
		t.Fatalf("expected X-Upstream: sveltekit-ssr, got %q", rec.Header().Get("X-Upstream"))
	}
	if rec.Body.String() != "<html>Storefront SSR Page</html>" {
		t.Fatalf("expected SSR body, got %q", rec.Body.String())
	}
}
