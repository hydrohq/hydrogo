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

func TestReindexSQLiteRebuildsIndexesAndOptionalFTS(t *testing.T) {
	db, err := Open(context.Background(), SQLite, filepath.Join(t.TempDir(), "reindex.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE hydro_search_documents (rowid INTEGER PRIMARY KEY, title TEXT NOT NULL, body TEXT NOT NULL)`,
		`CREATE INDEX hydro_search_documents_title ON hydro_search_documents(title)`,
		`CREATE VIRTUAL TABLE hydro_search_fts USING fts5(title, body, content='hydro_search_documents', content_rowid='rowid')`,
		`INSERT INTO hydro_search_documents(title, body) VALUES ('HydroGo', 'searchable content')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := Reindex(context.Background(), db, SQLite); err != nil {
		t.Fatal(err)
	}
	var matches int
	if err := db.QueryRow(`SELECT COUNT(*) FROM hydro_search_fts WHERE hydro_search_fts MATCH 'HydroGo'`).Scan(&matches); err != nil {
		t.Fatal(err)
	}
	if matches != 1 {
		t.Fatalf("FTS matches after reindex = %d, want 1", matches)
	}
}

func TestReindexRejectsUnsupportedDialectAndNilDatabase(t *testing.T) {
	if err := Reindex(context.Background(), nil, SQLite); !errors.Is(err, ErrDatabaseRequired) {
		t.Fatalf("nil database error = %v", err)
	}
	db, err := Open(context.Background(), SQLite, filepath.Join(t.TempDir(), "unsupported-reindex.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Reindex(context.Background(), db, MySQL); !errors.Is(err, ErrReindexUnsupported) {
		t.Fatalf("unsupported dialect error = %v", err)
	}
}

func TestPostgresReindexQuotesDatabaseIdentifier(t *testing.T) {
	statement := postgresReindexStatement(`tenant"; DROP DATABASE shared; --`)
	if statement != `REINDEX DATABASE CONCURRENTLY "tenant""; DROP DATABASE shared; --"` {
		t.Fatalf("reindex SQL identifier was not quoted: %s", statement)
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
