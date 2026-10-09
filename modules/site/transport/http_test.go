package transport

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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
	"github.com/ardaozyildirim/hydrogo/internal/forms"
	formssql "github.com/ardaozyildirim/hydrogo/internal/forms/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	sitesql "github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
)

type fakeStore struct{}

type siteFontRoundTripper func(*http.Request) (*http.Response, error)

func (roundTrip siteFontRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

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

func TestInboxExportsAreAuthorizedAndSpreadsheetSafe(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-inbox-export.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	service := &site.Service{Transactions: database.NewTransactor(db), Store: store}
	city := `=HYPERLINK("https://evil.test")`
	if err := service.Submit(context.Background(), site.Submission{ID: "export-001", FormKey: "contact", Email: "customer@example.test", Payload: `{"message":"Help"}`, SourceIPHash: strings.Repeat("a", 64), LocationCity: city}); err != nil {
		t.Fatal(err)
	}
	handler := Handler{DB: db, Store: store, Service: service, FormsEnabled: true}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		if strings.Contains(route.Pattern, "/admin/site/inbox/{form}/export.") {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	for _, format := range []string{"csv", "xlsx"} {
		path := "/admin/site/inbox/contact/export." + format
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}))
		denied := httptest.NewRecorder()
		mux.ServeHTTP(denied, request)
		if denied.Code != http.StatusForbidden {
			t.Errorf("Editor %s export status=%d, want 403", format, denied.Code)
		}

		request = httptest.NewRequest(http.MethodGet, path, nil)
		request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("Admin %s export status=%d body=%s", format, response.Code, response.Body.String())
		}
		if !strings.Contains(response.Header().Get("Content-Disposition"), "attachment") {
			t.Errorf("%s export lacks download disposition: %q", format, response.Header().Get("Content-Disposition"))
		}
		if format == "csv" {
			records, err := csv.NewReader(response.Body).ReadAll()
			if err != nil {
				t.Fatalf("parse CSV: %v", err)
			}
			if len(records) != 2 || records[1][4] != "'"+city {
				t.Fatalf("CSV formula escaping/data mismatch: %#v", records)
			}
			continue
		}
		archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
		if err != nil {
			t.Fatalf("open XLSX: %v", err)
		}
		var worksheet []byte
		for _, file := range archive.File {
			if file.Name == "xl/worksheets/sheet1.xml" {
				opened, err := file.Open()
				if err != nil {
					t.Fatal(err)
				}
				worksheet, err = io.ReadAll(opened)
				_ = opened.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
		}
		if len(worksheet) == 0 || bytes.Contains(worksheet, []byte("<f>")) || !bytes.Contains(worksheet, []byte(`t="inlineStr"`)) || !bytes.Contains(worksheet, []byte("=HYPERLINK(&#34;https://evil.test&#34;)")) {
			t.Fatalf("XLSX worksheet should encode formula-like input as inline text: %s", worksheet)
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

func TestFontLocalizationRouteRequiresPermissionAndPersistsDownloadedAssets(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-font-route.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	service := &site.Service{Transactions: database.NewTransactor(db), Store: store}
	fontsDir := filepath.Join(t.TempDir(), "fonts")
	fontCSS := "@font-face{font-family:'Inter';font-style:normal;font-weight:400 700;src:url(https://fonts.gstatic.com/s/inter/v13/test.woff2) format('woff2');}"
	requests := 0
	client := &http.Client{Transport: siteFontRoundTripper(func(request *http.Request) (*http.Response, error) {
		requests++
		body := "font-bytes"
		if request.URL.Host == "fonts.googleapis.com" {
			body = fontCSS
		} else if request.URL.Host != "fonts.gstatic.com" {
			return nil, errors.New("unexpected font host: " + request.URL.Host)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: request}, nil
	})}
	handler := Handler{Service: service, FontsDir: fontsDir, FontsEnabled: true, fontHTTPClient: client}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		if route.Pattern == "POST /admin/site/fonts" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	submit := func(user *auth.User, token, family string) *httptest.ResponseRecorder {
		t.Helper()
		form := url.Values{"_csrf": {token}, "family": {family}}
		request := httptest.NewRequest(http.MethodPost, "/admin/site/fonts", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	assertNoPersistence := func() {
		t.Helper()
		if _, found, err := store.GetSetting(context.Background(), db, "design.font_css"); err != nil || found {
			t.Fatalf("unexpected persisted font setting: found=%t err=%v", found, err)
		}
		if files, err := filepath.Glob(filepath.Join(fontsDir, "*")); err != nil || len(files) != 0 {
			t.Fatalf("unexpected downloaded font files: %v err=%v", files, err)
		}
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	for _, denied := range []struct {
		name  string
		user  *auth.User
		token string
	}{
		{name: "invalid CSRF", user: admin, token: "wrong-token"},
		{name: "Editor", user: &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, token: "valid-token"},
		{name: "Viewer", user: &auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}, token: "valid-token"},
		{name: "permission denied", user: &auth.User{ID: "denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}, token: "valid-token"},
	} {
		if response := submit(denied.user, denied.token, "Inter"); response.Code != http.StatusForbidden {
			t.Fatalf("%s font change status=%d body=%s", denied.name, response.Code, response.Body.String())
		}
		assertNoPersistence()
		if requests != 0 {
			t.Fatalf("denied request contacted font provider %d times", requests)
		}
	}
	if response := submit(admin, "valid-token", "Inter;url(evil)"); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid family status=%d body=%s", response.Code, response.Body.String())
	}
	assertNoPersistence()
	if requests != 0 {
		t.Fatalf("invalid family contacted font provider %d times", requests)
	}
	if response := submit(admin, "valid-token", "Inter"); response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/site/fonts?saved=1" {
		t.Fatalf("authorized font change status=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	setting, found, err := store.GetSetting(context.Background(), db, "design.font_css")
	if err != nil || !found || setting.Value != "/fonts/font-inter.css" || !setting.Public {
		t.Fatalf("persisted font setting=%+v found=%t err=%v", setting, found, err)
	}
	css, err := os.ReadFile(filepath.Join(fontsDir, "font-inter.css"))
	if err != nil || !strings.Contains(string(css), "font-display:swap") || !strings.Contains(string(css), "@font-face") {
		t.Fatalf("localized CSS=%q err=%v", css, err)
	}
	fontFiles, err := filepath.Glob(filepath.Join(fontsDir, "font-*.woff2"))
	if err != nil || len(fontFiles) != 1 {
		t.Fatalf("localized assets=%v err=%v", fontFiles, err)
	}
	data, err := os.ReadFile(fontFiles[0])
	if err != nil || string(data) != "font-bytes" {
		t.Fatalf("localized font=%q err=%v", data, err)
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
	registry := &site.RedirectRegistry{}
	handler := Handler{Service: &site.Service{Transactions: database.NewTransactor(db), Store: store}, Redirects: registry}
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
	const cyclicBody = `{"SourcePath":"/new","TargetPath":"/old","Status":301}`
	if response := post(admin, "valid-token", cyclicBody); response.Code != http.StatusBadRequest {
		t.Fatalf("cyclic redirect status=%d body=%s", response.Code, response.Body.String())
	}
	if _, err := store.GetRedirect(context.Background(), db, "/new"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cyclic redirect was persisted: %v", err)
	}
	if _, found := registry.Match(tenant.Scope{}, "/new"); found {
		t.Fatal("cyclic redirect entered the runtime registry")
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

func TestSiteStaffNoticeAdminRoutePersistsPrivateValidatedNotice(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-staff-notice-route.sqlite"))
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
		if route.Pattern == "POST /admin/site/broadcast" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	post := func(user *auth.User, token string, form url.Values) *httptest.ResponseRecorder {
		t.Helper()
		form.Set("_csrf", token)
		request := httptest.NewRequest(http.MethodPost, "/admin/site/broadcast", strings.NewReader(form.Encode()))
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
		if _, found, err := store.GetSetting(context.Background(), db, site.StaffNoticeSettingKey); err != nil || found {
			t.Fatalf("denied mutation persisted staff notice: found=%t err=%v", found, err)
		}
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	form := url.Values{"active": {"true"}, "severity": {"warning"}, "message": {"Yayın penceresi kısa süreliğine kapalı."}, "expires_at": {"2026-10-10T12:00"}}
	if response := post(admin, "wrong-token", form); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d, want 403", response.Code)
	}
	assertMissing()
	deniedAdmin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if response := post(deniedAdmin, "valid-token", form); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin status=%d, want 403", response.Code)
	}
	assertMissing()
	if response := post(admin, "valid-token", form); response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/site/broadcast?saved=1" {
		t.Fatalf("authorized staff notice response=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	setting, found, err := store.GetSetting(context.Background(), db, site.StaffNoticeSettingKey)
	if err != nil || !found || setting.Public {
		t.Fatalf("staff notice setting=%+v found=%t err=%v", setting, found, err)
	}
	want := site.StaffBroadcastNotice{Active: true, Severity: "warning", Message: "Yayın penceresi kısa süreliğine kapalı.", ExpiresAt: "2026-10-10T12:00:00Z"}
	if got := site.StaffBroadcastNoticeFromSetting(setting); got != want {
		t.Fatalf("persisted staff notice=%+v want=%+v", got, want)
	}
	publicSettings, err := store.ListPublicSettings(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	for _, publicSetting := range publicSettings {
		if publicSetting.Key == site.StaffNoticeSettingKey {
			t.Fatal("internal staff notice leaked into public settings projection")
		}
	}
	invalid := url.Values{"active": {"true"}, "severity": {"internal"}, "message": {"Invalid"}, "expires_at": {"2026-10-10T12:00"}}
	if response := post(admin, "valid-token", invalid); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid notice status=%d, want 400", response.Code)
	}
	unchanged, found, err := store.GetSetting(context.Background(), db, site.StaffNoticeSettingKey)
	if err != nil || !found || unchanged.Value != setting.Value {
		t.Fatalf("invalid update changed staff notice: found=%t err=%v", found, err)
	}
}

func TestSiteCustomCodeAdminRouteIsPrivateAndAdminOnly(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-custom-code-route.sqlite"))
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
		if route.Pattern == "POST /admin/site/custom-code" || route.Pattern == "GET /admin/site/custom-code" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	post := func(user *auth.User, token, head string) *httptest.ResponseRecorder {
		t.Helper()
		form := url.Values{"_csrf": {token}, "custom_head_code": {head}}
		request := httptest.NewRequest(http.MethodPost, "/admin/site/custom-code", strings.NewReader(form.Encode()))
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
		if _, found, err := store.GetSetting(context.Background(), db, site.CustomCodeSettingKey); err != nil || found {
			t.Fatalf("unauthorized custom code persisted: found=%t err=%v", found, err)
		}
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	valid := `<script src="https://www.googletagmanager.com/gtm.js"></script>`
	if response := post(admin, "wrong-token", valid); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d, want 403", response.Code)
	}
	assertMissing()
	for name, user := range map[string]*auth.User{
		"editor with site permission": {ID: "editor", Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}},
		"inactive admin":              {ID: "inactive", Role: auth.RoleAdmin, Active: false},
	} {
		t.Run(name, func(t *testing.T) {
			if response := post(user, "valid-token", valid); response.Code != http.StatusForbidden {
				t.Fatalf("custom code status=%d, want 403", response.Code)
			}
			assertMissing()
		})
	}
	if response := post(admin, "valid-token", valid); response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/site/custom-code?saved=1" {
		t.Fatalf("authorized custom code response=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	setting, found, err := store.GetSetting(context.Background(), db, site.CustomCodeSettingKey)
	if err != nil || !found || setting.Public || site.CustomCodeFromSetting(setting).Head != valid {
		t.Fatalf("private custom code=%+v found=%t err=%v", setting, found, err)
	}
	publicSettings, err := store.ListPublicSettings(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	for _, publicSetting := range publicSettings {
		if publicSetting.Key == site.CustomCodeSettingKey {
			t.Fatal("custom code appeared in the public settings projection")
		}
	}
	getRequest := httptest.NewRequest(http.MethodGet, "/admin/site/custom-code", nil)
	getRequest = getRequest.WithContext(auth.WithUser(getRequest.Context(), admin))
	getResponse := httptest.NewRecorder()
	secured.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK || !strings.Contains(getResponse.Body.String(), "www.googletagmanager.com/gtm.js") {
		t.Fatalf("custom code editor did not read persisted value: status=%d body=%s", getResponse.Code, getResponse.Body.String())
	}
	if response := post(admin, "valid-token", `<script src="https://evil.example/steal.js"></script>`); response.Code != http.StatusBadRequest {
		t.Fatalf("unapproved provider status=%d, want 400", response.Code)
	}
	unchanged, found, err := store.GetSetting(context.Background(), db, site.CustomCodeSettingKey)
	if err != nil || !found || site.CustomCodeFromSetting(unchanged).Head != valid {
		t.Fatalf("invalid custom code replaced prior value: found=%t err=%v value=%+v", found, err, unchanged)
	}
}

func TestSiteTranslationAdminRoutePersistsTenantScopedLocaleOverride(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-translation-route.sqlite"))
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
		if route.Pattern == "POST /admin/translations" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	const tenantID = "translation-test-tenant"
	defer i18n.SetOverride(tenantID, "en", "admin.dashboard", "")
	post := func(user *auth.User, token, locale, key, value string) *httptest.ResponseRecorder {
		t.Helper()
		form := url.Values{"_csrf": {token}, "locale": {locale}, "key": {key}, "value": {value}}
		request := httptest.NewRequest(http.MethodPost, "/admin/translations", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		ctx := auth.WithUser(request.Context(), user)
		ctx = tenant.WithScope(ctx, tenant.Scope{TenantID: tenantID, Slug: "translations"})
		request = request.WithContext(ctx)
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	assertMissing := func() {
		t.Helper()
		if _, found, err := store.GetSetting(context.Background(), db, i18n.OverrideSettingKey("en", "admin.dashboard")); err != nil || found {
			t.Fatalf("unauthorized translation persisted: found=%t err=%v", found, err)
		}
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	for name, userToken := range map[string]struct {
		user  *auth.User
		token string
	}{
		"invalid CSRF": {user: admin, token: "wrong-token"},
		"editor":       {user: &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, token: "valid-token"},
		"admin deny":   {user: &auth.User{ID: "admin-denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}, token: "valid-token"},
	} {
		t.Run(name, func(t *testing.T) {
			if response := post(userToken.user, userToken.token, "en-US", "admin.dashboard", "Tenant dashboard"); response.Code != http.StatusForbidden {
				t.Fatalf("translation status=%d, want 403", response.Code)
			}
			assertMissing()
		})
	}
	response := post(admin, "valid-token", "en-US", "admin.dashboard", "Tenant dashboard")
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/translations?locale=en&saved=1" {
		t.Fatalf("authorized translation response=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	setting, found, err := store.GetSetting(context.Background(), db, i18n.OverrideSettingKey("en", "admin.dashboard"))
	if err != nil || !found || !setting.Public || setting.Value != "Tenant dashboard" {
		t.Fatalf("persisted locale override=%+v found=%t err=%v", setting, found, err)
	}
	rows := i18n.CatalogRows(tenantID, "en")
	var translated bool
	for _, row := range rows {
		if row.Key == "admin.dashboard" && row.Translation == "Tenant dashboard" && !row.Missing {
			translated = true
		}
	}
	if !translated {
		t.Fatal("saved translation was not visible in the tenant-scoped catalog")
	}
	if response := post(admin, "valid-token", "en", "unknown.key", "Should not persist"); response.Code != http.StatusBadRequest {
		t.Fatalf("unknown catalog key status=%d, want 400", response.Code)
	}
	unchanged, found, err := store.GetSetting(context.Background(), db, i18n.OverrideSettingKey("en", "admin.dashboard"))
	if err != nil || !found || unchanged.Value != "Tenant dashboard" {
		t.Fatalf("invalid translation changed stored override: found=%t err=%v value=%+v", found, err, unchanged)
	}
}

func TestSiteRedirectRuleAdminRoutesPersistAuthorizedCRUD(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-redirect-rule-route.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	service := &site.Service{Transactions: database.NewTransactor(db), Store: store}
	registry := &site.RedirectRegistry{}
	handler := Handler{Service: service, Redirects: registry}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		if route.Pattern == "POST /admin/site/redirect-rules" || route.Pattern == "POST /admin/site/redirect-rules/{id}/delete" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	create := func(user *auth.User, token, pattern, kind, target, status string) *httptest.ResponseRecorder {
		t.Helper()
		form := url.Values{"_csrf": {token}, "pattern": {pattern}, "kind": {kind}, "target": {target}, "status": {status}}
		request := httptest.NewRequest(http.MethodPost, "/admin/site/redirect-rules", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	remove := func(user *auth.User, token, id string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/admin/site/redirect-rules/"+url.PathEscape(id)+"/delete", strings.NewReader("_csrf="+token))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	list := func() []site.RedirectRule {
		t.Helper()
		rules, err := service.ListRedirectRules(context.Background(), 100)
		if err != nil {
			t.Fatal(err)
		}
		return rules
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	if response := create(admin, "wrong-token", "/legacy/*", "wildcard", "/new/*", "301"); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF create status=%d, want 403", response.Code)
	}
	if len(list()) != 0 {
		t.Fatal("invalid CSRF created redirect rule")
	}
	deniedAdmin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if response := create(deniedAdmin, "valid-token", "/legacy/*", "wildcard", "/new/*", "301"); response.Code != http.StatusForbidden {
		t.Fatalf("denied create status=%d, want 403", response.Code)
	}
	if len(list()) != 0 {
		t.Fatal("denied Admin created redirect rule")
	}
	if response := create(admin, "valid-token", "/legacy/*", "wildcard", "/new/*", "301"); response.Code != http.StatusSeeOther {
		t.Fatalf("authorized create status=%d body=%s", response.Code, response.Body.String())
	}
	rules := list()
	if len(rules) != 1 || rules[0].ID == "" || rules[0].Pattern != "/legacy/*" || rules[0].TargetPath != "/new/*" || rules[0].Status != http.StatusMovedPermanently {
		t.Fatalf("persisted redirect rules=%+v", rules)
	}
	if got, found := registry.Match(tenant.Scope{}, "/legacy/page"); !found || got.TargetPath != "/new/page" {
		t.Fatalf("saved redirect did not reach runtime registry: rule=%+v found=%t", got, found)
	}
	if response := create(admin, "valid-token", "/new/page", "exact", "/legacy/page", "301"); response.Code != http.StatusBadRequest {
		t.Fatalf("cyclic redirect status=%d body=%s", response.Code, response.Body.String())
	}
	if unchanged := list(); len(unchanged) != 1 || unchanged[0].Pattern != "/legacy/*" {
		t.Fatalf("cyclic redirect changed persistence: %+v", unchanged)
	}
	id := rules[0].ID
	if response := remove(&auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, "valid-token", id); response.Code != http.StatusForbidden {
		t.Fatalf("Editor delete status=%d, want 403", response.Code)
	}
	if len(list()) != 1 {
		t.Fatal("denied Editor deleted redirect rule")
	}
	if response := remove(deniedAdmin, "valid-token", id); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin delete status=%d, want 403", response.Code)
	}
	if len(list()) != 1 {
		t.Fatal("denied Admin deleted redirect rule")
	}
	if response := remove(admin, "valid-token", id); response.Code != http.StatusSeeOther {
		t.Fatalf("authorized delete status=%d body=%s", response.Code, response.Body.String())
	}
	if remaining := list(); len(remaining) != 0 {
		t.Fatalf("deleted redirect rule remains: %+v", remaining)
	}
	if _, found := registry.Match(tenant.Scope{}, "/legacy/page"); found {
		t.Fatal("deleted redirect remains in runtime registry")
	}
}

func TestSiteRedirectRuleCSVImportIsAuthorizedAndAllOrNothing(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-redirect-import-route.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	service := &site.Service{Transactions: database.NewTransactor(db), Store: store}
	registry := &site.RedirectRegistry{}
	handler := Handler{Service: service, Redirects: registry}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		if route.Pattern == "POST /admin/site/redirect-rules/import" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	importCSV := func(user *auth.User, token, csv string) *httptest.ResponseRecorder {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "redirects.csv")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(csv)); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, "/admin/site/redirect-rules/import", &body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	list := func() []site.RedirectRule {
		t.Helper()
		rules, err := service.ListRedirectRules(context.Background(), 100)
		if err != nil {
			t.Fatal(err)
		}
		return rules
	}
	assertEmpty := func() {
		t.Helper()
		if rules := list(); len(rules) != 0 {
			t.Fatalf("denied or invalid import persisted rules: %+v", rules)
		}
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	validCSV := "pattern,kind,target,status\n/old-a,exact,/new-a,301\n/old-b,exact,/new-b,308\n"
	if response := importCSV(admin, "wrong-token", validCSV); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d, want 403", response.Code)
	}
	assertEmpty()
	if response := importCSV(&auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, "valid-token", validCSV); response.Code != http.StatusForbidden {
		t.Fatalf("Editor import status=%d, want 403", response.Code)
	}
	assertEmpty()
	deniedAdmin := &auth.User{ID: "admin-denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if response := importCSV(deniedAdmin, "valid-token", validCSV); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin import status=%d, want 403", response.Code)
	}
	assertEmpty()
	invalidCSV := "pattern,kind,target,status\n/valid-first,exact,/target-first,301\n/invalid-second,exact,//evil.example/path,301\n"
	if response := importCSV(admin, "valid-token", invalidCSV); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid batch status=%d, want 400", response.Code)
	}
	assertEmpty()
	cyclicCSV := "pattern,kind,target,status\n/cycle-a,exact,/cycle-b,301\n/cycle-b,exact,/cycle-a,301\n"
	if response := importCSV(admin, "valid-token", cyclicCSV); response.Code != http.StatusBadRequest {
		t.Fatalf("cyclic CSV status=%d body=%s", response.Code, response.Body.String())
	}
	assertEmpty()
	if response := importCSV(admin, "valid-token", validCSV); response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/site/redirect-rules" {
		t.Fatalf("authorized import status=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	rules := list()
	if len(rules) != 2 || rules[0].Pattern != "/old-a" || rules[0].TargetPath != "/new-a" || rules[0].Status != http.StatusMovedPermanently || rules[1].Pattern != "/old-b" || rules[1].Status != http.StatusPermanentRedirect {
		t.Fatalf("imported redirect rules=%+v", rules)
	}
	if got, found := registry.Match(tenant.Scope{}, "/old-a"); !found || got.TargetPath != "/new-a" {
		t.Fatalf("imported redirect did not reach runtime registry: rule=%+v found=%t", got, found)
	}
	if response := importCSV(admin, "valid-token", invalidCSV); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid follow-up batch status=%d, want 400", response.Code)
	}
	if unchanged := list(); len(unchanged) != 2 || unchanged[0].Pattern != "/old-a" || unchanged[1].Pattern != "/old-b" {
		t.Fatalf("failed import partially changed persisted rules: %+v", unchanged)
	}
}

func TestSiteFormBuilderAdminRoutesPersistAuthorizedCRUD(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-form-builder-route.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	formStore := formssql.Store{Executor: db, Dialect: database.SQLite}
	service := &site.Service{Transactions: database.NewTransactor(db), Store: store, Forms: formStore}
	handler := Handler{FormsEnabled: true, Service: service}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		if route.Pattern == "POST /admin/site/forms" || route.Pattern == "POST /admin/site/forms/{id}/delete" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	save := func(user *auth.User, token, schema string) *httptest.ResponseRecorder {
		t.Helper()
		form := url.Values{"_csrf": {token}, "schema": {schema}}
		request := httptest.NewRequest(http.MethodPost, "/admin/site/forms", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	remove := func(user *auth.User, token, id string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/admin/site/forms/"+url.PathEscape(id)+"/delete", strings.NewReader("_csrf="+token))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	get := func() (forms.Form, bool) {
		t.Helper()
		form, err := formStore.Get(context.Background(), "contact")
		if errors.Is(err, sql.ErrNoRows) {
			return forms.Form{}, false
		}
		if err != nil {
			t.Fatal(err)
		}
		return form, true
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	created := `{"id":"contact","title":"Contact","fields":[{"id":"email","type":"email","label":"Email","required":true}]}`
	if response := save(admin, "wrong-token", created); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF create status=%d, want 403", response.Code)
	}
	if _, found := get(); found {
		t.Fatal("invalid CSRF created form")
	}
	deniedAdmin := &auth.User{ID: "admin-denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if response := save(deniedAdmin, "valid-token", created); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin create status=%d, want 403", response.Code)
	}
	if _, found := get(); found {
		t.Fatal("denied Admin created form")
	}
	if response := save(&auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, "valid-token", created); response.Code != http.StatusForbidden {
		t.Fatalf("Editor create status=%d, want 403", response.Code)
	}
	if response := save(admin, "valid-token", created); response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/site/forms?id=contact" {
		t.Fatalf("authorized create status=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	stored, found := get()
	if !found || stored.Title != "Contact" || len(stored.Fields) != 1 || stored.Fields[0].Type != forms.Email {
		t.Fatalf("created form=%+v found=%t", stored, found)
	}
	updated := `{"id":"contact","title":"Contact support","fields":[{"id":"email","type":"email","label":"Email address","required":true}]}`
	if response := save(admin, "valid-token", updated); response.Code != http.StatusSeeOther {
		t.Fatalf("authorized update status=%d body=%s", response.Code, response.Body.String())
	}
	stored, found = get()
	if !found || stored.Title != "Contact support" || stored.Fields[0].Label != "Email address" {
		t.Fatalf("updated form=%+v found=%t", stored, found)
	}
	invalid := `{"id":"contact","title":"Invalid","fields":[{"id":"bad id","type":"email","label":"Email"}]}`
	if response := save(admin, "valid-token", invalid); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid schema status=%d, want 400", response.Code)
	}
	stored, found = get()
	if !found || stored.Title != "Contact support" {
		t.Fatalf("invalid schema replaced valid form: %+v found=%t", stored, found)
	}
	if response := remove(&auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, "valid-token", "contact"); response.Code != http.StatusForbidden {
		t.Fatalf("Editor delete status=%d, want 403", response.Code)
	}
	if response := remove(deniedAdmin, "valid-token", "contact"); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin delete status=%d, want 403", response.Code)
	}
	if response := remove(admin, "wrong-token", "contact"); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF delete status=%d, want 403", response.Code)
	}
	if _, found := get(); !found {
		t.Fatal("denied delete removed form")
	}
	if response := remove(admin, "valid-token", "contact"); response.Code != http.StatusSeeOther {
		t.Fatalf("authorized delete status=%d body=%s", response.Code, response.Body.String())
	}
	if _, found := get(); found {
		t.Fatal("authorized delete left form in the registry")
	}
}

func TestSiteFormWorkingHoursRoutePersistsOnlyValidAuthorizedSchedule(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-working-hours-route.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	formStore := formssql.Store{Executor: db, Dialect: database.SQLite}
	service := &site.Service{Transactions: database.NewTransactor(db), Store: store, Forms: formStore}
	baseline := forms.WorkingHours{Timezone: "UTC", Notice: "Existing hours", Days: [7]forms.DailyHours{{}, {Open: "08:00", Close: "16:00"}}}
	initial := forms.Form{ID: "contact", Title: "Contact", Fields: []forms.Field{{ID: "email", Type: forms.Email, Label: "Email", Required: true}}, WorkingHours: &baseline}
	if err := service.SaveForm(context.Background(), initial); err != nil {
		t.Fatal(err)
	}
	handler := Handler{FormsEnabled: true, Service: service}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		if route.Pattern == "POST /admin/site/forms/{id}/working-hours" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	save := func(user *auth.User, token string, values url.Values) *httptest.ResponseRecorder {
		t.Helper()
		values.Set("_csrf", token)
		request := httptest.NewRequest(http.MethodPost, "/admin/site/forms/contact/working-hours", strings.NewReader(values.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	load := func() forms.Form {
		t.Helper()
		form, err := formStore.Get(context.Background(), "contact")
		if err != nil {
			t.Fatal(err)
		}
		return form
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	valid := url.Values{"timezone": {"Europe/Istanbul"}, "notice": {"We reopen at {time}."}, "open_1": {"09:00"}, "close_1": {"17:00"}}
	for _, denied := range []struct {
		name  string
		user  *auth.User
		token string
	}{
		{name: "invalid CSRF", user: admin, token: "wrong-token"},
		{name: "Editor", user: &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, token: "valid-token"},
		{name: "permission denied", user: &auth.User{ID: "denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}, token: "valid-token"},
	} {
		t.Run(denied.name, func(t *testing.T) {
			if response := save(denied.user, denied.token, valid); response.Code != http.StatusForbidden {
				t.Fatalf("denied update status=%d body=%s", response.Code, response.Body.String())
			}
			if got := load().WorkingHours; got == nil || *got != baseline {
				t.Fatalf("denied update changed schedule: got=%+v want=%+v", got, baseline)
			}
		})
	}
	invalid := url.Values{"timezone": {"Not/AZone"}, "open_1": {"17:00"}, "close_1": {"09:00"}}
	if response := save(admin, "valid-token", invalid); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid schedule status=%d body=%s", response.Code, response.Body.String())
	}
	if got := load().WorkingHours; got == nil || *got != baseline {
		t.Fatalf("invalid schedule replaced current state: got=%+v want=%+v", got, baseline)
	}
	if response := save(admin, "valid-token", valid); response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/site/forms?id=contact" {
		t.Fatalf("authorized schedule status=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	saved := load()
	if saved.WorkingHours == nil || saved.WorkingHours.Timezone != "Europe/Istanbul" || saved.WorkingHours.Days[1] != (forms.DailyHours{Open: "09:00", Close: "17:00"}) || saved.WorkingHours.Notice != "We reopen at {time}." {
		t.Fatalf("persisted schedule mismatch: %+v", saved.WorkingHours)
	}
}

func TestPublicFormSubmissionToAdminInboxAndLeadWorkflow(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-public-inbox-workflow.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	formStore := formssql.Store{Executor: db, Dialect: database.SQLite}
	service := &site.Service{Transactions: database.NewTransactor(db), Store: store, Forms: formStore}
	form := forms.Form{ID: "contact", Title: "Contact", Fields: []forms.Field{
		{ID: "email", Type: forms.Email, Label: "Email", Required: true},
		{ID: "name", Type: forms.Text, Label: "Name", Required: true},
		{ID: "message", Type: forms.Textarea, Label: "Message", Required: true},
	}}
	if err := formStore.Upsert(context.Background(), form); err != nil {
		t.Fatal(err)
	}
	handler := Handler{DB: db, Store: store, Service: service, FormsEnabled: true, AllowedForms: map[string]struct{}{"contact": {}}}
	publicMux := http.NewServeMux()
	adminMux := http.NewServeMux()
	for _, route := range handler.Routes() {
		switch route.Pattern {
		case "POST /forms/{form}/submit":
			publicMux.Handle(route.Pattern, route.Handler)
		case "GET /admin/site/inbox/{form}", "POST /admin/site/inbox/{id}/status", "POST /admin/site/inbox/{id}/lead":
			adminMux.Handle(route.Pattern, route.Handler)
		}
	}
	securedAdmin := security.CSRFMiddleware(false, nil)(adminMux)
	submit := func(message, key string) *httptest.ResponseRecorder {
		t.Helper()
		body := url.Values{"email": {"customer@example.test"}, "name": {"Aylin"}, "message": {message}}
		request := httptest.NewRequest(http.MethodPost, "/forms/contact/submit", strings.NewReader(body.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("Idempotency-Key", key)
		response := httptest.NewRecorder()
		publicMux.ServeHTTP(response, request)
		return response
	}
	const key = "contact-submission-0001"
	if response := submit("Teklif almak istiyorum", key); response.Code != http.StatusAccepted {
		t.Fatalf("public form submit status=%d body=%s", response.Code, response.Body.String())
	}
	if response := submit("Teklif almak istiyorum", key); response.Code != http.StatusAccepted {
		t.Fatalf("idempotent replay status=%d body=%s", response.Code, response.Body.String())
	}
	if response := submit("Farklı içerik", key); response.Code != http.StatusConflict {
		t.Fatalf("idempotency conflict status=%d, want 409 body=%s", response.Code, response.Body.String())
	}
	loadSubmissions := func() []site.Submission {
		t.Helper()
		items, err := store.ListSubmissions(context.Background(), db, "contact", 100)
		if err != nil {
			t.Fatal(err)
		}
		return items
	}
	items := loadSubmissions()
	if len(items) != 1 || items[0].Email != "customer@example.test" || items[0].Status != "new" || items[0].LeadStage != "new" {
		t.Fatalf("persisted public submission=%+v", items)
	}
	submissionID := items[0].ID
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	getInbox := func(user *auth.User) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/admin/site/inbox/contact", nil)
		request.Header.Set("Accept", "text/html")
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		securedAdmin.ServeHTTP(response, request)
		return response
	}
	response := getInbox(admin)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "customer@example.test") || !strings.Contains(response.Body.String(), "Gelen talepler") {
		t.Fatalf("Admin inbox status=%d body=%s", response.Code, response.Body.String())
	}
	for _, internalValue := range []string{items[0].SourceIPHash, items[0].RequestHash, items[0].IdempotencyKey} {
		if internalValue != "" && strings.Contains(response.Body.String(), internalValue) {
			t.Errorf("inbox exposed internal submission metadata")
		}
	}
	if denied := getInbox(&auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}); denied.Code != http.StatusForbidden {
		t.Fatalf("Editor inbox status=%d, want 403", denied.Code)
	}
	postAdmin := func(user *auth.User, token, endpoint, values string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(values+"&_csrf="+token))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		result := httptest.NewRecorder()
		securedAdmin.ServeHTTP(result, request)
		return result
	}
	statusPath := "/admin/site/inbox/" + url.PathEscape(submissionID) + "/status"
	leadPath := "/admin/site/inbox/" + url.PathEscape(submissionID) + "/lead"
	if denied := postAdmin(admin, "wrong-token", statusPath, "status=triaged"); denied.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status update=%d, want 403", denied.Code)
	}
	deniedAdmin := &auth.User{ID: "denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if denied := postAdmin(deniedAdmin, "valid-token", statusPath, "status=triaged"); denied.Code != http.StatusForbidden {
		t.Fatalf("permission-denied status update=%d, want 403", denied.Code)
	}
	if unchanged := loadSubmissions(); unchanged[0].Status != "new" {
		t.Fatalf("denied status mutation changed submission: %+v", unchanged[0])
	}
	if updated := postAdmin(admin, "valid-token", statusPath, "status=triaged"); updated.Code != http.StatusNoContent {
		t.Fatalf("authorized status update=%d body=%s", updated.Code, updated.Body.String())
	}
	if updated := postAdmin(admin, "valid-token", leadPath, "lead_status=contacted&assigned_to=operator-1"); updated.Code != http.StatusNoContent {
		t.Fatalf("authorized lead update=%d body=%s", updated.Code, updated.Body.String())
	}
	final := loadSubmissions()
	if len(final) != 1 || final[0].Status != "triaged" || final[0].LeadStage != "contacted" || final[0].AssignedTo != "operator-1" {
		t.Fatalf("persisted inbox/lead workflow=%+v", final)
	}
}

func TestLeadNotesRequireSitePermissionAndPersistAuthor(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-lead-notes-route.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	service := &site.Service{Transactions: database.NewTransactor(db), Store: store}
	if err := service.Submit(context.Background(), site.Submission{ID: "form-note-001", FormKey: "contact", Email: "customer@example.test", Payload: `{"message":"Help"}`, SourceIPHash: strings.Repeat("a", 64)}); err != nil {
		t.Fatal(err)
	}
	handler := Handler{Service: service, FormsEnabled: true}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		if route.Pattern == "POST /admin/site/inbox/{id}/notes" || route.Pattern == "GET /admin/site/inbox/{id}/notes" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	list := func(user *auth.User) ([]site.LeadNote, int) {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/admin/site/inbox/form-note-001/notes", nil)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			return nil, response.Code
		}
		var notes []site.LeadNote
		if err := json.Unmarshal(response.Body.Bytes(), &notes); err != nil {
			t.Fatalf("decode lead notes: %v; body=%s", err, response.Body.String())
		}
		return notes, response.Code
	}
	add := func(user *auth.User, token, note string) *httptest.ResponseRecorder {
		t.Helper()
		form := url.Values{"_csrf": {token}, "note": {note}}
		request := httptest.NewRequest(http.MethodPost, "/admin/site/inbox/form-note-001/notes", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	if response := add(admin, "wrong-token", "Call customer tomorrow"); response.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF note status=%d, want 403", response.Code)
	}
	if response := add(&auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, "valid-token", "Call customer tomorrow"); response.Code != http.StatusForbidden {
		t.Fatalf("Editor note status=%d, want 403", response.Code)
	}
	deniedAdmin := &auth.User{ID: "admin-denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
	if response := add(deniedAdmin, "valid-token", "Call customer tomorrow"); response.Code != http.StatusForbidden {
		t.Fatalf("denied Admin note status=%d, want 403", response.Code)
	}
	if notes, _ := list(admin); len(notes) != 0 {
		t.Fatalf("denied note requests persisted notes: %+v", notes)
	}
	permittedEditor := &auth.User{ID: "editor-1", Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}}
	if response := add(permittedEditor, "valid-token", "Call customer tomorrow"); response.Code != http.StatusCreated {
		t.Fatalf("permitted Editor note status=%d body=%s", response.Code, response.Body.String())
	}
	if response := add(admin, "valid-token", ""); response.Code != http.StatusBadRequest {
		t.Fatalf("empty note status=%d, want 400", response.Code)
	}
	notes, status := list(admin)
	if status != http.StatusOK || len(notes) != 1 || notes[0].AuthorID != "editor-1" || notes[0].SubmissionID != "form-note-001" || notes[0].Note != "Call customer tomorrow" {
		t.Fatalf("persisted lead notes=%+v status=%d", notes, status)
	}
	if _, status := list(&auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}); status != http.StatusForbidden {
		t.Fatalf("viewer notes status=%d, want 403", status)
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
