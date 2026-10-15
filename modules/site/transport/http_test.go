package transport

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/testhttp"
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

func routeHandler(t *testing.T, handler Handler, pattern string) http.Handler {
	t.Helper()
	for _, route := range handler.Routes() {
		if route.Pattern == pattern {
			return route.Handler
		}
	}
	t.Fatalf("route %q not registered", pattern)
	return nil
}

func TestPublicSiteRoutesExposeOnlyPublicProjection(t *testing.T) {
	handler := Handler{DB: fakeExecutor{}, Store: fakeStore{}}
	const origin = "http://site.test"
	client := testhttp.Client(testhttp.Handlers{"site.test": routeHandler(t, handler, "GET /api/v1/site/settings")})
	response, err := client.Get(origin + "/api/v1/site/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	var body struct {
		Data map[string]string `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil || body.Data["site.name"] != "HydroGo" {
		t.Fatalf("body = %+v err=%v", body, err)
	}
}

func TestPublicSiteRejectsTypedNilExecutorAsUnavailable(t *testing.T) {
	var nilDB *sql.DB
	handler := Handler{DB: nilDB, Store: fakeStore{}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/site/settings", nil)
	res := httptest.NewRecorder()
	handler.publicSettings(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("typed nil executor status = %d", res.Code)
	}
}

func TestPublicSiteRejectsTypedNilStoreAsUnavailable(t *testing.T) {
	var nilStore *sitesql.Store
	handler := Handler{DB: fakeExecutor{}, Store: nilStore}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/site/settings", nil)
	res := httptest.NewRecorder()
	handler.publicSettings(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("typed nil store status = %d", res.Code)
	}
}

func TestSitemapRouteUsesPublishedProvider(t *testing.T) {
	handler := Handler{SitemapOrigin: "https://example.com", SitemapProvider: func(context.Context) ([]site.SitemapEntry, error) {
		return []site.SitemapEntry{{URL: "https://example.com/tr/about", LastModified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}, nil
	}}
	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	res := httptest.NewRecorder()
	handler.sitemap(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "https://example.com/tr/about") {
		t.Fatalf("sitemap status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestStreamingSitemapProducesCategoryIndexAndArticleParts(t *testing.T) {
	handler := Handler{SitemapOrigin: "https://example.com", SitemapCategories: []string{"pages", "articles"}, SitemapStream: func(_ context.Context, category string, visit func(site.SitemapEntry) error) error {
		for i := 0; i < 2; i++ {
			if err := visit(site.SitemapEntry{URL: "https://example.com/" + category + "/" + string(rune('a'+i)), LastModified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Category: category}); err != nil {
				return err
			}
		}
		return nil
	}}
	index := httptest.NewRecorder()
	handler.sitemapIndex(index, httptest.NewRequest(http.MethodGet, "/sitemap-index.xml", nil))
	if index.Code != http.StatusOK || !strings.Contains(index.Body.String(), "sitemap-articles.xml") {
		t.Fatalf("index status=%d body=%s", index.Code, index.Body.String())
	}
	articles := httptest.NewRecorder()
	handler.sitemapArticlePart(articles, httptest.NewRequest(http.MethodGet, "/sitemap-articles.xml", nil))
	if articles.Code != http.StatusOK || !strings.Contains(articles.Body.String(), "https://example.com/articles/a") {
		t.Fatalf("articles status=%d body=%s", articles.Code, articles.Body.String())
	}
}

func TestAdminSiteMutationRequiresAdminContext(t *testing.T) {
	handler := Handler{Service: &site.Service{}}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/site/settings", strings.NewReader(`{"settings":[{"key":"site.name","value":"x"}]}`))
	res := httptest.NewRecorder()
	handler.adminSaveSettings(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("unauthorized mutation status = %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"code":"forbidden"`) {
		t.Fatalf("body = %s", res.Body.String())
	}
}

// openSiteDB returns a migrated SQLite database and a transport handler on it.
func openSiteDB(t *testing.T) (*sql.DB, Handler, sitesql.Store) {
	t.Helper()
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	handler := Handler{DB: db, Store: store, Service: &site.Service{Transactions: database.NewTransactor(db), Store: store}, Redirects: &site.RedirectRegistry{}, NavigationLocales: []string{"tr", "en"}}
	return db, handler, store
}

