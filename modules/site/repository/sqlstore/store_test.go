package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/testmysql"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func TestSQLiteVisibilityDialectContract(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runVisibilityContract(t, db, database.SQLite)
}

func TestMySQLSiteRepositoryContracts(t *testing.T) {
	db := testmysql.Open(t)
	testmysql.MigrateAll(t, db)
	runVisibilityContract(t, db, database.MySQL)
	runRedirectRuleContract(t, db, database.MySQL)
	runNotFoundPathContract(t, db, database.MySQL)
	runIdempotencyContract(t, db, database.MySQL)
}

func TestScopedSiteSettingsAndNavigationCannotCrossTenant(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "scoped-site.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	alpha, _ := tenant.NewScope("tenant-a", "alpha")
	beta, _ := tenant.NewScope("tenant-b", "beta")
	store := Store{Dialect: database.SQLite, RequireTenantScope: true}
	if err := store.UpsertSetting(tenant.WithScope(ctx, alpha), db, site.Setting{Key: "title", Value: "Alpha", Public: true}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertSetting(tenant.WithScope(ctx, beta), db, site.Setting{Key: "title", Value: "Beta", Public: true}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertSetting(tenant.WithScope(ctx, alpha), db, site.Setting{Key: "site.maintenance", Value: "{\"enabled\":true}", Public: false}); err != nil {
		t.Fatal(err)
	}
	private, found, err := store.GetSetting(tenant.WithScope(ctx, alpha), db, "site.maintenance")
	if err != nil || !found || private.Public || private.Value != "{\"enabled\":true}" {
		t.Fatalf("private scoped setting = %#v found=%t err=%v", private, found, err)
	}
	if _, found, err := store.GetSetting(tenant.WithScope(ctx, beta), db, "site.maintenance"); err != nil || found {
		t.Fatalf("cross-tenant private setting leaked: found=%t err=%v", found, err)
	}
	items, err := store.ListPublicSettings(tenant.WithScope(ctx, alpha), db)
	if err != nil || len(items) != 1 || items[0].Value != "Alpha" {
		t.Fatalf("scoped settings = %#v/%v", items, err)
	}
	if _, err := store.ListPublicSettings(ctx, db); !errors.Is(err, tenant.ErrMissingScope) {
		t.Fatalf("missing site scope error = %v", err)
	}
}

func TestPostgresVisibilityDialectContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	db, err := database.Open(context.Background(), database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.Postgres, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), "DELETE FROM site_settings"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), "DELETE FROM navigation_items"); err != nil {
		t.Fatal(err)
	}
	runVisibilityContract(t, db, database.Postgres)
}

func TestSQLiteRedirectRuleContract(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "redirects.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runRedirectRuleContract(t, db, database.SQLite)
	runNotFoundPathContract(t, db, database.SQLite)
}

func runNotFoundPathContract(t *testing.T, db database.QueryExecutor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	alpha, _ := tenant.NewScope("miss-alpha", "alpha")
	beta, _ := tenant.NewScope("miss-beta", "beta")
	store := Store{Dialect: dialect, RequireTenantScope: true}
	sqldb, ok := db.(*sql.DB)
	if !ok {
		t.Fatal("database executor is not a SQL database")
	}
	transactions := database.NewTenantTransactor(sqldb, dialect, true)
	at := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	record := func(scope tenant.Scope) error {
		scoped := tenant.WithScope(ctx, scope)
		return transactions.Within(scoped, func(exec database.Executor) error { return store.RecordNotFound(scoped, exec, "/missing-page", at) })
	}
	for _, scope := range []tenant.Scope{alpha, beta} {
		scoped := tenant.WithScope(ctx, scope)
		if err := transactions.Within(scoped, func(exec database.Executor) error { return store.DeleteNotFound(scoped, exec, "/missing-page") }); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err := record(alpha); err != nil {
			t.Fatal(err)
		}
	}
	if err := record(beta); err != nil {
		t.Fatal(err)
	}
	var items []site.NotFoundPath
	alphaCtx := tenant.WithScope(ctx, alpha)
	err := transactions.WithinQuery(alphaCtx, func(query database.QueryExecutor) error {
		var err error
		items, err = store.ListNotFound(alphaCtx, query, 10)
		return err
	})
	if err != nil || len(items) != 1 || items[0].Hits != 2 || items[0].Path != "/missing-page" {
		t.Fatalf("alpha misses=%+v err=%v", items, err)
	}
	betaCtx := tenant.WithScope(ctx, beta)
	err = transactions.WithinQuery(betaCtx, func(query database.QueryExecutor) error {
		var err error
		items, err = store.ListNotFound(betaCtx, query, 10)
		return err
	})
	if err != nil || len(items) != 1 || items[0].Hits != 1 {
		t.Fatalf("beta misses=%+v err=%v", items, err)
	}
	if err := transactions.Within(alphaCtx, func(exec database.Executor) error { return store.DeleteNotFound(alphaCtx, exec, "/missing-page") }); err != nil {
		t.Fatal(err)
	}
	err = transactions.WithinQuery(alphaCtx, func(query database.QueryExecutor) error {
		var err error
		items, err = store.ListNotFound(alphaCtx, query, 10)
		return err
	})
	if err != nil || len(items) != 0 {
		t.Fatalf("deleted misses=%+v err=%v", items, err)
	}
}

func TestPostgresRedirectRuleContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	db, err := database.Open(context.Background(), database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.Postgres, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runRedirectRuleContract(t, db, database.Postgres)
	runNotFoundPathContract(t, db, database.Postgres)
}

