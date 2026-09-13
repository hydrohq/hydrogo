package database_test

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestMigrationAPIsRejectMissingDatabaseAndDialect(t *testing.T) {
	ctx := context.Background()
	if err := database.ApplyMigrations(ctx, nil, database.SQLite, migrations.FS); err == nil {
		t.Fatal("ApplyMigrations accepted a nil database")
	}
	if err := database.CheckCompatibility(ctx, nil, database.SQLite, migrations.FS); err == nil {
		t.Fatal("CheckCompatibility accepted a nil database")
	}
	db := &sql.DB{}
	if err := database.ApplyMigrations(ctx, db, database.Dialect("oracle"), migrations.FS); err == nil {
		t.Fatal("ApplyMigrations accepted an unsupported dialect")
	}
	if err := database.CheckCompatibility(ctx, db, database.Dialect("oracle"), migrations.FS); err == nil {
		t.Fatal("CheckCompatibility accepted an unsupported dialect")
	}
}

func TestMigrationAPIsNormalizeNilContext(t *testing.T) {
	db, err := database.Open(nil, database.SQLite, filepath.Join(t.TempDir(), "nil-migration-context.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(nil, db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("migrations with nil context failed: %v", err)
	}
	if err := database.CheckCompatibility(nil, db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("compatibility with nil context failed: %v", err)
	}
}

func TestSQLiteMigrationRestartAndCompatibilityContract(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "hydro.sqlite")
	db, err := database.Open(ctx, database.SQLite, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := database.CheckCompatibility(ctx, db, database.SQLite, migrations.FS); err != nil {
		db.Close()
		t.Fatalf("compatibility after first apply: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	// A restarted binary sees the same ledger and performs no duplicate DDL.
	db, err = database.Open(ctx, database.SQLite, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("restart apply: %v", err)
	}
	var applied int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied != migrationCount(t) {
		t.Fatalf("migration rows = %d, want %d", applied, migrationCount(t))
	}

	// A binary with changed SQL must refuse to serve the existing schema.
	var firstID string
	if err := db.QueryRowContext(ctx, "SELECT id FROM schema_migrations ORDER BY id LIMIT 1").Scan(&firstID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE schema_migrations SET checksum = ? WHERE id = ?", strings.Repeat("0", 64), firstID); err != nil {
		t.Fatal(err)
	}
	if err := database.CheckCompatibility(ctx, db, database.SQLite, migrations.FS); err == nil {
		t.Fatal("checksum mismatch was accepted")
	}
}

func TestSQLiteModuleFilteringLeavesInactiveModuleSchemaAbsent(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "modules.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	enabled := map[string]bool{"pages": true, "media": true}
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, enabled); err != nil {
		t.Fatalf("filtered migrations failed: %v", err)
	}
	for _, table := range []struct {
		name string
		want bool
	}{
		{name: "schema_migrations", want: true},
		{name: "media_assets", want: true},
		{name: "media_variants", want: true},
		{name: "commerce_products", want: false},
		{name: "news_items", want: false},
	} {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table.name).Scan(&count); err != nil {
			t.Fatalf("inspect table %q: %v", table.name, err)
		}
		if (count == 1) != table.want {
			t.Fatalf("table %q present = %v, want %v", table.name, count == 1, table.want)
		}
	}

	if err := database.CheckCompatibilityForModules(ctx, db, database.SQLite, migrations.FS, enabled); err != nil {
		t.Fatalf("filtered compatibility check failed: %v", err)
	}
}

func TestNonTransactionalMigrationUsesMetadataExecutionMode(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "nontransactional.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	migrationFS := fstest.MapFS{
		"0001_base/sqlite.sql":       &fstest.MapFile{Data: []byte("CREATE TABLE sample (id INTEGER PRIMARY KEY);\n")},
		"0001_base/migration.yaml":   &fstest.MapFile{Data: []byte("transactional:\n  sqlite: true\n")},
		"0002_vacuum/sqlite.sql":     &fstest.MapFile{Data: []byte("VACUUM;\n")},
		"0002_vacuum/migration.yaml": &fstest.MapFile{Data: []byte("transactional:\n  sqlite: false\n")},
	}
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrationFS); err != nil {
		t.Fatal(err)
	}
	var applied int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied != 2 {
		t.Fatalf("migration rows = %d, want 2", applied)
	}
}

func TestCompatibilityRejectsUnknownAppliedMigration(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "future.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO schema_migrations (id, checksum) VALUES (?, ?)", "209901010001_future", strings.Repeat("f", 64)); err != nil {
		t.Fatal(err)
	}
	if err := database.CheckCompatibility(ctx, db, database.SQLite, migrations.FS); !errors.Is(err, database.ErrUnknownMigration) {
		t.Fatalf("unknown migration compatibility error = %v, want %v", err, database.ErrUnknownMigration)
	}
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); !errors.Is(err, database.ErrUnknownMigration) {
		t.Fatalf("unknown migration apply error = %v, want %v", err, database.ErrUnknownMigration)
	}
}

func migrationCount(t *testing.T) int {
	t.Helper()
	entries, err := sqlFSReadDir(migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

func sqlFSReadDir(fsys fs.ReadDirFS) ([]fs.DirEntry, error) {
	return fsys.ReadDir(".")
}
