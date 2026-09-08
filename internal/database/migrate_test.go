package database_test

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

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
