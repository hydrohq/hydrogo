package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestStaticDiskCacheMiddleware(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hydro-static-cache-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>Hello Public Page</body></html>"))
	})

	middleware := StaticDiskCacheMiddleware(StaticCacheOptions{
		BaseDir: tempDir,
	})

	cachedHandler := middleware(handler)

	// 1. GET /about should write to <tempDir>/about.html
	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	rec := httptest.NewRecorder()
	cachedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	cachedFile := filepath.Join(tempDir, "about.html")
	data, err := os.ReadFile(cachedFile)
	if err != nil {
		t.Fatalf("expected cached file %s to exist, err: %v", cachedFile, err)
	}
	if string(data) != "<html><body>Hello Public Page</body></html>" {
		t.Fatalf("unexpected cached content: %s", string(data))
	}

	// 2. GET with session cookie should bypass writing
	recWithCookie := httptest.NewRecorder()
	reqWithCookie := httptest.NewRequest(http.MethodGet, "/contact", nil)
	reqWithCookie.AddCookie(&http.Cookie{Name: "hydro_session", Value: "session-123"})
	cachedHandler.ServeHTTP(recWithCookie, reqWithCookie)

	contactFile := filepath.Join(tempDir, "contact.html")
	if _, err := os.Stat(contactFile); !os.IsNotExist(err) {
		t.Fatalf("expected authenticated request not to create cached file %s", contactFile)
	}

	// 3. Purge specific path
	if err := PurgeStaticDiskCachePath(tempDir, "/about"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cachedFile); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be deleted after purge", cachedFile)
	}
}
