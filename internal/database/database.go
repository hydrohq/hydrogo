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
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/telemetry"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type Dialect string

const (
	SQLite   Dialect = "sqlite"
	Postgres Dialect = "postgres"
	MySQL    Dialect = "mysql"
)

type OpenOptions struct {
	MaxOpenConns  int
	MaxIdleConns  int
	SQLiteProfile string
}

const (
	SQLiteCompactProfile = "compact"
	SQLiteVPSProfile     = "vps"
)

type Executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// QueryExecutor is the row-producing extension used by repositories that
// need to scan a bounded result set. It intentionally stays separate from
// Executor so existing narrow fakes and adapter contracts remain source
// compatible.
type QueryExecutor interface {
	Executor
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Transactor struct {
	db                 *sql.DB
	dialect            Dialect
	requireTenantScope bool
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

// NewTenantTransactor binds the request/job tenant to PostgreSQL's
// transaction-local RLS settings. SQLite keeps the same explicit scope
// contract in repositories but does not execute PostgreSQL session settings.
// When requireTenantScope is false, PostgreSQL is explicitly put in
// single-tenant mode for the transaction.
func NewTenantTransactor(db *sql.DB, dialect Dialect, requireTenantScope bool) Transactor {
	return Transactor{db: db, dialect: dialect, requireTenantScope: requireTenantScope}
}

func (t Transactor) Within(ctx context.Context, fn func(Executor) error) error {
	if t.db == nil {
		return ErrDatabaseRequired
	}
	if fn == nil {
		return errors.New("transaction callback is required")
	}
	ctx = nonNilContext(ctx)
	ctx, span := telemetry.StartSpan(ctx, "database", "transaction")
	defer span.End()
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if t.dialect == Postgres {
		if t.requireTenantScope {
			scope, scopeErr := tenant.Require(ctx)
			if scopeErr != nil {
				return scopeErr
			}
			if err := tenant.BindPostgresTransaction(ctx, tx, scope); err != nil {
				return err
			}
		} else if err := tenant.BindPostgresSingleTenantTransaction(ctx, tx); err != nil {
			return err
		}
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// WithinQuery is the query-producing counterpart of Within. The callback and
// its rows always use the same transaction as the tenant RLS binding.
func (t Transactor) WithinQuery(ctx context.Context, fn func(QueryExecutor) error) error {
	if t.db == nil {
		return ErrDatabaseRequired
	}
	if fn == nil {
		return errors.New("transaction callback is required")
	}
	ctx = nonNilContext(ctx)
	ctx, span := telemetry.StartSpan(ctx, "database", "transaction.query")
	defer span.End()
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if t.dialect == Postgres {
		if t.requireTenantScope {
			scope, scopeErr := tenant.Require(ctx)
			if scopeErr != nil {
				return scopeErr
			}
			if err := tenant.BindPostgresTransaction(ctx, tx, scope); err != nil {
				return err
			}
		} else if err := tenant.BindPostgresSingleTenantTransaction(ctx, tx); err != nil {
			return err
		}
	}
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
	if options.SQLiteProfile != "" && options.SQLiteProfile != SQLiteCompactProfile && options.SQLiteProfile != SQLiteVPSProfile {
		return nil, fmt.Errorf("unsupported SQLite profile %q", options.SQLiteProfile)
	}

	driver := ""
	switch dialect {
	case SQLite:
		driver = "sqlite"
		var err error
		dsn, err = normalizeSQLiteDSN(dsn, options.SQLiteProfile)
		if err != nil {
			return nil, err
		}
	case Postgres:
		driver = "pgx"
	case MySQL:
		driver = "mysql"
		var err error
		dsn, err = prepareMySQLDSN(dsn)
		if err != nil {
			return nil, fmt.Errorf("invalid MySQL DSN: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported database dialect %q", dialect)
	}
	if err := ValidateDialectAvailable(dialect); err != nil {
		return nil, err
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
	if dialect == MySQL {
		db.SetConnMaxLifetime(3 * time.Minute)
	}
	return db, nil
}

// ValidateDialectAvailable checks that this build includes the selected SQL
// adapter without opening a connection. Edition composition can use it during
// config preflight; Compact builds intentionally omit PostgreSQL.
func ValidateDialectAvailable(dialect Dialect) error {
	var driver string
	switch dialect {
	case SQLite:
		driver = "sqlite"
	case Postgres:
		driver = "pgx"
	case MySQL:
		driver = "mysql"
	default:
		return fmt.Errorf("unsupported database dialect %q", dialect)
	}
	if !driverRegistered(driver) {
		return fmt.Errorf("database driver %q is not included in this Edition build", driver)
	}
	return nil
}

func driverRegistered(name string) bool {
	for _, registered := range sql.Drivers() {
		if registered == name {
			return true
		}
	}
	return false
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
	applySQLiteProfile(query, SQLiteCompactProfile)
	u.RawQuery = query.Encode()
	return u.String()
}

func normalizeSQLiteDSN(dsn, profile string) (string, error) {
	if !strings.HasPrefix(dsn, "file:") {
		uri, err := url.Parse(SQLiteDSN(dsn))
		if err != nil {
			return "", err
		}
		query := uri.Query()
		applySQLiteProfile(query, profile)
		uri.RawQuery = query.Encode()
		return uri.String(), nil
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("parse SQLite DSN: %w", err)
	}
	query := u.Query()
	applySQLiteProfile(query, profile)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func applySQLiteProfile(query url.Values, profile string) {
	if profile == "" {
		profile = SQLiteCompactProfile
	}
	query.Set("_busy_timeout", "5000")
	query.Set("_foreign_keys", "on")
	query.Set("_journal_mode", "WAL")
	query.Set("_synchronous", "NORMAL")
	query.Del("_pragma")
	switch profile {
	case SQLiteCompactProfile:
		query.Add("_pragma", "cache_size=-2000")
	case SQLiteVPSProfile:
		query.Set("_busy_timeout", "10000")
		query.Add("_pragma", "cache_size=-8000")
		query.Add("_pragma", "mmap_size=67108864")
	default:
		// OpenWithOptions validates profiles before reaching this helper.
	}
}
