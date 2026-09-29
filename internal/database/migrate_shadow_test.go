package database_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	storagesqlite "github.com/ardaozyildirim/hydrogo/internal/storage/sqlite"
	_ "modernc.org/sqlite"
)

func TestApplySQLiteShadowMigrationRecordsAtomicCutover(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "migration.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE records (id TEXT PRIMARY KEY, value TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO records VALUES ('r1','old')`); err != nil {
		t.Fatal(err)
	}
	spec := storagesqlite.ShadowMigration{
		Table:     "records",
		CreateSQL: `CREATE TABLE {{table}} (id TEXT PRIMARY KEY, value TEXT NOT NULL, extra TEXT NOT NULL DEFAULT '')`,
		Columns: []storagesqlite.ShadowColumn{
			{Name: "id", Copy: `id`, OnWrite: `NEW."id"`},
			{Name: "value", Copy: `value`, OnWrite: `NEW."value"`},
		},
		AfterSwap: []string{`CREATE INDEX idx_records_value ON records(value)`},
	}
	descriptorChecksum, err := storagesqlite.Checksum(spec)
	if err != nil {
		t.Fatal(err)
	}
	metadata := fmt.Sprintf("module: core\ntransactional:\n  sqlite: false\nsqlite_shadow:\n  table: %s\n  create_sql: %q\n  batch_size: %d\n  columns:\n    - name: %s\n      copy: %s\n      on_write: %q\n    - name: %s\n      copy: %s\n      on_write: %q\n  after_swap:\n    - %q\n  checksum_sha256: %s\n", spec.Table, spec.CreateSQL, spec.BatchSize, spec.Columns[0].Name, spec.Columns[0].Copy, spec.Columns[0].OnWrite, spec.Columns[1].Name, spec.Columns[1].Copy, spec.Columns[1].OnWrite, spec.AfterSwap[0], descriptorChecksum)
	migrationFS := fstest.MapFS{
		"202609290001_records_shadow/sqlite.sql":     &fstest.MapFile{Data: []byte("-- applied by the SQLite shadow migration runner\n")},
		"202609290001_records_shadow/migration.yaml": &fstest.MapFile{Data: []byte(metadata)},
	}
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrationFS); err != nil {
		t.Fatalf("apply shadow migration: %v", err)
	}
	var value, extra string
	if err := db.QueryRow(`SELECT value,extra FROM records WHERE id='r1'`).Scan(&value, &extra); err != nil {
		t.Fatal(err)
	}
	if value != "old" || extra != "" {
		t.Fatalf("migrated row=(%q,%q)", value, extra)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE id='202609290001_records_shadow'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("ledger count=%d err=%v", count, err)
	}
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrationFS); err != nil {
		t.Fatalf("reapply completed migration: %v", err)
	}
}
