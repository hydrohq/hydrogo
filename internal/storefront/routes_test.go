package storefront

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hydrohq/hydrogo/internal/config"
)

func newRoutesStorefront(t *testing.T, manifest string) *ExternalStorefront {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>spa</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if manifest != "" {
		if err := os.WriteFile(filepath.Join(dir, RoutesManifestFile), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	es, err := NewExternalStorefront(config.ExternalStorefrontConfig{Enabled: true, StaticDir: dir}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return es
}

func getStatus(es *ExternalStorefront, target string) (*httptest.ResponseRecorder, string) {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	es.ServeHTTP(rec, req)
	return rec, rec.Body.String()
}

const testManifest = `{
  "exact": ["/", "/urunler", "/sepet"],
  "prefixes": ["/siparis/"],
  "dynamic": [
    {"prefix": "/urun/", "collection": "products", "field": "slug"},
    {"prefix": "/", "collection": "legal_pages", "field": "slug"}
  ]
}`

func TestExternalStorefrontWithoutManifestKeepsServing200(t *testing.T) {
	es := newRoutesStorefront(t, "")
	rec, body := getStatus(es, "/anything-at-all")
	if rec.Code != http.StatusOK || body != "<html>spa</html>" {
		t.Fatalf("no manifest must keep the old fallback, got %d %q", rec.Code, body)
	}
}

func TestExternalStorefrontManifestGivesRealNotFound(t *testing.T) {
	es := newRoutesStorefront(t, testManifest)
	es.SetSlugLookup(func(_ context.Context, collection, field, value string) (bool, error) {
		return (collection == "products" && field == "slug" && value == "kars-ari-poleni") ||
			(collection == "legal_pages" && value == "kvkk"), nil
	})

	for _, test := range []struct {
		path string
		want int
	}{
		{"/", http.StatusOK},
		{"/urunler", http.StatusOK},
		{"/urunler/", http.StatusOK},
		{"/siparis/order_123", http.StatusOK},
		{"/urun/kars-ari-poleni", http.StatusOK},
		{"/kvkk", http.StatusOK},
		{"/urun/yok-boyle-urun", http.StatusNotFound},
		{"/urun/a/b", http.StatusNotFound},
		{"/siparis/", http.StatusNotFound},
		{"/.env", http.StatusNotFound},
		{"/hydro.db", http.StatusNotFound},
		{"/bin/hydro", http.StatusNotFound},
		{"/wp-login.php", http.StatusNotFound},
	} {
		rec, body := getStatus(es, test.path)
		if rec.Code != test.want {
			t.Errorf("%s: status %d, want %d", test.path, rec.Code, test.want)
		}
		if body != "<html>spa</html>" {
			t.Errorf("%s: the SPA must still render (body %q)", test.path, body)
		}
		if test.want == http.StatusNotFound && rec.Header().Get("X-Robots-Tag") != "noindex" {
			t.Errorf("%s: a 404 page must be noindex", test.path)
		}
	}
}

func TestExternalStorefrontNotFoundIgnoresConditionalHeaders(t *testing.T) {
	es := newRoutesStorefront(t, testManifest)
	es.SetSlugLookup(func(context.Context, string, string, string) (bool, error) { return false, nil })

	first := httptest.NewRecorder()
	es.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/nope", nil))
	lastModified := first.Header().Get("Last-Modified")

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}
	rec := httptest.NewRecorder()
	es.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("a revalidated unknown page must stay 404, got %d", rec.Code)
	}
}

func TestExternalStorefrontLookupFailsOpen(t *testing.T) {
	es := newRoutesStorefront(t, testManifest)

	// No lookup wired: a dynamic route cannot be verified, so it is not 404'd.
	if rec, _ := getStatus(es, "/urun/whatever"); rec.Code != http.StatusOK {
		t.Fatalf("unverifiable dynamic route must serve 200, got %d", rec.Code)
	}

	es.SetSlugLookup(func(context.Context, string, string, string) (bool, error) {
		return false, errors.New("database is down")
	})
	if rec, _ := getStatus(es, "/urun/another"); rec.Code != http.StatusOK {
		t.Fatalf("a failing lookup must never 404 real pages, got %d", rec.Code)
	}
}

func TestExternalStorefrontCachesLookups(t *testing.T) {
	es := newRoutesStorefront(t, testManifest)
	calls := 0
	es.SetSlugLookup(func(context.Context, string, string, string) (bool, error) {
		calls++
		return false, nil
	})
	for i := 0; i < 5; i++ {
		getStatus(es, "/urun/scanner-probe")
	}
	if calls != 1 {
		t.Fatalf("lookup ran %d times for one slug, want 1 (scanners must not hammer the database)", calls)
	}
}

func TestExternalStorefrontBrokenManifestBehavesAsAbsent(t *testing.T) {
	es := newRoutesStorefront(t, `{not json`)
	if rec, _ := getStatus(es, "/whatever"); rec.Code != http.StatusOK {
		t.Fatalf("a broken manifest must not take pages down, got %d", rec.Code)
	}
}

func TestExternalStorefrontStaticFilesAreUnaffectedByManifest(t *testing.T) {
	es := newRoutesStorefront(t, testManifest)
	if err := os.WriteFile(filepath.Join(es.staticDir, "sitemap.xml"), []byte("<urlset/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec, body := getStatus(es, "/sitemap.xml")
	if rec.Code != http.StatusOK || body != "<urlset/>" {
		t.Fatalf("real static files must keep working, got %d %q", rec.Code, body)
	}
}