func runRedirectRuleContract(t *testing.T, db database.QueryExecutor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	alpha, _ := tenant.NewScope("redirect-alpha", "alpha")
	beta, _ := tenant.NewScope("redirect-beta", "beta")
	store := Store{Dialect: dialect, RequireTenantScope: true}
	rule := site.RedirectRule{ID: "rule-a", TenantID: alpha.TenantID, Pattern: "/old", Kind: "exact", TargetPath: "/new", Status: 301, CreatedAt: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)}
	if err := store.UpsertRedirectRule(tenant.WithScope(ctx, alpha), db, rule); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertRedirectRule(tenant.WithScope(ctx, beta), db, site.RedirectRule{ID: "rule-b", Pattern: "/old", Kind: "exact", TargetPath: "/beta", Status: 302, CreatedAt: rule.CreatedAt}); err != nil {
		t.Fatal(err)
	}
	alphaRules, err := store.ListRedirectRules(tenant.WithScope(ctx, alpha), db, 10)
	if err != nil || len(alphaRules) != 1 || alphaRules[0].TargetPath != "/new" {
		t.Fatalf("alpha rules = %#v/%v", alphaRules, err)
	}
	betaRules, err := store.ListRedirectRules(tenant.WithScope(ctx, beta), db, 10)
	if err != nil || len(betaRules) != 1 || betaRules[0].TargetPath != "/beta" {
		t.Fatalf("beta rules = %#v/%v", betaRules, err)
	}
	if err := store.DeleteRedirectRule(tenant.WithScope(ctx, alpha), db, "rule-a"); err != nil {
		t.Fatal(err)
	}
	alphaRules, err = store.ListRedirectRules(tenant.WithScope(ctx, alpha), db, 10)
	if err != nil || len(alphaRules) != 0 {
		t.Fatalf("deleted alpha rules = %#v/%v", alphaRules, err)
	}
}

func runVisibilityContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	store := Store{Dialect: dialect}
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	if err := store.UpsertSetting(ctx, db, site.Setting{Key: "public", Value: "yes", Public: true, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertSetting(ctx, db, site.Setting{Key: "private", Value: "no", Public: false, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertSetting(ctx, db, site.Setting{Key: "i18n.override.en.admin.language", Value: "Interface language", Public: true, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	settings, err := store.ListPublicSettings(ctx, db)
	if err != nil || len(settings) != 2 || !settings[0].Public || !settings[1].Public {
		t.Fatalf("public settings = %#v/%v", settings, err)
	}
	foundOverride := false
	for _, setting := range settings {
		if setting.Key == "i18n.override.en.admin.language" && setting.Value == "Interface language" {
			foundOverride = true
		}
	}
	if !foundOverride {
		t.Fatal("persisted translation override missing from public settings projection")
	}
	if err := store.ReplaceNavigation(ctx, db, "tr", []site.NavigationItem{
		{ID: "visible", Label: "Visible", Href: "/visible", Position: 1, Visible: true, UpdatedAt: now},
		{ID: "child", ParentID: "visible", Target: "_blank", Label: "Child", Href: "/child", Position: 2, Visible: true, UpdatedAt: now},
		{ID: "hidden", Label: "Hidden", Href: "/hidden", Position: 3, Visible: false, UpdatedAt: now},
		{ID: "hidden-parent", Label: "Hidden parent", Href: "/hidden-parent", Position: 4, Visible: false, UpdatedAt: now},
		{ID: "orphaned-child", ParentID: "hidden-parent", Label: "Orphaned child", Href: "/orphaned-child", Position: 5, Visible: true, UpdatedAt: now},
	}); err != nil {
		t.Fatal(err)
	}
	navigation, err := store.ListNavigation(ctx, db, "tr")
	if err != nil || len(navigation) != 2 || navigation[0].ID != "visible" || !navigation[0].Visible || navigation[1].ParentID != "visible" || navigation[1].Target != "_blank" {
		t.Fatalf("visible navigation = %#v/%v", navigation, err)
	}
	adminNavigation, err := store.ListAdminNavigation(ctx, db, "tr")
	if err != nil || len(adminNavigation) != 5 {
		t.Fatalf("admin navigation = %#v/%v", adminNavigation, err)
	}
}

func TestIdempotencyDialectContractSQLite(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "idem.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runIdempotencyContract(t, db, database.SQLite)
}

func TestIdempotencyDialectContractPostgres(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	db, err := database.Open(context.Background(), database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.Postgres, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), "DELETE FROM form_inbox"); err != nil {
		t.Fatal(err)
	}
	runIdempotencyContract(t, db, database.Postgres)
}

func runIdempotencyContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	store := Store{Dialect: dialect}
	value := site.Submission{ID: "form-idem-1", FormKey: "contact", Email: "user@example.com", Payload: `{"message":"hello"}`, SourceIPHash: "hash", IdempotencyKey: "0123456789abcdef", RequestHash: "hash-1"}
	if err := store.CreateSubmission(ctx, db, value); err != nil {
		t.Fatal(err)
	}
	got, found, err := store.FindSubmissionByIdempotency(ctx, db, "contact", value.IdempotencyKey)
	if err != nil || !found || got.RequestHash != value.RequestHash {
		t.Fatalf("lookup = %#v found=%v err=%v", got, found, err)
	}
	missing, found, err := store.FindSubmissionByIdempotency(ctx, db, "contact", "fedcba9876543210")
	if err != nil || found || missing.ID != "" {
		t.Fatalf("missing lookup = %#v found=%v err=%v", missing, found, err)
	}
}
