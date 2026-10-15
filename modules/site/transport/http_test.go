package transport

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"time"

	"github.com/ardaozyildirim/hydrogo/internal/testhttp"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	sitesql "github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
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

func TestPublicSiteRoutesExposeOnlyPublicProjection(t *testing.T) {
	handler := Handler{DB: fakeExecutor{}, Store: fakeStore{}}
	const origin = "http://site.test"
	client := testhttp.Client(testhttp.Handlers{"site.test": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, route := range handler.Routes() {
			if route.Pattern == "GET /site/settings" {
				route.Handler.ServeHTTP(w, r)
				return
			}
		}
		t.Fatal("settings route not found")
	})})
	response, err := client.Get(origin + "/site/settings")
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

type recordingSettingStore struct {
	fakeStore
	upserts int
}

func (s *recordingSettingStore) UpsertSetting(context.Context, database.Executor, site.Setting) error {
	s.upserts++
	return nil
}

func TestAdminSiteSettingUsesSiteManageOverrides(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-permission.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, tc := range []struct {
		name string
		user *auth.User
		want int
	}{
		{name: "editor default denied", user: &auth.User{Role: auth.RoleEditor, Active: true}, want: http.StatusForbidden},
		{name: "editor deny override", user: &auth.User{Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}, want: http.StatusForbidden},
		{name: "viewer allow override", user: &auth.User{Role: auth.RoleViewer, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}}, want: http.StatusNoContent},
		{name: "admin deny override", user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}, want: http.StatusForbidden},
		{name: "inactive grant denied", user: &auth.User{Role: auth.RoleAdmin, Active: false, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}}, want: http.StatusForbidden},
		{name: "admin default allowed", user: &auth.User{Role: auth.RoleAdmin, Active: true}, want: http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &recordingSettingStore{}
			handler := Handler{Service: &site.Service{Transactions: database.NewTransactor(db), Store: store}}
			req := httptest.NewRequest(http.MethodPost, "/admin/site/settings", strings.NewReader(`{"key":"site.name","value":"HydroGo"}`))
			req = req.WithContext(auth.WithUser(req.Context(), tc.user))
			res := httptest.NewRecorder()
			handler.saveSetting(res, req)
			if res.Code != tc.want {
				t.Fatalf("POST /admin/site/settings status=%d want=%d body=%s", res.Code, tc.want, res.Body.String())
			}
			wantWrites := 0
			if tc.want == http.StatusNoContent {
				wantWrites = 1
			}
			if store.upserts != wantWrites {
				t.Fatalf("setting writes=%d want=%d", store.upserts, wantWrites)
			}
		})
	}
}

func TestAdminSiteSettingRouteEnforcesCSRFBeforePermission(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-setting-csrf.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	handler := Handler{Service: &site.Service{Transactions: database.NewTransactor(db), Store: store}}
	mux := http.NewServeMux()
	registered := false
	for _, route := range handler.Routes() {
		if route.Pattern == "POST /admin/site/settings" {
			mux.Handle(route.Pattern, route.Handler)
			registered = true
			break
		}
	}
	if !registered {
		t.Fatal("POST /admin/site/settings was not registered")
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	post := func(user *auth.User, token string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/admin/site/settings", strings.NewReader(`{"key":"site.name","value":"HydroGo"}`))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}

	if response := post(&auth.User{Role: auth.RoleAdmin, Active: true}, "wrong-token"); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d, want 403", response.Code)
	}
	if _, found, err := store.GetSetting(context.Background(), db, "site.name"); err != nil || found {
		t.Fatalf("invalid CSRF changed persisted setting: found=%t err=%v", found, err)
	}
	if response := post(&auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}, "valid-token"); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin status=%d, want 403", response.Code)
	}
	if _, found, err := store.GetSetting(context.Background(), db, "site.name"); err != nil || found {
		t.Fatalf("denied Admin changed persisted setting: found=%t err=%v", found, err)
	}
	if response := post(&auth.User{Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}}, "valid-token"); response.Code != http.StatusNoContent {
		t.Fatalf("granted editor status=%d, want 204: %s", response.Code, response.Body.String())
	}
	setting, found, err := store.GetSetting(context.Background(), db, "site.name")
	if err != nil || !found || setting.Value != "HydroGo" {
		t.Fatalf("permitted setting was not persisted: setting=%+v found=%t err=%v", setting, found, err)
	}
}

func TestSiteNavigationAdminRoutePersistsOnlyAuthorizedCSRFMutation(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-navigation-route.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	handler := Handler{Service: &site.Service{Transactions: database.NewTransactor(db), Store: store}}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		if route.Pattern == "POST /admin/site/navigation/{locale}" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	post := func(user *auth.User, token string) *httptest.ResponseRecorder {
		t.Helper()
		body := `[{"ID":"home","Label":"Ana sayfa","Href":"/","Position":0,"Visible":true},{"ID":"private","Label":"İç sayfa","Href":"/internal","parent_id":"home","Position":1,"Visible":false}]`
		request := httptest.NewRequest(http.MethodPost, "/admin/site/navigation/tr", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	assertEmpty := func() {
		t.Helper()
		items, err := store.ListAdminNavigation(context.Background(), db, "tr")
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 0 {
			t.Fatalf("denied mutation persisted navigation: %+v", items)
		}
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	if response := post(admin, "wrong-token"); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d, want 403", response.Code)
	}
	assertEmpty()
	deniedAdmin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if response := post(deniedAdmin, "valid-token"); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin status=%d, want 403", response.Code)
	}
	assertEmpty()
	permittedEditor := &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}}
	if response := post(permittedEditor, "valid-token"); response.Code != http.StatusNoContent {
		t.Fatalf("permitted Editor status=%d body=%s", response.Code, response.Body.String())
	}
	adminItems, err := store.ListAdminNavigation(context.Background(), db, "tr")
	if err != nil || len(adminItems) != 2 || adminItems[1].ParentID != "home" || adminItems[1].Target != "_self" || adminItems[1].Visible {
		t.Fatalf("persisted admin navigation=%+v err=%v", adminItems, err)
	}
	publicItems, err := store.ListNavigation(context.Background(), db, "tr")
	if err != nil || len(publicItems) != 1 || publicItems[0].ID != "home" {
		t.Fatalf("public navigation projection=%+v err=%v", publicItems, err)
	}
}
