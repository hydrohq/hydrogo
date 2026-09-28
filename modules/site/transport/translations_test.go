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
