package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	"github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
)

func TestTranslationEditorPersistsTenantOverrideAndUpdatesRenderedCatalog(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "translations.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{"site": true}); err != nil {
		t.Fatal(err)
	}
	scope, err := tenant.NewScope("translation-tenant", "translation")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { i18n.ReplaceOverrides(scope.TenantID, nil) })
	requestCtx := tenant.WithScope(ctx, scope)
	store := sqlstore.Store{Dialect: database.SQLite, RequireTenantScope: true}
	service := site.Service{Transactions: database.NewTenantTransactor(db, database.SQLite, true), Store: store}
	handler := Handler{DB: db, Store: store, Service: &service}
	form := url.Values{"locale": {"en"}, "key": {"admin.language"}, "value": {"Interface language"}}
	request := httptest.NewRequest(http.MethodPost, "/admin/translations", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(auth.WithUser(requestCtx, &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response := httptest.NewRecorder()
	handler.saveTranslation(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("save status=%d body=%s", response.Code, response.Body.String())
	}
	if got := i18n.TForTenant(scope.TenantID, "en", "admin.language"); got != "Interface language" {
		t.Fatalf("rendered translation = %q", got)
	}
	var rendered strings.Builder
	if err := ui.RenderAdminArea(&rendered, ui.AdminAreaData{Email: "admin@example.test", Area: "account", Locale: "en", TenantID: scope.TenantID}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered.String(), "Interface language") {
		t.Fatal("updated catalog value did not appear in a rendered Admin view")
	}
	settings, err := store.ListPublicSettings(requestCtx, db)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, setting := range settings {
		if setting.Key == i18n.OverrideSettingKey("en", "admin.language") && setting.Value == "Interface language" {
			found = true
		}
	}
	if !found {
		t.Fatal("translation override was not persisted")
	}
	other, _ := tenant.NewScope("another-translation-tenant", "other")
	if got := i18n.TForTenant(other.TenantID, "en", "admin.language"); got != "Language" {
		t.Fatalf("translation leaked across tenants: %q", got)
	}
}

func TestTranslationEditorListsSearchableLocaleMatrix(t *testing.T) {
	scope, _ := tenant.NewScope("matrix-tenant", "matrix")
	ctx := tenant.WithScope(context.Background(), scope)
	request := httptest.NewRequest(http.MethodGet, "/admin/translations?locale=en", nil)
	request = request.WithContext(auth.WithUser(ctx, &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response := httptest.NewRecorder()
	(Handler{Store: fakeStore{}, Service: &site.Service{}}).translations(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	for _, marker := range []string{"translation-matrix", "translation-search", "admin.pages", "Türkçe (TR)", "English (EN)", "aria-controls", "name=\"_csrf\""} {
		if !strings.Contains(response.Body.String(), marker) {
			t.Errorf("translation matrix missing %q", marker)
		}
	}
}

func TestNavigationEditorRoundTripsNestedLocalizedMenu(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "navigation-editor.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{"site": true}); err != nil {
		t.Fatal(err)
	}
	scope, _ := tenant.NewScope("navigation-tenant", "navigation")
	requestCtx := tenant.WithScope(ctx, scope)
	store := sqlstore.Store{Dialect: database.SQLite, RequireTenantScope: true}
	service := site.Service{Transactions: database.NewTenantTransactor(db, database.SQLite, true), Store: store}
	if err := service.SaveNavigation(requestCtx, "en", []site.NavigationItem{
		{ID: "root", Label: "Services", Href: "/services", Position: 0, Visible: true},
		{ID: "child", ParentID: "root", Target: "_blank", Label: "Consulting", Href: "/consulting", Position: 1, Visible: true},
	}); err != nil {
		t.Fatal(err)
	}
	handler := Handler{DB: db, Store: store, Service: &service, NavigationLocales: []string{"en", "fr"}, NavigationSuggestions: func(_ context.Context, locale string) ([]site.NavigationSuggestion, error) {
		if locale != "en" {
			t.Fatalf("suggestion locale = %q", locale)
		}
		return []site.NavigationSuggestion{{Label: "Contact", Href: "/public/pages/en/contact/html"}}, nil
	}}
	adminCtx := auth.WithUser(requestCtx, &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true})
	get := httptest.NewRequest(http.MethodGet, "/admin/site/navigation/editor?locale=en", nil).WithContext(adminCtx)
	page := httptest.NewRecorder()
	handler.navigationEditor(page, get)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), `name="parent_id"`) || !strings.Contains(page.Body.String(), `data-move-up`) || !strings.Contains(page.Body.String(), `draggable="true"`) || !strings.Contains(page.Body.String(), `value="/public/pages/en/contact/html"`) || !strings.Contains(page.Body.String(), `locale=fr`) {
		t.Fatalf("editor response status=%d body=%s", page.Code, page.Body.String())
	}
	form := url.Values{
		"locale":    {"en"},
		"id":        {"root", "child"},
		"label":     {"Services", "Consulting"},
		"href":      {"/services", "/consulting"},
		"parent_id": {"", "root"},
		"target":    {"_self", "_blank"},
		"visible":   {"true", "true"},
	}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		if route.Pattern == "POST /admin/site/navigation/editor" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	submit := func(user *auth.User, token string) *httptest.ResponseRecorder {
		t.Helper()
		values := url.Values{}
		for key, entries := range form {
			values[key] = append([]string(nil), entries...)
		}
		values.Set("_csrf", token)
		post := httptest.NewRequest(http.MethodPost, "/admin/site/navigation/editor", strings.NewReader(values.Encode())).WithContext(auth.WithUser(requestCtx, user))
		post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		post.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		post.Header.Set(security.CSRFHeaderName, token)
		saved := httptest.NewRecorder()
		secured.ServeHTTP(saved, post)
		return saved
	}
	for _, denied := range []struct {
		name  string
		user  *auth.User
		token string
	}{
		{name: "invalid CSRF", user: &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}, token: "wrong-token"},
		{name: "Editor", user: &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, token: "valid-token"},
		{name: "permission denied", user: &auth.User{ID: "denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}, token: "valid-token"},
	} {
		if response := submit(denied.user, denied.token); response.Code != http.StatusForbidden {
			t.Fatalf("%s save status=%d body=%s", denied.name, response.Code, response.Body.String())
		}
		unchanged, err := store.ListNavigation(requestCtx, db, "en")
		if err != nil || len(unchanged) != 2 || unchanged[1].ParentID != "root" || unchanged[1].Target != "_blank" {
			t.Fatalf("%s changed saved navigation = %#v, err=%v", denied.name, unchanged, err)
		}
	}
	if saved := submit(&auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}, "valid-token"); saved.Code != http.StatusSeeOther {
		t.Fatalf("save status=%d body=%s", saved.Code, saved.Body.String())
	}
	items, err := store.ListNavigation(requestCtx, db, "en")
	if err != nil || len(items) != 2 || items[1].ParentID != "root" || items[1].Target != "_blank" {
		t.Fatalf("saved navigation = %#v, err=%v", items, err)
	}
}
