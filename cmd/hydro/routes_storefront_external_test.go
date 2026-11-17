package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hydrohq/hydrogo/internal/config"
	"github.com/hydrohq/hydrogo/internal/router"
	"github.com/hydrohq/hydrogo/internal/server"
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

func TestServeApp_ExternalStorefrontFullServerStack(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte("<!--[--><div>form error</div><script>__sveltekit</script><!--]-->"))
	}))
	defer upstream.Close()

	app := &serveApp{
		cfg: config.Config{
			Storefront: config.StorefrontConfig{
				External: config.ExternalStorefrontConfig{
					Enabled:  true,
					Upstream: upstream.URL,
				},
			},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	if err := app.initStorefront(); err != nil {
		t.Fatalf("initStorefront failed: %v", err)
	}

	routes := append(app.storefrontRoutes(), router.Route{
		Pattern: "POST /api/v1/commerce/checkout",
		Access:  router.Public,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
	})

	h := server.NewHandler(server.Config{
		Routes:       routes,
		PublicOrigin: "https://shop.example.com",
		EnableHSTS:   true,
	})

	// 1. Browser POST to SvelteKit action (/odeme) must bypass HydroGo CSRF,
	// ErrorMiddleware, HeadersMiddleware CSP, and HTMLCommentMiddleware.
	formBody := strings.NewReader("guest_email=test%40example.com")
	req := httptest.NewRequest(http.MethodPost, "https://shop.example.com/odeme", formBody)
	req.Host = "shop.example.com"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("POST /odeme status = %d, want 422 (body=%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "<!--[-->") || !strings.Contains(rec.Body.String(), "<script>__sveltekit</script>") {
		t.Fatalf("POST /odeme body modified unexpectedly: %q", rec.Body.String())
	}
	if csp := rec.Header().Get("Content-Security-Policy"); csp != "" {
		t.Fatalf("POST /odeme unexpectedly received HydroGo CSP header %q", csp)
	}
	if hsts := rec.Header().Get("Strict-Transport-Security"); hsts == "" {
		t.Fatalf("POST /odeme missing HSTS header when EnableHSTS is true")
	}

	// 2. HydroGo's own API route without CSRF token must still be rejected with 403.
	apiReq := httptest.NewRequest(http.MethodPost, "https://shop.example.com/api/v1/commerce/checkout", strings.NewReader(`{}`))
	apiReq.Host = "shop.example.com"
	apiReq.Header.Set("Content-Type", "application/json")
	apiRec := httptest.NewRecorder()
	h.ServeHTTP(apiRec, apiReq)

	if apiRec.Code != http.StatusForbidden {
		t.Fatalf("POST /api/v1/commerce/checkout without CSRF = %d, want 403", apiRec.Code)
	}
}
