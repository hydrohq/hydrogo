package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type Dialect string

const (
	SQLite   Dialect = "sqlite"
	Postgres Dialect = "postgres"
)

type OpenOptions struct {
	MaxOpenConns int
	MaxIdleConns int
}

type Executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Transactor struct {
	db *sql.DB
}

var ErrDatabaseRequired = errors.New("database is required")

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// RequireExecutor validates the common SQL adapter boundary before an
// adapter invokes methods on an executor.
func RequireExecutor(executor Executor) error {
	if executor == nil {
		return ErrDatabaseRequired
	}
	value := reflect.ValueOf(executor)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		if value.IsNil() {
			return ErrDatabaseRequired
		}
	}
	if !value.IsValid() {
		return ErrDatabaseRequired
	}
	return nil
}

func NewTransactor(db *sql.DB) Transactor {
	return Transactor{db: db}
}

func (t Transactor) Within(ctx context.Context, fn func(Executor) error) error {
	if t.db == nil {
		return ErrDatabaseRequired
	}
	if fn == nil {
		return errors.New("transaction callback is required")
	}
	ctx = nonNilContext(ctx)
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func Open(ctx context.Context, dialect Dialect, dsn string) (*sql.DB, error) {
	return OpenWithOptions(ctx, dialect, dsn, OpenOptions{})
}

func OpenWithOptions(ctx context.Context, dialect Dialect, dsn string, options OpenOptions) (*sql.DB, error) {
	ctx = nonNilContext(ctx)
	if dsn == "" {
		return nil, errors.New("database DSN is required")
	}
	if options.MaxOpenConns < 0 {
		return nil, errors.New("database max open connections cannot be negative")
	}
	if options.MaxIdleConns < 0 {
		return nil, errors.New("database max idle connections cannot be negative")
	}

	driver := ""
	switch dialect {
	case SQLite:
		driver = "sqlite"
		var err error
		dsn, err = normalizeSQLiteDSN(dsn)
		if err != nil {
			return nil, err
		}
	case Postgres:
		driver = "pgx"
	default:
		return nil, fmt.Errorf("unsupported database dialect %q", dialect)
	}

	if options.MaxOpenConns <= 0 {
		if dialect == SQLite {
			options.MaxOpenConns = 1
		} else {
			options.MaxOpenConns = 10
		}
	}
	if options.MaxIdleConns == 0 {
		if dialect == SQLite {
			options.MaxIdleConns = 1
		} else {
			options.MaxIdleConns = 5
		}
	}
	if options.MaxIdleConns > options.MaxOpenConns {
		return nil, errors.New("database max idle connections exceed max open connections")
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(options.MaxOpenConns)
	db.SetMaxIdleConns(options.MaxIdleConns)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func SQLiteDSN(path string) string {
	uriPath := filepath.ToSlash(path)
	if filepath.IsAbs(path) && !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	// Opaque keeps relative paths after the `file:` scheme. Using Path alone
	// serializes `hydro.sqlite` as `file://hydro.sqlite`, which SQLite treats as
	// an invalid URI authority instead of a relative filename.
	u := &url.URL{Scheme: "file", Opaque: uriPath}
	query := u.Query()
	query.Set("_busy_timeout", "5000")
	query.Set("_foreign_keys", "on")
	query.Set("_journal_mode", "WAL")
	query.Set("_synchronous", "NORMAL")
	u.RawQuery = query.Encode()
	return u.String()
}

func normalizeSQLiteDSN(dsn string) (string, error) {
	if !strings.HasPrefix(dsn, "file:") {
		return SQLiteDSN(dsn), nil
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("parse SQLite DSN: %w", err)
	}
	query := u.Query()
	query.Set("_busy_timeout", "5000")
	query.Set("_foreign_keys", "on")
	query.Set("_journal_mode", "WAL")
	query.Set("_synchronous", "NORMAL")
	u.RawQuery = query.Encode()
	return u.String(), nil
}
