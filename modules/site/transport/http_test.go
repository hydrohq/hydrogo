package transport

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type fakeStore struct{}

func (fakeStore) UpsertSetting(context.Context, database.Executor, site.Setting) error { return nil }
func (fakeStore) ListPublicSettings(context.Context, database.Executor) ([]site.Setting, error) {
	return []site.Setting{{Key: "site.name", Value: "HydroGo", Public: true}}, nil
}
func (fakeStore) ReplaceNavigation(context.Context, database.Executor, string, []site.NavigationItem) error {
	return nil
}
func (fakeStore) ListNavigation(context.Context, database.Executor, string) ([]site.NavigationItem, error) {
	return []site.NavigationItem{{ID: "home", Locale: "en", Label: "Home", Href: "/", Position: 0, Visible: true}}, nil
}
func (fakeStore) PutRedirect(context.Context, database.Executor, site.Redirect) error { return nil }
func (fakeStore) GetRedirect(context.Context, database.Executor, string) (site.Redirect, error) {
	return site.Redirect{}, sql.ErrNoRows
}
func (fakeStore) CreateSubmission(context.Context, database.Executor, site.Submission) error {
	return nil
}
func (fakeStore) ListSubmissions(context.Context, database.Executor, string, int) ([]site.Submission, error) {
	return nil, nil
}
func (fakeStore) UpdateSubmissionStatus(context.Context, database.Executor, string, string) error {
	return nil
}
func (fakeStore) PurgeSubmissionsBefore(context.Context, database.Executor, time.Time) (int64, error) {
	return 0, nil
}

type fakeExecutor struct{}

func (fakeExecutor) ExecContext(context.Context, string, ...any) (sql.Result, error) { return nil, nil }
func (fakeExecutor) QueryRowContext(context.Context, string, ...any) *sql.Row        { return nil }

func TestFormBuilderRoutesArePresent(t *testing.T) {
	handler := Handler{FormsEnabled: true}
	want := map[string]bool{"GET /admin/site/forms": false, "POST /admin/site/forms": false, "POST /admin/site/forms/{id}/delete": false}
	for _, route := range handler.Routes() {
		if _, ok := want[route.Pattern]; ok {
			want[route.Pattern] = true
		}
	}
	for pattern, found := range want {
		if !found {
			t.Fatalf("missing form route %q", pattern)
		}
	}
}

func TestSEORoutesAreAvailableOnlyWithSiteModuleComposition(t *testing.T) {
	handler := Handler{FontsEnabled: true}
	want := map[string]bool{"GET /admin/site/seo": false, "POST /admin/site/seo": false, "GET /admin/site/fonts": false, "POST /admin/site/fonts": false, "GET /sitemap.xml": false}
	for _, route := range handler.Routes() {
		if _, ok := want[route.Pattern]; ok {
			want[route.Pattern] = true
		}
	}
	for pattern, found := range want {
		if !found {
			t.Errorf("missing SEO route %q", pattern)
		}
	}
}

func TestFontPickerRoutesStayAbsentWithoutPages(t *testing.T) {
	for _, route := range (Handler{}).Routes() {
		if strings.Contains(route.Pattern, "/admin/site/fonts") {
			t.Fatalf("inactive Pages font picker route registered: %s", route.Pattern)
		}
	}
}

func TestFormRoutesAreAbsentWhenFormsAreDisabled(t *testing.T) {
	for _, route := range (Handler{}).Routes() {
		if strings.Contains(route.Pattern, "/forms") || strings.Contains(route.Pattern, "/inbox") {
			t.Fatalf("inactive Forms route registered: %s", route.Pattern)
		}
	}
}

func TestPublicSiteRoutesExposeOnlyPublicProjection(t *testing.T) {
	handler := Handler{DB: fakeExecutor{}, Store: fakeStore{}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, route := range handler.Routes() {
			if route.Pattern == "GET /site/settings" {
				route.Handler.ServeHTTP(w, r)
				return
			}
		}
		t.Fatal("settings route not found")
	}))
	defer server.Close()
	response, err := http.Get(server.URL + "/site/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
}

func TestPublicSiteRejectsTypedNilExecutorAsUnavailable(t *testing.T) {
	var db *fakeExecutor
	handler := Handler{DB: db, Store: fakeStore{}}
	req := httptest.NewRequest(http.MethodGet, "/site/settings", nil)
	res := httptest.NewRecorder()
	handler.publicSettings(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("typed nil executor status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}

func TestPublicSiteRejectsTypedNilStoreAsUnavailable(t *testing.T) {
	var store *fakeStore
	handler := Handler{DB: fakeExecutor{}, Store: store}
	req := httptest.NewRequest(http.MethodGet, "/site/settings", nil)
	res := httptest.NewRecorder()
	handler.publicSettings(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("typed nil store status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}

func TestSitemapRouteUsesPublishedProvider(t *testing.T) {
	handler := Handler{SitemapOrigin: "https://example.com", SitemapProvider: func(context.Context) ([]site.SitemapEntry, error) {
		return []site.SitemapEntry{{URL: "https://example.com/en/home"}}, nil
	}}
	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	res := httptest.NewRecorder()
	handler.sitemap(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "https://example.com/en/home") {
		t.Fatalf("sitemap response = %d %s", res.Code, res.Body.String())
	}
}

func TestStreamingSitemapProducesCategoryIndexAndArticleParts(t *testing.T) {
	handler := Handler{
		SitemapOrigin:     "https://example.com",
		SitemapCategories: []string{"pages", "articles"},
		SitemapStream: func(_ context.Context, category string, visit func(site.SitemapEntry) error) error {
			if category == "pages" {
				return visit(site.SitemapEntry{URL: "https://example.com/en/home", Category: category})
			}
			if category == "articles" {
				return visit(site.SitemapEntry{URL: "https://example.com/public/news/en/story", Category: category})
			}
			return nil
		},
	}
	indexResponse := httptest.NewRecorder()
	handler.sitemap(indexResponse, httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil))
	if indexResponse.Code != http.StatusOK || !strings.Contains(indexResponse.Body.String(), "/sitemap-pages.xml?part=1") || !strings.Contains(indexResponse.Body.String(), "/sitemap-articles.xml?part=1") {
		t.Fatalf("categorized sitemap index = %d %s", indexResponse.Code, indexResponse.Body.String())
	}
	articleResponse := httptest.NewRecorder()
	handler.sitemapArticlePart(articleResponse, httptest.NewRequest(http.MethodGet, "/sitemap-articles.xml?part=1", nil))
	if articleResponse.Code != http.StatusOK || !strings.Contains(articleResponse.Body.String(), "public/news/en/story") {
		t.Fatalf("article sitemap part = %d %s", articleResponse.Code, articleResponse.Body.String())
	}
}

func TestAdminSiteMutationRequiresAdminContext(t *testing.T) {
	handler := Handler{Service: &site.Service{}}
	req := httptest.NewRequest(http.MethodPost, "/admin/site/settings", strings.NewReader(`{"key":"site.name","value":"x"}`))
	res := httptest.NewRecorder()
	handler.saveSetting(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("unauthorized mutation status = %d", res.Code)
	}
}

func TestFormSubmissionRejectsUnconfiguredForm(t *testing.T) {
	handler := Handler{Service: &site.Service{}, AllowedForms: map[string]struct{}{"contact": {}}}
	req := httptest.NewRequest(http.MethodPost, "/forms/unknown/submit", strings.NewReader("email=a%40b.test&message=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	handler.submitForm(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("unknown form status = %d", res.Code)
	}
}
