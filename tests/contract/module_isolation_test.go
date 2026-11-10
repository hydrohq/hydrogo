package contract

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

// TestPostgresModuleFilteringLeavesInactiveModuleSchemaAbsent mirrors the
// SQLite module-isolation contract on a real PostgreSQL connection. A
// dedicated schema keeps the disposable contract database isolated from other
// tests and is removed when the test finishes.
func TestPostgresModuleFilteringLeavesInactiveModuleSchemaAbsent(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	admin, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer admin.Close()

	schema := fmt.Sprintf("hydro_module_contract_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA "`+schema+`"`); err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	defer func() {
		if _, err := admin.ExecContext(context.Background(), `DROP SCHEMA "`+schema+`" CASCADE`); err != nil {
			t.Errorf("drop isolated schema: %v", err)
		}
	}()

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema)
	parsed.RawQuery = query.Encode()

	db, err := database.OpenWithOptions(ctx, database.Postgres, parsed.String(), database.OpenOptions{MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatalf("open isolated db: %v", err)
	}
	defer db.Close()

	enabled := map[string]bool{"pages": true, "media": true}
	if err := database.ApplyMigrationsForModules(ctx, db, database.Postgres, migrations.FS, enabled); err != nil {
		t.Fatalf("filtered postgres migrations failed: %v", err)
	}
	for _, table := range []struct {
		name string
		want bool
	}{
		{name: "schema_migrations", want: true},
		{name: "media_assets", want: true},
		{name: "media_variants", want: true},
		{name: "commerce_stock", want: false},
		{name: "news_items", want: false},
		{name: "news_taxonomy_terms", want: false},
		{name: "news_taxonomy_relations", want: false},
	} {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1`, table.name).Scan(&count); err != nil {
			t.Fatalf("inspect table %q: %v", table.name, err)
		}
		if (count == 1) != table.want {
			t.Fatalf("table %q present = %v, want %v", table.name, count == 1, table.want)
		}
	}
	if err := database.CheckCompatibilityForModules(ctx, db, database.Postgres, migrations.FS, enabled); err != nil {
		t.Fatalf("filtered postgres compatibility check failed: %v", err)
	}
}
