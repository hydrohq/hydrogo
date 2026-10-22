// Package testpostgres gives opt-in PostgreSQL contracts their own schema, so a
// contract can run again, and next to other contracts, on one shared database.
// The configured account needs CREATE SCHEMA on the database named by
// HYDRO_TEST_POSTGRES_URL.
package testpostgres

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

// Open returns a connection whose search_path is a schema created for this test
// and dropped when it ends. The test is skipped when no server is configured.
func Open(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	admin, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "hydro_test_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}
	drop := func() {
		_, _ = admin.ExecContext(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		_ = admin.Close()
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		drop()
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema)
	parsed.RawQuery = query.Encode()
	db, err := database.Open(ctx, database.Postgres, parsed.String())
	if err != nil {
		drop()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		drop()
	})
	return db
}
