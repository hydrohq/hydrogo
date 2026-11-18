package storefront

import (
	"bytes"
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/hydrohq/hydrogo/internal/config"
)

func TestExternalStorefront_StaticFilesAndPrecompression(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	immutableDir := filepath.Join(tempDir, "_app", "immutable", "assets")
	if err := os.MkdirAll(immutableDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cssContent := "body { color: red; }"
	cssFile := filepath.Join(immutableDir, "style.css")
	if err := os.WriteFile(cssFile, []byte(cssContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create .gz precompressed
	var gzBuf bytes.Buffer
	gzWriter := gzip.NewWriter(&gzBuf)
	_, _ = gzWriter.Write([]byte(cssContent))
	_ = gzWriter.Close()
	if err := os.WriteFile(cssFile+".gz", gzBuf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create .br precompressed
	var brBuf bytes.Buffer
	brWriter := brotli.NewWriter(&brBuf)
	_, _ = brWriter.Write([]byte(cssContent))
	_ = brWriter.Close()
	if err := os.WriteFile(cssFile+".br", brBuf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create favicon in root
	if err := os.WriteFile(filepath.Join(tempDir, "favicon.svg"), []byte("<svg></svg>"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.ExternalStorefrontConfig{
		Enabled:   true,
		StaticDir: tempDir,
		Upstream:  "http://127.0.0.1:9999", // Unused because static hits first
	}

	es, err := NewExternalStorefront(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewExternalStorefront: %v", err)
	}

	// Test 1: Request Brotli
	req := httptest.NewRequest(http.MethodGet, "/_app/immutable/assets/style.css", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	rec := httptest.NewRecorder()
	es.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Encoding") != "br" {
		t.Fatalf("expected Content-Encoding: br, got %q", rec.Header().Get("Content-Encoding"))
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("expected immutable Cache-Control, got %q", rec.Header().Get("Cache-Control"))
	}

	// Test 2: Request Gzip
	req = httptest.NewRequest(http.MethodGet, "/_app/immutable/assets/style.css", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	rec = httptest.NewRecorder()
	es.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected Content-Encoding: gzip, got %q", rec.Header().Get("Content-Encoding"))
	}

	// Test 3: Request Uncompressed (no Accept-Encoding)
	req = httptest.NewRequest(http.MethodGet, "/_app/immutable/assets/style.css", nil)
	rec = httptest.NewRecorder()
	es.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("expected no Content-Encoding, got %q", rec.Header().Get("Content-Encoding"))
	}
	if rec.Body.String() != cssContent {
		t.Fatalf("expected body %q, got %q", cssContent, rec.Body.String())
	}

	// Test 4: Missing file under /_app/ returns 404 without hitting upstream
	req = httptest.NewRequest(http.MethodGet, "/_app/missing.js", nil)
	rec = httptest.NewRecorder()
	es.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing /_app/ file, got %d", rec.Code)
	}

	// Test 5: Favicon serving
	req = httptest.NewRequest(http.MethodGet, "/favicon.svg", nil)
	rec = httptest.NewRecorder()
	es.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for favicon, got %d", rec.Code)
	}
	if rec.Body.String() != "<svg></svg>" {
		t.Fatalf("expected body '<svg></svg>', got %q", rec.Body.String())
	}
}

func TestExternalStorefront_ProxyToUpstream(t *testing.T) {
	// Create mock upstream server (e.g. SvelteKit SSR)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("X-SSR", "sveltekit")
			_, _ = w.Write([]byte("<html>Home SSR</html>"))
			return
		}
		if r.URL.Path == "/api/cart" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer upstream.Close()

	cfg := config.ExternalStorefrontConfig{
		Enabled:  true,
		Upstream: upstream.URL,
	}

	es, err := NewExternalStorefront(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewExternalStorefront: %v", err)
	}

	// Test dynamic page SSR
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	es.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-SSR") != "sveltekit" {
		t.Fatalf("expected X-SSR: sveltekit, got %q", rec.Header().Get("X-SSR"))
	}
	if rec.Body.String() != "<html>Home SSR</html>" {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}

	// Test storefront API route
	req = httptest.NewRequest(http.MethodPost, "/api/cart", nil)
	rec = httptest.NewRecorder()
	es.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != `{"items":[]}` {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}

func TestExternalStorefront_UpstreamDownReturns503(t *testing.T) {
	cfg := config.ExternalStorefrontConfig{
		Enabled:  true,
		Upstream: "http://127.0.0.1:59999", // closed port
	}

	es, err := NewExternalStorefront(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewExternalStorefront: %v", err)
	}

	// HTML request
	req := httptest.NewRequest(http.MethodGet, "/urun/ornek", nil)
	rec := httptest.NewRecorder()
	es.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Mağaza Hazırlanıyor") {
		t.Fatalf("expected friendly startup page, got %q", rec.Body.String())
	}

	// JSON request
	req = httptest.NewRequest(http.MethodGet, "/api/cart", nil)
	req.Header.Set("Accept", "application/json")
	rec = httptest.NewRecorder()
	es.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "storefront starting up") {
		t.Fatalf("expected json error, got %q", rec.Body.String())
	}
}

func TestParseEnvFile(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "test.env")
	content := `
# Comment
PORT=3100
HOST="127.0.0.1"
PUBLIC_NAME='BEYOND Luxury'
EMPTY=
`
	if err := os.WriteFile(tempFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	env, err := parseEnvFile(tempFile)
	if err != nil {
		t.Fatalf("parseEnvFile: %v", err)
	}

	if env["PORT"] != "3100" {
		t.Errorf("expected PORT=3100, got %q", env["PORT"])
	}
	if env["HOST"] != "127.0.0.1" {
		t.Errorf("expected HOST=127.0.0.1, got %q", env["HOST"])
	}
	if env["PUBLIC_NAME"] != "BEYOND Luxury" {
		t.Errorf("expected PUBLIC_NAME='BEYOND Luxury', got %q", env["PUBLIC_NAME"])
	}
}

func TestExternalStorefront_ForwardsProtoAndSetsFlushInterval(t *testing.T) {
	var seenProto string
	var seenHost string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenProto = r.Header.Get("X-Forwarded-Proto")
		seenHost = r.Header.Get("X-Forwarded-Host")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	es, err := NewExternalStorefront(config.ExternalStorefrontConfig{
		Enabled:  true,
		Upstream: upstream.URL,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewExternalStorefront: %v", err)
	}
	if es.proxy.FlushInterval <= 0 {
		t.Fatalf("FlushInterval = %v, want > 0 for streaming SSR", es.proxy.FlushInterval)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "beyond.example"
	rec := httptest.NewRecorder()
	es.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || seenProto != "http" || seenHost != "beyond.example" {
		t.Fatalf("status=%d proto=%q host=%q", rec.Code, seenProto, seenHost)
	}
}

func TestExternalStorefront_SingleBinaryStaticSPA(t *testing.T) {
	tempDir := t.TempDir()

	// Write index.html
	indexHTML := "<html><body><h1>Kiziroglu Storefront</h1></body></html>"
	if err := os.WriteFile(filepath.Join(tempDir, "index.html"), []byte(indexHTML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write an asset
	assetsDir := filepath.Join(tempDir, "assets")
	_ = os.MkdirAll(assetsDir, 0o755)
	_ = os.WriteFile(filepath.Join(assetsDir, "app.js"), []byte("console.log('hi');"), 0o644)

	cfg := config.ExternalStorefrontConfig{
		Enabled:   true,
		StaticDir: tempDir,
		Upstream:  "", // Pure static / single-binary mode
	}

	es, err := NewExternalStorefront(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewExternalStorefront: %v", err)
	}

	// 1. Root "/" must serve index.html
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	es.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Kiziroglu Storefront") {
		t.Fatalf("expected index.html content, got %s", rr.Body.String())
	}

	// 2. Asset "/assets/app.js" must serve JS file
	req = httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rr = httptest.NewRecorder()
	es.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for asset, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "console.log") {
		t.Fatalf("expected asset content, got %s", rr.Body.String())
	}

	// 3. SPA Route "/kategori/kars-peynirleri" must fallback to index.html
	req = httptest.NewRequest(http.MethodGet, "/kategori/kars-peynirleri", nil)
	rr = httptest.NewRecorder()
	es.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for SPA route, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Kiziroglu Storefront") {
		t.Fatalf("expected fallback index.html, got %s", rr.Body.String())
	}
}

func TestExternalStorefront_ServesEmbeddedFallback(t *testing.T) {
	cfg := config.ExternalStorefrontConfig{
		Enabled:   true,
		StaticDir: "/nonexistent/static/path",
		Upstream:  "", // Pure embedded single binary mode
	}

	es, err := NewExternalStorefront(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewExternalStorefront: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	es.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from embedded dist, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "hydro storefront") {
		t.Fatalf("expected embedded content, got %s", rr.Body.String())
	}
}
