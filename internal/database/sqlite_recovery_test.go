package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRecoverSQLiteChecksIntegrityAndWAL(t *testing.T) {
	db, err := Open(context.Background(), SQLite, filepath.Join(t.TempDir(), "recover.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE recovery_probe(id INTEGER PRIMARY KEY); INSERT INTO recovery_probe(id) VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	if err := RecoverSQLite(context.Background(), db); err != nil {
		t.Fatalf("recover = %v", err)
	}
	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM recovery_probe`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("recovered rows = %d, %v", count, err)
	}
}
