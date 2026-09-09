package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenWithOptionsValidatesPoolLimits(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "pool.sqlite")

	// Invalid: MaxIdleConns negative
	if _, err := OpenWithOptions(ctx, SQLite, dbPath, OpenOptions{MaxIdleConns: -1}); err == nil {
		t.Fatal("negative max idle connections accepted")
	}

	// Invalid: MaxIdleConns > MaxOpenConns
	if _, err := OpenWithOptions(ctx, SQLite, dbPath, OpenOptions{MaxOpenConns: 2, MaxIdleConns: 5}); err == nil {
		t.Fatal("max idle exceeding max open accepted")
	}

	// Valid pool configuration
	db, err := OpenWithOptions(ctx, SQLite, dbPath, OpenOptions{MaxOpenConns: 5, MaxIdleConns: 2})
	if err != nil {
		t.Fatalf("valid pool open failed: %v", err)
	}
	defer db.Close()

	if stats := db.Stats(); stats.MaxOpenConnections != 5 {
		t.Fatalf("stats max open = %d, want 5", stats.MaxOpenConnections)
	}
}

func TestOpenRejectsUnsupportedDialect(t *testing.T) {
	if _, err := Open(context.Background(), Dialect("oracle"), "dsn"); err == nil {
		t.Fatal("unsupported dialect accepted")
	}
}
