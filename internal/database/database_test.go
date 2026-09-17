package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLiteDSNRelativePathIsValidURI(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	dsn := SQLiteDSN("hydro.sqlite")
	if strings.Contains(dsn, "file://hydro.sqlite") {
		t.Fatalf("relative SQLite DSN has an authority: %q", dsn)
	}
	db, err := Open(context.Background(), SQLite, "hydro.sqlite")
	if err != nil {
		t.Fatalf("open relative SQLite DSN: %v (dsn=%q)", err, dsn)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close SQLite database: %v", err)
	}
}

func TestTransactorRejectsMissingDatabaseAndCallback(t *testing.T) {
	if err := (Transactor{}).Within(context.Background(), func(Executor) error { return nil }); !errors.Is(err, ErrDatabaseRequired) {
		t.Fatalf("missing database error = %v, want %v", err, ErrDatabaseRequired)
	}
	db, err := Open(context.Background(), SQLite, filepath.Join(t.TempDir(), "transaction.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := NewTransactor(db).Within(context.Background(), nil); err == nil {
		t.Fatal("nil transaction callback was accepted")
	}
}

func TestDatabaseBoundariesNormalizeNilContext(t *testing.T) {
	db, err := Open(nil, SQLite, filepath.Join(t.TempDir(), "nil-context.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := NewTransactor(db).Within(nil, func(Executor) error { return nil }); err != nil {
		t.Fatalf("transaction with nil context failed: %v", err)
	}
}

func TestRequireExecutorRejectsNil(t *testing.T) {
	if err := RequireExecutor(nil); !errors.Is(err, ErrDatabaseRequired) {
		t.Fatalf("nil executor error = %v, want %v", err, ErrDatabaseRequired)
	}
	var typedNil *sql.DB
	if err := RequireExecutor(typedNil); !errors.Is(err, ErrDatabaseRequired) {
		t.Fatalf("typed-nil executor error = %v, want %v", err, ErrDatabaseRequired)
	}
}

func TestOpenWithOptionsValidatesPoolLimits(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "pool.sqlite")

	// Invalid: MaxOpenConns negative
	if _, err := OpenWithOptions(ctx, SQLite, dbPath, OpenOptions{MaxOpenConns: -1}); err == nil {
		t.Fatal("negative max open connections accepted")
	}

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
