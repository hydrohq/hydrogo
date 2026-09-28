package database

import (
	"context"
	"fmt"
	"os"
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

func TestCheckpointSQLiteTruncatesWAL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checkpoint.sqlite")
	db, err := Open(context.Background(), SQLite, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), `PRAGMA wal_autocheckpoint=0; CREATE TABLE checkpoint_probe(id INTEGER PRIMARY KEY, value TEXT)`); err != nil {
		t.Fatal(err)
	}
	for i := range 100 {
		if _, err := db.ExecContext(context.Background(), `INSERT INTO checkpoint_probe(value) VALUES (?)`, fmt.Sprintf("%0200d", i)); err != nil {
			t.Fatal(err)
		}
	}
	walPath := path + "-wal"
	before, err := os.Stat(walPath)
	if err != nil {
		t.Fatal(err)
	}
	if before.Size() == 0 {
		t.Fatal("expected writes to create WAL frames")
	}
	if err := CheckpointSQLite(context.Background(), db); err != nil {
		t.Fatalf("checkpoint = %v", err)
	}
	after, err := os.Stat(walPath)
	if err != nil {
		t.Fatal(err)
	}
	if after.Size() != 0 {
		t.Fatalf("WAL size after truncating checkpoint = %d, want 0", after.Size())
	}
	var rows int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM checkpoint_probe`).Scan(&rows); err != nil || rows != 100 {
		t.Fatalf("rows after checkpoint = %d, %v", rows, err)
	}
}
