package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestShadowRebuildMirrorsConcurrentWritesAndResumes(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "shadow.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(3)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE records (id TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE schema_migrations (id TEXT PRIMARY KEY, checksum TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		if _, err := db.Exec(`INSERT INTO records(id,value) VALUES (?,?)`, string(rune('a'+i)), "old"); err != nil {
			t.Fatal(err)
		}
	}
	spec := ShadowMigration{
		Table:     "records",
		CreateSQL: `CREATE TABLE {{table}} (id TEXT PRIMARY KEY, value TEXT NOT NULL, extra TEXT NOT NULL DEFAULT '')`,
		BatchSize: 2,
		Columns: []ShadowColumn{
			{Name: "id", Copy: `id`, OnWrite: `NEW."id"`},
			{Name: "value", Copy: `value`, OnWrite: `NEW."value"`},
		},
		AfterSwap: []string{`CREATE INDEX idx_records_value ON records(value)`},
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	const migrationID = "test_shadow_records"
	shadow := "records__hydro_shadow_" + migrationID
	if err := prepare(ctx, conn, migrationID, spec, shadow); err != nil {
		t.Fatalf("prepare shadow: %v", err)
	}
	// Simulate writes by the old application version after trigger installation
	// but while the historical rows are still being copied.
	if _, err := db.Exec(`UPDATE records SET value='updated' WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM records WHERE id='b'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO records(id,value) VALUES ('z','new')`); err != nil {
		t.Fatal(err)
	}
	if err := copyBatches(ctx, conn, migrationID, spec, shadow); err != nil {
		t.Fatalf("copy batches: %v", err)
	}
	if err := cutover(ctx, conn, migrationID, spec, shadow, func(ctx context.Context, conn *sql.Conn) error {
		_, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations(id,checksum) VALUES (?,?)`, migrationID, "checksum")
		return err
	}); err != nil {
		t.Fatalf("cutover: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM records`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 6 {
		t.Fatalf("row count = %d, want 6", count)
	}
	var value, extra string
	if err := db.QueryRow(`SELECT value,extra FROM records WHERE id='a'`).Scan(&value, &extra); err != nil {
		t.Fatal(err)
	}
	if value != "updated" || extra != "" {
		t.Fatalf("updated row = (%q,%q)", value, extra)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM records WHERE id='b'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("deleted row count=%d err=%v", count, err)
	}
	if err := db.QueryRow(`SELECT value FROM records WHERE id='z'`).Scan(&value); err != nil || value != "new" {
		t.Fatalf("inserted row=%q err=%v", value, err)
	}
	var indexCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_records_value'`).Scan(&indexCount); err != nil || indexCount != 1 {
		t.Fatalf("index count=%d err=%v", indexCount, err)
	}
	var foreignKeys int
	if err := conn.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatalf("foreign_keys=%d err=%v", foreignKeys, err)
	}
}

func TestShadowRebuildRollsBackFailedCutoverAndCanResume(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "resume.sqlite"))
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
	if _, err := db.Exec(`INSERT INTO records VALUES ('a','value')`); err != nil {
		t.Fatal(err)
	}
	spec := ShadowMigration{Table: "records", CreateSQL: `CREATE TABLE {{table}} (id TEXT PRIMARY KEY, value TEXT)`, Columns: []ShadowColumn{{Name: "id", Copy: `id`, OnWrite: `NEW."id"`}, {Name: "value", Copy: `value`, OnWrite: `NEW."value"`}}, AfterSwap: []string{`CREATE INDEX idx_resume_value ON records(value)`}}
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	const migrationID = "test_shadow_resume"
	if err := RebuildTable(ctx, conn, migrationID, spec, func(context.Context, *sql.Conn) error { return errors.New("simulated ledger failure") }); err == nil {
		t.Fatal("simulated ledger failure unexpectedly succeeded")
	}
	var value string
	if err := db.QueryRow(`SELECT value FROM records WHERE id='a'`).Scan(&value); err != nil || value != "value" {
		t.Fatalf("source after failed cutover = %q err=%v", value, err)
	}
	changedSpec := spec
	changedSpec.Columns = append([]ShadowColumn(nil), spec.Columns...)
	changedSpec.Columns[1].Copy = `upper(value)`
	if err := RebuildTable(ctx, conn, migrationID, changedSpec, func(context.Context, *sql.Conn) error { return nil }); !errors.Is(err, ErrShadowInvalid) {
		t.Fatalf("changed descriptor resume error = %v", err)
	}
	if err := RebuildTable(ctx, conn, migrationID, spec, func(context.Context, *sql.Conn) error { return nil }); err != nil {
		t.Fatalf("resume after failed cutover/ledger: %v", err)
	}
}

func TestShadowRebuildRequiresWALAndRejectsUnsupportedTables(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "unsupported.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE records (id INTEGER PRIMARY KEY, value TEXT)`); err != nil {
		t.Fatal(err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	spec := ShadowMigration{Table: "records", CreateSQL: `CREATE TABLE {{table}} (id INTEGER PRIMARY KEY, value TEXT)`, Columns: []ShadowColumn{{Name: "id", Copy: `id`, OnWrite: `NEW."id"`}}}
	if err := RebuildTable(ctx, conn, "test_shadow_no_wal", spec, func(context.Context, *sql.Conn) error { return nil }); !errors.Is(err, ErrShadowUnsupported) {
		t.Fatalf("without WAL error = %v", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		t.Fatal(err)
	}
	if err := RebuildTable(ctx, conn, "test_shadow_integer_pk", spec, func(context.Context, *sql.Conn) error { return nil }); !errors.Is(err, ErrShadowUnsupported) {
		t.Fatalf("integer PK error = %v", err)
	}
}
