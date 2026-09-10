package contract_test

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// openIsolatedPostgres gives each contract test its own schema. The CI
// database is intentionally reused by the package, so a test that exercises a
// synthetic migration set must not see the application's real migration
// ledger or another test's tables.
func openIsolatedPostgres(t *testing.T, ctx context.Context) *sql.DB {
	t.Helper()
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	admin, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "hydro_contract_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		_, _ = admin.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		_ = admin.Close()
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema)
	parsed.RawQuery = query.Encode()

	db, err := database.Open(ctx, database.Postgres, parsed.String())
	if err != nil {
		_, _ = admin.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		_ = admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = admin.ExecContext(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE")
		_ = admin.Close()
	})
	return db
}
