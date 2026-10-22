package sqlstore

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

// TestPostgresAuditChainSerializesConcurrentWriters checks on a real server
// that writers racing for one tenant's head queue on its row lock, leaving a
// chain without forks or gaps.
func TestPostgresAuditChainSerializesConcurrentWriters(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	adminDB, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schemaName := "audit_chain_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := adminDB.ExecContext(ctx, "CREATE SCHEMA "+schemaName); err != nil {
		adminDB.Close()
		t.Fatal(err)
	}
	defer func() {
		_, _ = adminDB.ExecContext(context.Background(), "DROP SCHEMA "+schemaName+" CASCADE")
		_ = adminDB.Close()
	}()
	parsed, _ := url.Parse(dsn)
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schemaName)
	parsed.RawQuery = query.Encode()
	db, err := database.Open(ctx, database.Postgres, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.Postgres, migrations.FS, map[string]bool{"core": true}); err != nil {
		t.Fatal(err)
	}
	SetChain(true)
	defer SetChain(false)
	store := Store{Dialect: database.Postgres}
	const writers, each = 8, 10
	var wg sync.WaitGroup
	errs := make(chan error, writers*each)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for n := 0; n < each; n++ {
				errs <- database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
					return store.Record(ctx, tx, fmt.Sprintf("w%d-%d", w, n), "content.updated", "r")
				})
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	report, err := store.VerifyChain(ctx, db, "", nil)
	if err != nil || len(report.Problems) != 0 || report.Links != writers*each {
		t.Fatalf("chain: %+v err=%v", report, err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM audit_chain WHERE seq = 3"); err == nil {
		t.Fatal("chain rows accepted a delete")
	}
}
