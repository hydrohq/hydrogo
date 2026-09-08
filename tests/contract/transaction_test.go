package contract_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"testing/fstest"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/pages/repository/sqlstore"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

func TestSQLiteTransactionContract(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "contract.sqlite")
	runTransactionContract(t, database.SQLite, dsn)
}

func TestPostgresTransactionContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runTransactionContract(t, database.Postgres, dsn)
}

func runTransactionContract(t *testing.T, dialect database.Dialect, dsn string) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, dialect, dsn)
	if err != nil {
		t.Fatalf("open %s: %v", dialect, err)
	}
	t.Cleanup(func() { db.Close() })

	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatalf("migrate %s: %v", dialect, err)
	}
	if err := database.CheckCompatibility(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatalf("schema compatibility %s: %v", dialect, err)
	}
	resetContractData(t, db, dialect)
	if dialect == database.SQLite {
		assertSQLitePragmas(t, db)
	}

	pageStore := sqlstore.Store{Dialect: dialect}
	auditStore := auditstore.Store{Dialect: dialect}
	outboxStore := outboxstore.Store{Dialect: dialect}
	creator := service.CreateDraft{
		Transactions: database.NewTransactor(db),
		Pages:        pageStore,
		Audit:        auditStore,
		Outbox:       failingOutbox{err: errors.New("injected outbox failure")},
	}
	if err := creator.Execute(ctx, "rollback-page", "must roll back"); err == nil {
		t.Fatal("injected transaction failure succeeded")
	}
	assertCount(t, db, "pages", 0)
	assertCount(t, db, "audit_events", 0)
	assertCount(t, db, "outbox_events", 0)

	creator.Outbox = outboxStore
	if err := creator.Execute(ctx, "committed-page", "committed"); err != nil {
		t.Fatalf("commit transaction: %v", err)
	}
	assertCount(t, db, "pages", 1)
	assertCount(t, db, "audit_events", 1)
	assertCount(t, db, "outbox_events", 1)
	assertOptimisticLock(t, ctx, database.NewTransactor(db), pageStore)

	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatalf("idempotent migration: %v", err)
	}
	tampered := fstest.MapFS{
		"202609060001_transaction_contract/" + string(dialect) + ".sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	}
	if err := database.ApplyMigrations(ctx, db, dialect, tampered); err == nil {
		t.Fatal("tampered published migration checksum was accepted")
	}
}

func assertOptimisticLock(t *testing.T, ctx context.Context, transactions database.Transactor, pages sqlstore.Store) {
	t.Helper()
	start := make(chan struct{})
	results := make(chan bool, 2)
	var group sync.WaitGroup
	for _, title := range []string{"editor one", "editor two"} {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			updated := false
			err := transactions.Within(ctx, func(tx database.Executor) error {
				var err error
				updated, err = pages.UpdateTitle(ctx, tx, "committed-page", 1, title)
				return err
			})
			results <- err == nil && updated
		}()
	}
	close(start)
	group.Wait()
	close(results)

	successes := 0
	for success := range results {
		if success {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("optimistic lock successes = %d, want 1", successes)
	}
}

func assertSQLitePragmas(t *testing.T, db *sql.DB) {
	t.Helper()
	checks := []struct {
		pragma string
		want   string
	}{
		{pragma: "foreign_keys", want: "1"},
		{pragma: "journal_mode", want: "wal"},
		{pragma: "synchronous", want: "1"},
		{pragma: "busy_timeout", want: "5000"},
	}
	for _, check := range checks {
		var got string
		if err := db.QueryRow("PRAGMA " + check.pragma).Scan(&got); err != nil {
			t.Fatalf("read PRAGMA %s: %v", check.pragma, err)
		}
		if got != check.want {
			t.Fatalf("PRAGMA %s = %q, want %q", check.pragma, got, check.want)
		}
	}
}

type failingOutbox struct {
	err error
}

func (f failingOutbox) Enqueue(context.Context, database.Executor, string, string, string, int) error {
	return f.err
}

func assertCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}

func resetContractData(t *testing.T, db *sql.DB, dialect database.Dialect) {
	t.Helper()
	if dialect == database.Postgres {
		if _, err := db.Exec(`TRUNCATE TABLE news_items, page_revisions, media_assets, sessions, users, pages, audit_events, outbox_events RESTART IDENTITY CASCADE`); err != nil {
			t.Fatalf("reset postgres contract data: %v", err)
		}
		return
	}
	for _, table := range []string{"news_items", "page_revisions", "media_assets", "sessions", "users", "pages", "audit_events", "outbox_events"} {
		if _, err := db.Exec("DELETE FROM " + table); err != nil {
			t.Fatalf("reset sqlite contract data %s: %v", table, err)
		}
	}
}
