package contract_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/analytics/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteContentViewCountsContract(t *testing.T) {
	runContentViewCountsContract(t, database.SQLite)
}

func TestPostgresContentViewCountsContract(t *testing.T) {
	if os.Getenv("HYDRO_TEST_POSTGRES_URL") == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runContentViewCountsContract(t, database.Postgres)
}

func runContentViewCountsContract(t *testing.T, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	var db *sql.DB
	if dialect == database.Postgres {
		db = openIsolatedPostgres(t, ctx)
	} else {
		var err error
		db, err = database.Open(ctx, dialect, filepath.Join(t.TempDir(), "analytics-counts.sqlite"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
	}
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM analytics_content_daily"); err != nil {
		t.Fatal(err)
	}
	insert := "INSERT INTO analytics_content_daily(day,tenant_id,content_type,content_id,locale,purpose_version,view_count) VALUES (" + placeholderForTest(dialect, 1) + "," + placeholderForTest(dialect, 2) + "," + placeholderForTest(dialect, 3) + "," + placeholderForTest(dialect, 4) + "," + placeholderForTest(dialect, 5) + "," + placeholderForTest(dialect, 6) + "," + placeholderForTest(dialect, 7) + ")"
	for _, row := range []struct {
		day, tenant, typ, id, locale string
		count                        int
	}{
		{"2026-09-01", "tenant-a", "page", "p1", "en", 3},
		{"2026-09-02", "tenant-a", "page", "p1", "tr", 4},
		{"2026-09-01", "tenant-a", "page", "p2", "en", 8},
		{"2026-09-01", "tenant-b", "page", "p1", "en", 100},
		{"2026-09-01", "tenant-a", "news", "p1", "en", 200},
	} {
		if _, err := db.ExecContext(ctx, insert, row.day, row.tenant, row.typ, row.id, row.locale, "v1", row.count); err != nil {
			t.Fatal(err)
		}
	}
	store := sqlstore.Store{Executor: db, Dialect: dialect}
	counts, err := store.ContentViewCounts(ctx, "tenant-a", "page", []string{"p1", "p2", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if counts["p1"] != 7 || counts["p2"] != 8 || counts["missing"] != 0 || len(counts) != 2 {
		t.Fatalf("content view totals = %#v", counts)
	}
}
