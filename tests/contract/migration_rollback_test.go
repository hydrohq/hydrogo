package contract_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

// TestMigrationRollbackCompatibilityContract models the old binary's narrow
// pages query against a schema after all expand migrations. The new binary may
// write the richer projection first; a rolled-back binary must still be able
// to read and update its original columns without destroying the new fields.
func TestMigrationRollbackCompatibilityContract(t *testing.T) {
	tests := []struct {
		name    string
		dialect database.Dialect
		dsn     string
	}{
		{name: "sqlite", dialect: database.SQLite, dsn: filepath.Join(t.TempDir(), "rollback.sqlite")},
	}
	if dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL"); dsn != "" {
		tests = append(tests, struct {
			name    string
			dialect database.Dialect
			dsn     string
		}{name: "postgres", dialect: database.Postgres, dsn: dsn})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runMigrationRollbackCompatibility(t, tt.dialect, tt.dsn)
		})
	}
}

func runMigrationRollbackCompatibility(t *testing.T, dialect database.Dialect, dsn string) {
	t.Helper()
	ctx := context.Background()
	var db *sql.DB
	var err error
	if dialect == database.Postgres {
		db = openIsolatedPostgres(t, ctx)
	} else {
		db, err = database.Open(ctx, dialect, dsn)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
	}
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := database.CheckCompatibility(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatalf("check compatibility: %v", err)
	}

	pageID := fmt.Sprintf("rollback-%d", time.Now().UnixNano())
	placeholder := "?"
	if dialect == database.Postgres {
		placeholder = "$1"
	}
	// New code writes fields introduced by the expand migrations.
	insert := `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at)
		VALUES (` + joinPlaceholders(dialect, 11) + `)`
	args := []any{pageID, "new title", 2, "en", pageID, "new body", "draft", 2, 4, "2026-09-09", "2026-09-09"}
	if _, err := db.ExecContext(ctx, insert, args...); err != nil {
		t.Fatalf("new binary insert: %v", err)
	}

	// The old binary only knows the columns present in migration 001.
	oldSelect := `SELECT title, version FROM pages WHERE id = ` + placeholder
	var title string
	var version int
	if err := db.QueryRowContext(ctx, oldSelect, pageID).Scan(&title, &version); err != nil {
		t.Fatalf("old binary read: %v", err)
	}
	if title != "new title" || version != 2 {
		t.Fatalf("old binary read = %q/%d", title, version)
	}
	oldUpdate := `UPDATE pages SET title = ` + placeholder + `, version = ` + secondPlaceholder(dialect) + ` WHERE id = ` + thirdPlaceholder(dialect)
	if _, err := db.ExecContext(ctx, oldUpdate, "rolled back title", 3, pageID); err != nil {
		t.Fatalf("old binary update: %v", err)
	}
	var body string
	if err := db.QueryRowContext(ctx, `SELECT title, version, body FROM pages WHERE id = `+placeholder, pageID).Scan(&title, &version, &body); err != nil {
		t.Fatalf("new binary read after rollback write: %v", err)
	}
	if title != "rolled back title" || version != 3 || body != "new body" {
		t.Fatalf("rollback compatibility = %q/%d/%q", title, version, body)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM pages WHERE id = `+placeholder, pageID); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
}

func joinPlaceholders(dialect database.Dialect, count int) string {
	values := make([]byte, 0, count*3)
	for i := 1; i <= count; i++ {
		if i > 1 {
			values = append(values, ',', ' ')
		}
		if dialect == database.Postgres {
			values = append(values, fmt.Sprintf("$%d", i)...)
		} else {
			values = append(values, '?')
		}
	}
	return string(values)
}

func secondPlaceholder(dialect database.Dialect) string {
	if dialect == database.Postgres {
		return "$2"
	}
	return "?"
}

func thirdPlaceholder(dialect database.Dialect) string {
	if dialect == database.Postgres {
		return "$3"
	}
	return "?"
}

func TestSQLiteOnlineExpandAllowsConcurrentReadContract(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "online-expand.sqlite")
	first := fstest.MapFS{
		"0001_base/sqlite.sql": &fstest.MapFile{Data: []byte("CREATE TABLE legacy (id TEXT PRIMARY KEY, value TEXT NOT NULL);\n")},
	}
	all := fstest.MapFS{
		"0001_base/sqlite.sql":   &fstest.MapFile{Data: []byte("CREATE TABLE legacy (id TEXT PRIMARY KEY, value TEXT NOT NULL);\n")},
		"0002_expand/sqlite.sql": &fstest.MapFile{Data: []byte("ALTER TABLE legacy ADD COLUMN expanded TEXT;\n")},
	}
	db, err := database.Open(ctx, database.SQLite, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, first); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO legacy (id, value) VALUES ('read-1', 'before')"); err != nil {
		t.Fatal(err)
	}
	reader, err := database.Open(ctx, database.SQLite, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	readTx, err := reader.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer readTx.Rollback()
	var value string
	if err := readTx.QueryRowContext(ctx, "SELECT value FROM legacy WHERE id = 'read-1'").Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != "before" {
		t.Fatalf("read transaction value = %q", value)
	}
	deadline, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := database.ApplyMigrations(deadline, db, database.SQLite, all); err != nil {
		t.Fatalf("online expand while read transaction is active: %v", err)
	}
	if err := readTx.Commit(); err != nil {
		t.Fatal(err)
	}
	var expanded int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('legacy') WHERE name = 'expanded'").Scan(&expanded); err != nil {
		t.Fatal(err)
	}
	if expanded != 1 {
		t.Fatal("expand column was not applied")
	}
}

func TestPostgresNonTransactionalConcurrentIndexAllowsReadContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	db := openIsolatedPostgres(t, ctx)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	table := "online_contract_" + suffix
	baseID := "online_" + suffix + "_base"
	indexID := "online_" + suffix + "_index"
	if _, err := db.ExecContext(ctx, "CREATE TABLE "+table+" (id BIGSERIAL PRIMARY KEY, value TEXT NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(context.Background(), "DROP TABLE "+table)
	if _, err := db.ExecContext(ctx, "INSERT INTO "+table+" (value) VALUES ('before')"); err != nil {
		t.Fatal(err)
	}
	baseMigrations := fstest.MapFS{
		baseID + "/postgres.sql":   &fstest.MapFile{Data: []byte("SELECT 1;\n")},
		baseID + "/migration.yaml": &fstest.MapFile{Data: []byte("transactional:\n  postgres: true\n")},
	}
	allMigrations := fstest.MapFS{
		baseID + "/postgres.sql":    &fstest.MapFile{Data: []byte("SELECT 1;\n")},
		baseID + "/migration.yaml":  &fstest.MapFile{Data: []byte("transactional:\n  postgres: true\n")},
		indexID + "/postgres.sql":   &fstest.MapFile{Data: []byte("CREATE INDEX CONCURRENTLY " + table + "_value_idx ON " + table + " (value);\n")},
		indexID + "/migration.yaml": &fstest.MapFile{Data: []byte("transactional:\n  postgres: false\n")},
	}
	if err := database.ApplyMigrations(ctx, db, database.Postgres, baseMigrations); err != nil {
		t.Fatal(err)
	}
	readCtx, stopReads := context.WithCancel(ctx)
	var reads atomic.Int64
	var readErrors atomic.Int64
	ready := make(chan struct{})
	var readyOnce sync.Once
	var readers sync.WaitGroup
	readers.Add(1)
	go func() {
		defer readers.Done()
		for readCtx.Err() == nil {
			var value string
			if err := db.QueryRowContext(readCtx, "SELECT value FROM "+table+" WHERE id = 1").Scan(&value); err != nil {
				if readCtx.Err() == nil {
					readErrors.Add(1)
				}
				continue
			}
			if value == "before" {
				reads.Add(1)
				readyOnce.Do(func() { close(ready) })
			}
		}
	}()
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		stopReads()
		readers.Wait()
		t.Fatal("read loop did not become ready")
	}
	deadline, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := database.ApplyMigrations(deadline, db, database.Postgres, allMigrations); err != nil {
		stopReads()
		readers.Wait()
		t.Fatalf("concurrent index migration while short reads are active: %v", err)
	}
	stopReads()
	readers.Wait()
	if readErrors.Load() != 0 || reads.Load() == 0 {
		t.Fatalf("online reads = %d, errors = %d", reads.Load(), readErrors.Load())
	}
	var indexName string
	if err := db.QueryRowContext(ctx, "SELECT indexname FROM pg_indexes WHERE tablename = $1 AND indexname = $2", table, table+"_value_idx").Scan(&indexName); err != nil {
		t.Fatal(err)
	}
	if indexName != table+"_value_idx" {
		t.Fatalf("index name = %q", indexName)
	}
}

func TestPostgresConcurrentMigrationRunnersSerializeContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db := openIsolatedPostgres(t, ctx)
	migrationID := fmt.Sprintf("lock_%d", time.Now().UnixNano())
	migrationFS := fstest.MapFS{
		migrationID + "/postgres.sql":   &fstest.MapFile{Data: []byte("SELECT pg_sleep(0.4);\n")},
		migrationID + "/migration.yaml": &fstest.MapFile{Data: []byte("transactional:\n  postgres: true\n")},
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var runners sync.WaitGroup
	for i := 0; i < 2; i++ {
		runners.Add(1)
		go func() {
			defer runners.Done()
			<-start
			results <- database.ApplyMigrations(ctx, db, database.Postgres, migrationFS)
		}()
	}
	close(start)
	runners.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatalf("concurrent migration runner failed: %v", err)
		}
	}

	var applied int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE id = $1", migrationID).Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied != 1 {
		t.Fatalf("migration rows = %d, want 1", applied)
	}
}

func TestPostgresConcurrentIndexWaitsForLongSnapshotContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	db := openIsolatedPostgres(t, ctx)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	table := "snapshot_contract_" + suffix
	baseID := "snapshot_" + suffix + "_base"
	indexID := "snapshot_" + suffix + "_index"
	if _, err := db.ExecContext(ctx, "CREATE TABLE "+table+" (id BIGSERIAL PRIMARY KEY, value TEXT NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(context.Background(), "DROP TABLE "+table)
	if _, err := db.ExecContext(ctx, "INSERT INTO "+table+" (value) VALUES ('before')"); err != nil {
		t.Fatal(err)
	}
	base := fstest.MapFS{
		baseID + "/postgres.sql":   &fstest.MapFile{Data: []byte("SELECT 1;\n")},
		baseID + "/migration.yaml": &fstest.MapFile{Data: []byte("transactional:\n  postgres: true\n")},
	}
	all := fstest.MapFS{
		baseID + "/postgres.sql":    &fstest.MapFile{Data: []byte("SELECT 1;\n")},
		baseID + "/migration.yaml":  &fstest.MapFile{Data: []byte("transactional:\n  postgres: true\n")},
		indexID + "/postgres.sql":   &fstest.MapFile{Data: []byte("CREATE INDEX CONCURRENTLY " + table + "_value_idx ON " + table + " (value);\n")},
		indexID + "/migration.yaml": &fstest.MapFile{Data: []byte("transactional:\n  postgres: false\n")},
	}
	if err := database.ApplyMigrations(ctx, db, database.Postgres, base); err != nil {
		t.Fatal(err)
	}
	readTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var value string
	if err := readTx.QueryRowContext(ctx, "SELECT value FROM "+table+" WHERE id = 1").Scan(&value); err != nil {
		readTx.Rollback()
		t.Fatal(err)
	}
	if value != "before" {
		readTx.Rollback()
		t.Fatalf("snapshot value = %q", value)
	}
	result := make(chan error, 1)
	go func() {
		migrationCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		result <- database.ApplyMigrations(migrationCtx, db, database.Postgres, all)
	}()
	select {
	case err := <-result:
		readTx.Rollback()
		t.Fatalf("concurrent index completed while snapshot was open: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
	if err := readTx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err != nil {
		t.Fatalf("concurrent index after snapshot release: %v", err)
	}
}
