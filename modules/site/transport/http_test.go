package transport

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/testhttp"
	"time"

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

func TestSocialLinkEditorRoutesArePresent(t *testing.T) {
	got := map[string]bool{}
	for _, route := range (Handler{}).Routes() {
		got[route.Pattern] = true
	}
	for _, pattern := range []string{"GET /admin/site/social-links", "POST /admin/site/social-links"} {
		if !got[pattern] {
			t.Errorf("missing social profile route %q", pattern)
		}
	}
}

func TestInboxXLSXExportIsAvailableToSiteAdmins(t *testing.T) {
	handler := Handler{Store: fakeStore{}, DB: fakeExecutor{}}
	request := httptest.NewRequest(http.MethodGet, "/admin/site/inbox/contact/export.xlsx", nil)
	request.SetPathValue("form", "contact")
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response := httptest.NewRecorder()
	handler.exportInboxXLSX(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Content-Type"), "spreadsheetml") {
		t.Fatalf("export status=%d content-type=%q", response.Code, response.Header().Get("Content-Type"))
	}
	if _, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len())); err != nil {
		t.Fatalf("invalid XLSX response: %v", err)
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

func TestAppearancePageOffersAccessibleLiveContrastFeedback(t *testing.T) {
	handler := Handler{Store: fakeStore{}, Service: &site.Service{}}
	request := httptest.NewRequest(http.MethodGet, "/admin/site/appearance", nil)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response := httptest.NewRecorder()
	handler.appearance(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	for _, marker := range []string{"/admin/site/appearance", `type="color"`, `aria-live="polite"`, "4,5:1", "WCAG AAA", "appearance-contrast", "Canlı site görünümü önizlemesi", "appearance-preview-logo", "media-picker-open", "data-media-target=\"appearance-logo\"", "appearance-login-background", "appearance-login-overlay", `type="range"`} {
		if !strings.Contains(response.Body.String(), marker) {
			t.Errorf("appearance page missing %q", marker)
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

func TestSiteAppearanceAdminRoutePersistsOnlyAuthorizedCSRFMutation(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-appearance-route.sqlite"))
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
		if route.Pattern == "POST /admin/site/appearance" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	values := "primary_color=%23155EEF&accent_color=%23155EEF&dark_primary_color=%23155EEF&dark_accent_color=%23155EEF&font_family=inter&logo_url=%2Fmedia%2Fbrand.svg&dark_logo_url=%2Fmedia%2Fbrand-dark.svg&favicon_url=%2Fmedia%2Ffavicon.svg&social_image_url=%2Fmedia%2Fsocial.webp&login_background_url=%2Fmedia%2Flogin.webp&login_overlay_opacity=40"
	post := func(user *auth.User, token string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/admin/site/appearance", strings.NewReader(values+"&_csrf="+token))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	assertNoAppearance := func() {
		t.Helper()
		_, found, err := store.GetSetting(context.Background(), db, "appearance.primary_color")
		if err != nil {
			t.Fatal(err)
		}
		if found {
			t.Fatal("denied mutation persisted primary color")
		}
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	if response := post(admin, "wrong-token"); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d, want 403", response.Code)
	}
	assertNoAppearance()
	deniedAdmin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if response := post(deniedAdmin, "valid-token"); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin status=%d, want 403", response.Code)
	}
	assertNoAppearance()
	response := post(admin, "valid-token")
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/site/appearance" {
		t.Fatalf("authorized appearance response=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	settings, err := store.ListPublicSettings(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	want := site.Appearance{PrimaryColor: "#155EEF", AccentColor: "#155EEF", DarkPrimaryColor: "#155EEF", DarkAccentColor: "#155EEF", FontFamily: "inter", LogoURL: "/media/brand.svg", DarkLogoURL: "/media/brand-dark.svg", FaviconURL: "/media/favicon.svg", SocialImageURL: "/media/social.webp", LoginBackgroundURL: "/media/login.webp", LoginOverlayOpacity: .4}
	if got := site.AppearanceFromSettings(settings); got != want {
		t.Fatalf("persisted appearance=%+v, want %+v", got, want)
	}
}

func TestSiteSEOAdminRoutePersistsValidatedPathsOnlyForAuthorizedUser(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-seo-route.sqlite"))
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
		if route.Pattern == "POST /admin/site/seo" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	post := func(user *auth.User, token, paths string) *httptest.ResponseRecorder {
		t.Helper()
		body := "_csrf=" + token + "&paths=" + strings.ReplaceAll(url.QueryEscape(paths), "+", "%20")
		request := httptest.NewRequest(http.MethodPost, "/admin/site/seo", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	assertValue := func(want string) {
		t.Helper()
		setting, found, err := store.GetSetting(context.Background(), db, "seo.llms_excluded_paths")
		if err != nil {
			t.Fatal(err)
		}
		if want == "" {
			if found {
				t.Fatalf("unexpected SEO setting persisted: %+v", setting)
			}
			return
		}
		if !found || setting.Value != want {
			t.Fatalf("persisted SEO paths=%q found=%t, want %q", setting.Value, found, want)
		}
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	if response := post(admin, "wrong-token", "/private"); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d, want 403", response.Code)
	}
	assertValue("")
	deniedAdmin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if response := post(deniedAdmin, "valid-token", "/private"); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin status=%d, want 403", response.Code)
	}
	assertValue("")
	if response := post(admin, "valid-token", "/private\n/archive\n/private"); response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/site/seo" {
		t.Fatalf("authorized SEO response=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	assertValue("/private\n/archive")
	if response := post(admin, "valid-token", "/private?leak=yes"); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid path status=%d, want 400", response.Code)
	}
	assertValue("/private\n/archive")
}

func TestSiteRedirectAdminRoutePersistsOnlyAuthorizedCSRFMutation(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-redirect-route.sqlite"))
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
		if route.Pattern == "POST /admin/site/redirects" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	post := func(user *auth.User, token, body string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/admin/site/redirects", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	assertMissing := func() {
		t.Helper()
		if _, err := store.GetRedirect(context.Background(), db, "/old"); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("denied redirect exists or lookup failed: %v", err)
		}
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	const body = `{"SourcePath":"/old","TargetPath":"/new","Status":301}`
	if response := post(admin, "wrong-token", body); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d, want 403", response.Code)
	}
	assertMissing()
	deniedAdmin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if response := post(deniedAdmin, "valid-token", body); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin status=%d, want 403", response.Code)
	}
	assertMissing()
	if response := post(admin, "valid-token", body); response.Code != http.StatusNoContent {
		t.Fatalf("authorized redirect status=%d body=%s", response.Code, response.Body.String())
	}
	redirect, err := store.GetRedirect(context.Background(), db, "/old")
	if err != nil || redirect.SourcePath != "/old" || redirect.TargetPath != "/new" || redirect.Status != http.StatusMovedPermanently {
		t.Fatalf("persisted redirect=%+v err=%v", redirect, err)
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

func TestSiteSocialLinksAdminRouteValidatesAndPersistsAuthorizedProfile(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-social-route.sqlite"))
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
		if route.Pattern == "POST /admin/site/social-links" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	post := func(user *auth.User, token, instagram string) *httptest.ResponseRecorder {
		t.Helper()
		body := "_csrf=" + token + "&social_instagram=" + instagram
		request := httptest.NewRequest(http.MethodPost, "/admin/site/social-links", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	assertProfiles := func(want string) {
		t.Helper()
		setting, found, err := store.GetSetting(context.Background(), db, "site.social_profiles")
		if err != nil {
			t.Fatal(err)
		}
		if want == "" {
			if found {
				t.Fatalf("unexpected social profile setting: %+v", setting)
			}
			return
		}
		if !found || setting.Value != want {
			t.Fatalf("stored social profiles=%q found=%t, want %q", setting.Value, found, want)
		}
		profiles := site.SocialProfilesFromSettings([]site.Setting{setting})
		if len(profiles) != 1 || profiles[0] != (site.SocialProfile{Platform: "instagram", URL: "https://instagram.com/hydrogo"}) {
			t.Fatalf("public social profile projection=%+v", profiles)
		}
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	if response := post(admin, "wrong-token", "https://instagram.com/hydrogo"); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d, want 403", response.Code)
	}
	assertProfiles("")
	deniedAdmin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if response := post(deniedAdmin, "valid-token", "https://instagram.com/hydrogo"); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin status=%d, want 403", response.Code)
	}
	assertProfiles("")
	if response := post(admin, "valid-token", "https://instagram.com/hydrogo"); response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/site/social-links?saved=1" {
		t.Fatalf("authorized social link response=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	assertProfiles(`[{"Platform":"instagram","URL":"https://instagram.com/hydrogo"}]`)
	if response := post(admin, "valid-token", "https://instagram.com.evil.test/hydrogo"); response.Code != http.StatusBadRequest {
		t.Fatalf("untrusted social host status=%d, want 400", response.Code)
	}
	assertProfiles(`[{"Platform":"instagram","URL":"https://instagram.com/hydrogo"}]`)
}

func TestSiteContactWidgetAdminRoutePersistsValidatedSettings(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-contact-widget-route.sqlite"))
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
		if route.Pattern == "POST /admin/appearance/contact-widget" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	post := func(user *auth.User, token, phone string) *httptest.ResponseRecorder {
		t.Helper()
		body := "_csrf=" + token + "&enabled=true&phone=" + phone + "&whatsapp_number=905551234567&message=Contact+us&position=bottom-left&mobile_only=true"
		request := httptest.NewRequest(http.MethodPost, "/admin/appearance/contact-widget", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	assertWidget := func(want site.ContactWidget) {
		t.Helper()
		settings, err := store.ListPublicSettings(context.Background(), db)
		if err != nil {
			t.Fatal(err)
		}
		if got := site.ContactWidgetFromSettings(settings); got != want {
			t.Fatalf("persisted contact widget=%+v, want %+v", got, want)
		}
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	if response := post(admin, "wrong-token", "905551234567"); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d, want 403", response.Code)
	}
	assertWidget(site.ContactWidget{Position: "bottom-right"})
	deniedAdmin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if response := post(deniedAdmin, "valid-token", "905551234567"); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin status=%d, want 403", response.Code)
	}
	assertWidget(site.ContactWidget{Position: "bottom-right"})
	if response := post(admin, "valid-token", "905551234567"); response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/appearance/contact-widget?saved=1" {
		t.Fatalf("authorized contact settings response=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	want := site.ContactWidget{Enabled: true, Phone: "905551234567", WhatsAppNumber: "905551234567", Message: "Contact us", Position: "bottom-left", MobileOnly: true}
	assertWidget(want)
	if response := post(admin, "valid-token", "invalid"); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid phone status=%d, want 400", response.Code)
	}
	assertWidget(want)
}

func TestSiteMaintenanceAdminRouteProtectsAndPersistsPrivateSettings(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-maintenance-route.sqlite"))
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
		if route.Pattern == "POST /admin/site/maintenance" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	post := func(user *auth.User, token string, form url.Values) *httptest.ResponseRecorder {
		t.Helper()
		form.Set("_csrf", token)
		request := httptest.NewRequest(http.MethodPost, "/admin/site/maintenance", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	assertMissing := func() {
		t.Helper()
		if _, found, err := store.GetSetting(context.Background(), db, site.MaintenanceSettingKey); err != nil || found {
			t.Fatalf("denied mutation persisted private maintenance setting: found=%t err=%v", found, err)
		}
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	form := url.Values{"enabled": {"true"}, "allowlist": {"127.0.0.1\n10.0.0.0/8"}, "headline": {"Kısa bakım"}, "message": {"Birazdan döneceğiz."}, "starts_at": {"2026-10-10T09:00"}, "ends_at": {"2026-10-10T12:00"}, "preview_key": {"sixteen-character-key"}}
	if response := post(admin, "wrong-token", form); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d, want 403", response.Code)
	}
	assertMissing()
	deniedAdmin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if response := post(deniedAdmin, "valid-token", form); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin status=%d, want 403", response.Code)
	}
	assertMissing()
	if response := post(admin, "valid-token", form); response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/site/maintenance?saved=1" {
		t.Fatalf("authorized maintenance response=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	setting, found, err := store.GetSetting(context.Background(), db, site.MaintenanceSettingKey)
	if err != nil || !found || setting.Public {
		t.Fatalf("maintenance setting=%+v found=%t err=%v", setting, found, err)
	}
	want := site.MaintenanceSettings{Enabled: true, Allowlist: []string{"127.0.0.1", "10.0.0.0/8"}, StartsAt: "2026-10-10T09:00:00Z", EndsAt: "2026-10-10T12:00:00Z", Headline: "Kısa bakım", Message: "Birazdan döneceğiz.", PreviewKeyHash: site.MaintenancePreviewKeyHash("sixteen-character-key")}
	if got := site.MaintenanceFromSetting(setting); !reflect.DeepEqual(got, want) || strings.Contains(setting.Value, "sixteen-character-key") {
		t.Fatalf("private maintenance settings=%+v want=%+v serialized=%s", got, want, setting.Value)
	}
	invalid := url.Values{"enabled": {"true"}, "allowlist": {"not-an-ip"}, "headline": {"Invalid"}, "message": {"Invalid"}}
	if response := post(admin, "valid-token", invalid); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid allowlist status=%d, want 400", response.Code)
	}
	unchanged, found, err := store.GetSetting(context.Background(), db, site.MaintenanceSettingKey)
	if err != nil || !found || unchanged.Value != setting.Value {
		t.Fatalf("invalid update changed maintenance settings: found=%t err=%v value=%s", found, err, unchanged.Value)
	}
}

func TestSiteCustomErrorPagesAdminSaveRendersPersistedPublicResponses(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-error-page-route.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	handler := Handler{DB: db, Store: store, Service: &site.Service{Transactions: database.NewTransactor(db), Store: store}}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		if route.Pattern == "POST /admin/site/error-pages/{status}" || route.Pattern == "GET /{path...}" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	form := url.Values{"headline": {"Sayfa bulunamadı"}, "message": {"İçerik taşınmış olabilir."}, "action_label": {"Ana sayfaya dön"}, "action_url": {"/"}, "links": {"Belgeler | /docs"}}
	post := func(user *auth.User, token, status string, values url.Values) *httptest.ResponseRecorder {
		t.Helper()
		body := url.Values{}
		for key, entries := range values {
			for _, value := range entries {
				body.Add(key, value)
			}
		}
		body.Set("_csrf", token)
		request := httptest.NewRequest(http.MethodPost, "/admin/site/error-pages/"+status, strings.NewReader(body.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	assertMissing := func() {
		t.Helper()
		if _, found, err := store.GetSetting(context.Background(), db, "site.error_page.404"); err != nil || found {
			t.Fatalf("unauthorized error page exists: found=%t err=%v", found, err)
		}
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	if response := post(admin, "wrong-token", "404", form); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d, want 403", response.Code)
	}
	assertMissing()
	deniedAdmin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if response := post(deniedAdmin, "valid-token", "404", form); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin status=%d, want 403", response.Code)
	}
	assertMissing()
	if response := post(admin, "valid-token", "404", form); response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/site/error-pages?saved=1" {
		t.Fatalf("authorized error page response=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	request := httptest.NewRequest(http.MethodGet, "/missing-page", nil)
	public := httptest.NewRecorder()
	secured.ServeHTTP(public, request)
	if public.Code != http.StatusNotFound || !strings.Contains(public.Body.String(), "Sayfa bulunamadı") || !strings.Contains(public.Body.String(), "İçerik taşınmış olabilir.") || !strings.Contains(public.Body.String(), `href="/docs"`) {
		t.Fatalf("public custom 404 status=%d body=%s", public.Code, public.Body.String())
	}
	invalid := url.Values{"headline": {"Invalid"}, "message": {"Invalid"}, "links": {"External | https://attacker.test"}}
	if response := post(admin, "valid-token", "404", invalid); response.Code != http.StatusBadRequest {
		t.Fatalf("external custom error link status=%d, want 400", response.Code)
	}
	public = httptest.NewRecorder()
	secured.ServeHTTP(public, httptest.NewRequest(http.MethodGet, "/missing-page", nil))
	if !strings.Contains(public.Body.String(), "Sayfa bulunamadı") {
		t.Fatalf("invalid update replaced public error page: %s", public.Body.String())
	}
	serverError := url.Values{"headline": {"Geçici bir sorun oluştu"}, "message": {"Ekibimiz sorunu inceliyor."}, "action_label": {"Yeniden dene"}, "action_url": {"/"}, "links": {"Destek | /contact"}}
	if response := post(admin, "valid-token", "500", serverError); response.Code != http.StatusSeeOther {
		t.Fatalf("authorized custom 500 save status=%d body=%s", response.Code, response.Body.String())
	}
	public = httptest.NewRecorder()
	handler.renderPublicErrorPage(public, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusInternalServerError)
	if public.Code != http.StatusInternalServerError || !strings.Contains(public.Body.String(), "Geçici bir sorun oluştu") || !strings.Contains(public.Body.String(), "Ekibimiz sorunu inceliyor.") || !strings.Contains(public.Body.String(), `href="/contact"`) {
		t.Fatalf("public custom 500 status=%d body=%s", public.Code, public.Body.String())
	}
}

func TestSiteAdminMutationRoutesEnforceCSRFAndSiteManage(t *testing.T) {
	handler := Handler{FormsEnabled: true, FontsEnabled: true}
	mux := http.NewServeMux()
	var mutations []string
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
		method, path, ok := strings.Cut(route.Pattern, " ")
		if ok && method == http.MethodPost && strings.HasPrefix(path, "/admin/") {
			mutations = append(mutations, route.Pattern)
		}
	}
	if len(mutations) < 20 {
		t.Fatalf("registered %d Site Admin mutations, want broad route coverage", len(mutations))
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	deniedUsers := []struct {
		name string
		user *auth.User
	}{
		{name: "editor default", user: &auth.User{Role: auth.RoleEditor, Active: true}},
		{name: "viewer default", user: &auth.User{Role: auth.RoleViewer, Active: true}},
		{name: "admin explicit deny", user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}},
		{name: "inactive explicit allow", user: &auth.User{Role: auth.RoleAdmin, Active: false, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}}},
	}
	for _, pattern := range mutations {
		method, path, _ := strings.Cut(pattern, " ")
		path = strings.NewReplacer("{id}", "test-id", "{locale}", "tr", "{form}", "contact", "{status}", "404").Replace(path)
		t.Run(pattern+"/invalid-csrf", func(t *testing.T) {
			request := httptest.NewRequest(method, path, strings.NewReader("_csrf=wrong-token"))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
			request.Header.Set(security.CSRFHeaderName, "wrong-token")
			request = request.WithContext(auth.WithUser(request.Context(), &auth.User{Role: auth.RoleAdmin, Active: true}))
			response := httptest.NewRecorder()
			secured.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("invalid CSRF status=%d, want 403; body=%s", response.Code, response.Body.String())
			}
		})
		for _, tc := range deniedUsers {
			t.Run(pattern+"/"+tc.name, func(t *testing.T) {
				request := httptest.NewRequest(method, path, strings.NewReader("_csrf=valid-token"))
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
				request.Header.Set(security.CSRFHeaderName, "valid-token")
				request = request.WithContext(auth.WithUser(request.Context(), tc.user))
				response := httptest.NewRecorder()
				secured.ServeHTTP(response, request)
				if response.Code != http.StatusForbidden {
					t.Fatalf("%s status=%d, want 403; body=%s", tc.name, response.Code, response.Body.String())
				}
			})
		}
	}
}

func TestOptionalSiteAssistantMutationEnforcesCSRFAndSiteManage(t *testing.T) {
	handler := Handler{
		FormsEnabled: true,
		Assistant:    ai.Assistant{Enabled: true, Provider: &aiSummaryProvider{}},
		AIAudit:      &aiAuditRecorder{},
	}
	mux := http.NewServeMux()
	registered := false
	for _, route := range handler.Routes() {
		if route.Pattern == "POST /admin/site/inbox/{form}/ai-summarize" {
			mux.Handle(route.Pattern, route.Handler)
			registered = true
			break
		}
	}
	if !registered {
		t.Fatal("enabled Site assistant mutation was not registered")
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	serve := func(user *auth.User, token string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/admin/site/inbox/contact/ai-summarize", strings.NewReader("submission_id=item&consent=yes&_csrf="+token))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}

	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	if response := serve(admin, "wrong-token"); response.Code != http.StatusForbidden {
		t.Fatalf("assistant invalid CSRF status=%d, want 403", response.Code)
	}
	for _, tc := range []struct {
		name string
		user *auth.User
	}{
		{name: "editor", user: &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}},
		{name: "viewer", user: &auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}},
		{name: "admin permission deny", user: &auth.User{ID: "admin-denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}},
		{name: "inactive explicit grant", user: &auth.User{ID: "inactive", Role: auth.RoleAdmin, Active: false, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if response := serve(tc.user, "valid-token"); response.Code != http.StatusForbidden {
				t.Fatalf("assistant denied status=%d, want 403; body=%s", response.Code, response.Body.String())
			}
		})
	}
}

type retryMailSpool struct {
	retried []string
	items   []mailer.SpoolEntry
	status  string
	limit   int
}

func (s *retryMailSpool) List(_ context.Context, status string, limit int) ([]mailer.SpoolEntry, error) {
	s.status, s.limit = status, limit
	if status == "" {
		return s.items, nil
	}
	var filtered []mailer.SpoolEntry
	for _, item := range s.items {
		if item.Status == status {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (s *retryMailSpool) RetryNow(_ context.Context, id string) error {
	s.retried = append(s.retried, id)
	return nil
}

func TestSiteMailQueueUsesAdminShellAndRendersSafeFilteredRows(t *testing.T) {
	spool := &retryMailSpool{items: []mailer.SpoolEntry{
		{ID: "failed/1", Message: mailer.Message{To: "person@example.test", Subject: `<script>alert("x")</script>`}, Status: mailer.SpoolFailedPermanent, Attempts: 3, LastError: "secret provider detail"},
		{ID: "sent-1", Message: mailer.Message{To: "other@example.test", Subject: "Delivered"}, Status: mailer.SpoolDelivered, Attempts: 1},
	}}
	handler := Handler{FormsEnabled: true, MailSpool: spool}
	var route http.Handler
	for _, candidate := range handler.Routes() {
		if candidate.Pattern == "GET /admin/site/mail" {
			route = candidate.Handler
			break
		}
	}
	if route == nil {
		t.Fatal("mail queue route is missing")
	}
	request := httptest.NewRequest(http.MethodGet, "/admin/site/mail?status=failed_permanent", nil)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response := httptest.NewRecorder()
	security.CSRFMiddleware(false, nil)(route).ServeHTTP(response, request)
	body := response.Body.String()
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, body)
	}
	for _, marker := range []string{"<title>E-posta kuyruğu · HydroGo Studio", "E-posta kuyruğu", "Tüm durumlar", "Kalıcı hata", "person@example.test", "&lt;script&gt;", "/admin/site/mail/failed%2F1/retry", "name=\"_csrf\""} {
		if !strings.Contains(body, marker) {
			t.Errorf("mail queue missing %q", marker)
		}
	}
	for _, marker := range []string{"<script>alert", "secret provider detail", "other@example.test", "TenantID"} {
		if strings.Contains(body, marker) {
			t.Errorf("mail queue leaked/unfiltered value %q", marker)
		}
	}
	if spool.status != mailer.SpoolFailedPermanent || spool.limit != 100 {
		t.Fatalf("list query status=%q limit=%d", spool.status, spool.limit)
	}
}

func TestSiteMailQueueEmptyStateAndAccessControl(t *testing.T) {
	spool := &retryMailSpool{}
	handler := Handler{FormsEnabled: true, MailSpool: spool}
	var route http.Handler
	for _, candidate := range handler.Routes() {
		if candidate.Pattern == "GET /admin/site/mail" {
			route = candidate.Handler
		}
	}
	if route == nil {
		t.Fatal("mail queue route is missing")
	}
	request := httptest.NewRequest(http.MethodGet, "/admin/site/mail", nil)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response := httptest.NewRecorder()
	security.CSRFMiddleware(false, nil)(route).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Bu filtrede ileti bulunamadı.") {
		t.Fatalf("empty queue response=%d %s", response.Code, response.Body.String())
	}
	deniedRequest := httptest.NewRequest(http.MethodGet, "/admin/site/mail", nil)
	deniedRequest = deniedRequest.WithContext(auth.WithUser(deniedRequest.Context(), &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}))
	denied := httptest.NewRecorder()
	route.ServeHTTP(denied, deniedRequest)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("unauthorized queue status=%d", denied.Code)
	}
	invalidRequest := httptest.NewRequest(http.MethodGet, "/admin/site/mail?status=internal-error", nil)
	invalidRequest = invalidRequest.WithContext(auth.WithUser(invalidRequest.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	invalid := httptest.NewRecorder()
	route.ServeHTTP(invalid, invalidRequest)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid filter status=%d, want 400", invalid.Code)
	}
}

func TestSiteMailRetryRequiresPermissionAndPersistsOnlyAuthorizedAction(t *testing.T) {
	spool := &retryMailSpool{}
	handler := Handler{FormsEnabled: true, MailSpool: spool}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		if route.Pattern == "POST /admin/site/mail/{id}/retry" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	serve := func(user *auth.User, token string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/admin/site/mail/message-1/retry", strings.NewReader("_csrf="+token))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	denied := []struct {
		name  string
		user  *auth.User
		token string
	}{
		{name: "invalid csrf", user: &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}, token: "wrong-token"},
		{name: "editor", user: &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, token: "valid-token"},
		{name: "viewer", user: &auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}, token: "valid-token"},
		{name: "admin explicit deny", user: &auth.User{ID: "admin-denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}, token: "valid-token"},
	}
	for _, tc := range denied {
		t.Run(tc.name, func(t *testing.T) {
			if response := serve(tc.user, tc.token); response.Code != http.StatusForbidden {
				t.Fatalf("mail retry status=%d, want 403; body=%s", response.Code, response.Body.String())
			}
			if len(spool.retried) != 0 {
				t.Fatalf("denied request retried messages: %v", spool.retried)
			}
		})
	}
	if response := serve(&auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}, "valid-token"); response.Code != http.StatusSeeOther {
		t.Fatalf("authorized mail retry status=%d, want 303; body=%s", response.Code, response.Body.String())
	}
	if len(spool.retried) != 1 || spool.retried[0] != "message-1" {
		t.Fatalf("authorized retry calls=%v, want [message-1]", spool.retried)
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
