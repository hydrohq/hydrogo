package transport

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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

type retryMailSpool struct{ retried []string }

func (s *retryMailSpool) List(context.Context, string, int) ([]mailer.SpoolEntry, error) {
	return nil, nil
}

func (s *retryMailSpool) RetryNow(_ context.Context, id string) error {
	s.retried = append(s.retried, id)
	return nil
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