func TestAdminSiteSettingUsesSiteManageOverrides(t *testing.T) {
	_, handler, store := openSiteDB(t)
	_ = store
	for _, tc := range []struct {
		name string
		user *auth.User
		want int
	}{
		{name: "editor default denied", user: &auth.User{Role: auth.RoleEditor, Active: true}, want: http.StatusForbidden},
		{name: "editor deny override", user: &auth.User{Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}, want: http.StatusForbidden},
		{name: "viewer allow override", user: &auth.User{Role: auth.RoleViewer, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}}, want: http.StatusOK},
		{name: "admin deny override", user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}, want: http.StatusForbidden},
		{name: "inactive grant denied", user: &auth.User{Role: auth.RoleAdmin, Active: false, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}}, want: http.StatusForbidden},
		{name: "admin default allowed", user: &auth.User{Role: auth.RoleAdmin, Active: true}, want: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/site/settings", strings.NewReader(`{"settings":[{"key":"site.name","value":"HydroGo"}]}`))
			req = req.WithContext(auth.WithUser(req.Context(), tc.user))
			res := httptest.NewRecorder()
			handler.adminSaveSettings(res, req)
			if res.Code != tc.want {
				t.Fatalf("PUT settings status=%d want=%d body=%s", res.Code, tc.want, res.Body.String())
			}
		})
	}
}

func TestAdminSiteSettingRouteEnforcesCSRFBeforePermission(t *testing.T) {
	db, handler, store := openSiteDB(t)
	mux := http.NewServeMux()
	mux.Handle("PUT /api/v1/admin/site/settings", routeHandler(t, handler, "PUT /api/v1/admin/site/settings"))
	secured := security.CSRFMiddleware(false, nil)(mux)
	put := func(user *auth.User, token string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/site/settings", strings.NewReader(`{"settings":[{"key":"site.name","value":"HydroGo"}]}`))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	if response := put(&auth.User{Role: auth.RoleAdmin, Active: true}, "wrong-token"); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d, want 403", response.Code)
	}
	if _, found, err := store.GetSetting(context.Background(), db, "site.name"); err != nil || found {
		t.Fatalf("invalid CSRF changed persisted setting: found=%t err=%v", found, err)
	}
	if response := put(&auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}, "valid-token"); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin status=%d, want 403", response.Code)
	}
	if response := put(&auth.User{Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}}, "valid-token"); response.Code != http.StatusOK {
		t.Fatalf("granted editor status=%d, want 200: %s", response.Code, response.Body.String())
	}
	setting, found, err := store.GetSetting(context.Background(), db, "site.name")
	if err != nil || !found || setting.Value != "HydroGo" {
		t.Fatalf("permitted setting was not persisted: setting=%+v found=%t err=%v", setting, found, err)
	}
}

func TestSiteNavigationAdminRoutePersistsOnlyAuthorizedCSRFMutation(t *testing.T) {
	db, handler, store := openSiteDB(t)
	mux := http.NewServeMux()
	mux.Handle("PUT /api/v1/admin/site/navigation/{locale}", routeHandler(t, handler, "PUT /api/v1/admin/site/navigation/{locale}"))
	secured := security.CSRFMiddleware(false, nil)(mux)
	put := func(user *auth.User, token string) *httptest.ResponseRecorder {
		t.Helper()
		body := `{"items":[{"id":"home","label":"Ana sayfa","href":"/","visible":true},{"id":"private","label":"İç sayfa","href":"/internal","parent_id":"home","visible":false}]}`
		request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/site/navigation/tr", strings.NewReader(body))
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
	if response := put(admin, "wrong-token"); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d, want 403", response.Code)
	}
	assertEmpty()
	deniedAdmin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if response := put(deniedAdmin, "valid-token"); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin status=%d, want 403", response.Code)
	}
	assertEmpty()
	permittedEditor := &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}}
	response := put(permittedEditor, "valid-token")
	if response.Code != http.StatusOK {
		t.Fatalf("permitted Editor status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Data navigationResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || len(body.Data.Items) != 2 || body.Data.Locale != "tr" {
		t.Fatalf("response = %s err=%v", response.Body.String(), err)
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
