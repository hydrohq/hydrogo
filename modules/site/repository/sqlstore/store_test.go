package sqlstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
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
	settings, err := store.ListPublicSettings(ctx, db)
	if err != nil || len(settings) != 1 || settings[0].Key != "public" || !settings[0].Public {
		t.Fatalf("public settings = %#v/%v", settings, err)
	}
	if err := store.ReplaceNavigation(ctx, db, "tr", []site.NavigationItem{
		{ID: "visible", Label: "Visible", Href: "/visible", Position: 1, Visible: true, UpdatedAt: now},
		{ID: "hidden", Label: "Hidden", Href: "/hidden", Position: 2, Visible: false, UpdatedAt: now},
	}); err != nil {
		t.Fatal(err)
	}
	navigation, err := store.ListNavigation(ctx, db, "tr")
	if err != nil || len(navigation) != 1 || navigation[0].ID != "visible" || !navigation[0].Visible {
		t.Fatalf("visible navigation = %#v/%v", navigation, err)
	}
}
