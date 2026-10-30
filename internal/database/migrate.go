package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"
	"time"

	storagesqlite "github.com/ardaozyildirim/hydrogo/internal/storage/sqlite"
	"gopkg.in/yaml.v3"
)

func discoverMigrations(migrationFS fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(migrationFS, ".")
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		// "baseline" is the reserved directory of the schema baseline, never a migration.
		if e.IsDir() && e.Name() != "baseline" {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

func ApplyMigrations(ctx context.Context, db *sql.DB, dialect Dialect, migrationFS fs.FS) error {
	return ApplyMigrationsForModules(ctx, db, dialect, migrationFS, nil)
}

// ApplyMigrationsForModules applies only selected module migrations. A nil
// module set preserves the legacy all-modules behavior used by low-level
// tooling and tests. Core migrations are always included when filtering.
func ApplyMigrationsForModules(ctx context.Context, db *sql.DB, dialect Dialect, migrationFS fs.FS, enabled map[string]bool) error {
	ctx = nonNilContext(ctx)
	if db == nil {
		return errors.New("database is required")
	}
	if err := validateMigrationDialect(dialect); err != nil {
		return err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("open migration connection: %w", err)
	}
	defer conn.Close()
	return applyMigrationsOnConnection(ctx, conn, dialect, migrationFS, enabled, true)
}

func applyMigrationsOnConnection(ctx context.Context, conn *sql.Conn, dialect Dialect, migrationFS fs.FS, enabled map[string]bool, lock bool) error {
	if conn == nil {
		return errors.New("migration connection is required")
	}
	if err := validateMigrationDialect(dialect); err != nil {
		return err
	}
	enabled = canonicalEnabledModules(enabled)
	migrations, err := discoverMigrations(migrationFS)
	if err != nil {
		return fmt.Errorf("failed to discover migrations: %w", err)
	}
	allMigrations := append([]string(nil), migrations...)
	if enabled != nil {
		migrations = migrations[:0]
		for _, migrationID := range allMigrations {
			selected, err := migrationSelected(migrationFS, migrationID, enabled)
			if err != nil {
				return err
			}
			if selected {
				migrations = append(migrations, migrationID)
			}
		}
	}
	if lock {
		switch dialect {
		case Postgres:
			if err := acquireMigrationLock(ctx, conn); err != nil {
				return err
			}
			defer releaseMigrationLock(context.Background(), conn)
		case MySQL:
			if err := acquireMySQLMigrationLock(ctx, conn); err != nil {
				return err
			}
			defer releaseMySQLMigrationLock(context.Background(), conn)
		case SQLite:
			release, err := acquireSQLiteMigrationLock(ctx, conn)
			if err != nil {
				return err
			}
			defer release()
		}
	}
	if err := ensureMigrationLedger(ctx, conn, dialect); err != nil {
		return err
	}
	known := make(map[string]struct{}, len(allMigrations))
	for _, migrationID := range allMigrations {
		known[migrationID] = struct{}{}
	}
	if err := rejectUnknownMigrations(ctx, conn, known); err != nil {
		return err
	}
	existingMap := make(map[string]string)
	rows, err := conn.QueryContext(ctx, "SELECT id, checksum FROM schema_migrations")
	if err == nil {
		for rows.Next() {
			var id, checksum string
			if err := rows.Scan(&id, &checksum); err == nil {
				existingMap[id] = checksum
			}
		}
		_ = rows.Close()
	}
	var pending []string
	pendingChecksums := make(map[string]string)
	for _, migrationID := range migrations {
		content, err := fs.ReadFile(migrationFS, fmt.Sprintf("%s/%s.sql", migrationID, dialect))
		if err != nil {
			return fmt.Errorf("missing migration script %s/%s.sql: %w", migrationID, dialect, err)
		}
		checksum, err := migrationChecksum(migrationFS, migrationID, dialect, content)
		if err != nil {
			return err
		}
		if existingChecksum, ok := existingMap[migrationID]; ok {
			if existingChecksum != checksum {
				return fmt.Errorf("migration %s checksum mismatch", migrationID)
			}
			continue
		}
		pending = append(pending, migrationID)
		pendingChecksums[migrationID] = checksum
	}
	pending, err = applyBaselineGroups(ctx, conn, dialect, migrationFS, enabled, pending, pendingChecksums, existingMap)
	if err != nil {
		return err
	}
	if err := rejectPagesOnCoreCreatedDatabase(pending, existingMap); err != nil {
		return err
	}
	migrations = pending
	for index := 0; index < len(migrations); {
		transactional, err := migrationIsTransactional(migrationFS, migrations[index], dialect)
		if err != nil {
			return err
		}
		if !transactional {
			if err := applyNonTransactionalMigration(ctx, conn, dialect, migrationFS, migrations[index]); err != nil {
				return err
			}
			index++
			continue
		}
		start := index
		for index < len(migrations) {
			transactional, err := migrationIsTransactional(migrationFS, migrations[index], dialect)
			if err != nil {
				return err
			}
			if !transactional {
				break
			}
			index++
		}
		if err := withinMigrationTransaction(ctx, conn, func(tx Executor) error {
			for _, migrationID := range migrations[start:index] {
				if err := applyMigration(ctx, tx, dialect, migrationFS, migrationID); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// PendingMigrations reports selected, unapplied migrations and verifies every
// recorded checksum without creating or changing the migration ledger.
func PendingMigrations(ctx context.Context, db *sql.DB, dialect Dialect, migrationFS fs.FS, enabled map[string]bool) ([]string, error) {
	if db == nil {
		return nil, ErrDatabaseRequired
	}
	if err := validateMigrationDialect(dialect); err != nil {
		return nil, err
	}
	all, err := discoverMigrations(migrationFS)
	if err != nil {
		return nil, err
	}
	selected := make([]string, 0, len(all))
	for _, id := range all {
		if enabled == nil {
			selected = append(selected, id)
			continue
		}
		include, err := migrationSelected(migrationFS, id, canonicalEnabledModules(enabled))
		if err != nil {
			return nil, err
		}
		if include {
			selected = append(selected, id)
		}
	}
	hasLedger := false
	if dialect == SQLite {
		err = db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='schema_migrations')`).Scan(&hasLedger)
	} else if dialect == Postgres {
		err = db.QueryRowContext(ctx, `SELECT to_regclass('public.schema_migrations') IS NOT NULL`).Scan(&hasLedger)
	} else {
		err = db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='schema_migrations')`).Scan(&hasLedger)
	}
	if err != nil {
		return nil, fmt.Errorf("inspect migration ledger: %w", err)
	}
	if !hasLedger {
		return selected, nil
	}
	known := make(map[string]struct{}, len(all))
	for _, id := range all {
		known[id] = struct{}{}
	}
	if err := rejectUnknownMigrations(ctx, db, known); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, "SELECT id,checksum FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("read migration ledger: %w", err)
	}
	applied := make(map[string]string)
	for rows.Next() {
		var id, checksum string
		if err := rows.Scan(&id, &checksum); err != nil {
			_ = rows.Close()
			return nil, err
		}
		applied[id] = checksum
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()
	pending := make([]string, 0)
	for _, id := range selected {
		content, err := fs.ReadFile(migrationFS, fmt.Sprintf("%s/%s.sql", id, dialect))
		if err != nil {
			return nil, fmt.Errorf("missing migration script %s: %w", id, err)
		}
		checksum, err := migrationChecksum(migrationFS, id, dialect, content)
		if err != nil {
			return nil, err
		}
		if prior, ok := applied[id]; ok {
			if prior != checksum {
				return nil, fmt.Errorf("migration %s checksum mismatch", id)
			}
			continue
		}
		pending = append(pending, id)
	}
	return pending, nil
}

func canonicalEnabledModules(enabled map[string]bool) map[string]bool {
	if enabled == nil {
		return nil
	}
	canonical := make(map[string]bool, len(enabled))
	for name, active := range enabled {
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" && active {
			canonical[name] = true
		}
	}
	return canonical
}

// ErrPagesNeedsFreshDatabase reports that a database created without the pages
// module cannot enable it yet (ADR 0009, known limit).
var ErrPagesNeedsFreshDatabase = errors.New("the pages module cannot be enabled on a database that was created without it")

var ErrUnknownMigration = errors.New("database contains a migration unknown to this binary")

type migrationTransactionStarter interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type migrationQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func withinMigrationTransaction(ctx context.Context, starter migrationTransactionStarter, fn func(Executor) error) error {
	ctx = nonNilContext(ctx)
	tx, err := starter.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func ensureMigrationLedger(ctx context.Context, db interface {
	migrationTransactionStarter
	Executor
}, dialect Dialect) error {
	statement := `CREATE TABLE IF NOT EXISTS schema_migrations (
            id TEXT PRIMARY KEY,
            checksum TEXT NOT NULL
	)`
	if dialect == MySQL {
		statement = `CREATE TABLE IF NOT EXISTS schema_migrations (
				id VARCHAR(255) PRIMARY KEY,
				checksum CHAR(64) NOT NULL
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
		_, err := db.ExecContext(ctx, statement)
		return err
	}
	return withinMigrationTransaction(ctx, db, func(tx Executor) error {
		_, err := tx.ExecContext(ctx, statement)
		return err
	})
}

func applyMigration(ctx context.Context, executor Executor, dialect Dialect, migrationFS fs.FS, migrationID string) error {
	path := fmt.Sprintf("%s/%s.sql", migrationID, dialect)
	content, err := fs.ReadFile(migrationFS, path)
	if err != nil {
		return fmt.Errorf("missing migration script %s: %w", path, err)
	}
	checksum, err := migrationChecksum(migrationFS, migrationID, dialect, content)
	if err != nil {
		return err
	}
	if _, err := executor.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("failed executing migration %s: %w", migrationID, err)
	}
	if _, err := executor.ExecContext(ctx,
		"INSERT INTO schema_migrations (id, checksum) VALUES ("+placeholder(dialect, 1)+", "+placeholder(dialect, 2)+")",
		migrationID, checksum,
	); err != nil {
		return fmt.Errorf("failed recording migration %s: %w", migrationID, err)
	}
	return nil
}

func applyNonTransactionalMigration(ctx context.Context, db Executor, dialect Dialect, migrationFS fs.FS, migrationID string) error {
	if dialect == SQLite {
		manifest, err := readSQLiteShadowManifest(migrationFS, migrationID)
		if err != nil {
			return err
		}
		if manifest != nil {
			content, err := fs.ReadFile(migrationFS, migrationID+"/sqlite.sql")
			if err != nil {
				return err
			}
			checksum, err := migrationChecksum(migrationFS, migrationID, dialect, content)
			if err != nil {
				return err
			}
			var existing string
			err = db.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE id = ?`, migrationID).Scan(&existing)
			if err == nil {
				if existing != checksum {
					return fmt.Errorf("migration %s checksum mismatch", migrationID)
				}
				return nil
			}
			if !isNoRows(err) {
				return err
			}
			conn, ok := db.(*sql.Conn)
			if !ok {
				return errors.New("SQLite shadow migration requires a dedicated connection")
			}
			return storagesqlite.RebuildTable(ctx, conn, migrationID, manifest.spec(), func(recordCtx context.Context, recordConn *sql.Conn) error {
				_, err := recordConn.ExecContext(recordCtx, `INSERT INTO schema_migrations (id, checksum) VALUES (?, ?)`, migrationID, checksum)
				return err
			})
		}
	}
	if err := applyMigration(ctx, db, dialect, migrationFS, migrationID); err != nil {
		return err
	}
	return nil
}

const migrationAdvisoryLockKey int64 = 0x687964726f676f

func acquireMigrationLock(ctx context.Context, conn *sql.Conn) error {
	var ignored any
	if err := conn.QueryRowContext(ctx, "SELECT pg_advisory_lock($1)", migrationAdvisoryLockKey).Scan(&ignored); err != nil {
		return fmt.Errorf("acquire PostgreSQL migration lock: %w", err)
	}
	return nil
}

func releaseMigrationLock(ctx context.Context, conn *sql.Conn) {
	var ignored any
	_ = conn.QueryRowContext(ctx, "SELECT pg_advisory_unlock($1)", migrationAdvisoryLockKey).Scan(&ignored)
}

func acquireMySQLMigrationLock(ctx context.Context, conn *sql.Conn) error {
	var acquired sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(CONCAT(DATABASE(), ':hydrogo:migrate'), 30)").Scan(&acquired); err != nil {
		return fmt.Errorf("acquire MySQL migration lock: %w", err)
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		return errors.New("acquire MySQL migration lock: timed out")
	}
	return nil
}

func releaseMySQLMigrationLock(ctx context.Context, conn *sql.Conn) {
	var ignored sql.NullInt64
	_ = conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(CONCAT(DATABASE(), ':hydrogo:migrate'))").Scan(&ignored)
}

var inMemorySQLiteMigrationMu sync.Mutex

func sqliteDatabasePath(ctx context.Context, conn *sql.Conn) string {
	rows, err := conn.QueryContext(ctx, "PRAGMA database_list")
	if err != nil {
		return ""
	}
	defer rows.Close()
	for rows.Next() {
		var seq int
		var name, file string
		if err := rows.Scan(&seq, &name, &file); err == nil && name == "main" {
			return file
		}
	}
	return ""
}

func acquireSQLiteMigrationLock(ctx context.Context, conn *sql.Conn) (func(), error) {
	path := sqliteDatabasePath(ctx, conn)
	if path == "" || path == ":memory:" || strings.Contains(path, "mode=memory") {
		inMemorySQLiteMigrationMu.Lock()
		return func() {
			inMemorySQLiteMigrationMu.Unlock()
		}, nil
	}
	lockPath := path + ".lock"
	fl, err := acquireFileLock(ctx, lockPath, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("acquire SQLite migration lock: %w", err)
	}
	return func() {
		_ = fl.Release()
	}, nil
}

func migrationIsTransactional(migrationFS fs.FS, migrationID string, dialect Dialect) (bool, error) {
	if dialect == SQLite {
		manifest, err := readSQLiteShadowManifest(migrationFS, migrationID)
		if err != nil {
			return false, err
		}
		if manifest != nil {
			return false, nil
		}
	}
	data, err := fs.ReadFile(migrationFS, migrationID+"/migration.yaml")
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read migration metadata %s: %w", migrationID, err)
	}
	var metadata struct {
		Transactional map[string]bool `yaml:"transactional"`
	}
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return false, fmt.Errorf("parse migration metadata %s: %w", migrationID, err)
	}
	value, ok := metadata.Transactional[string(dialect)]
	if !ok {
		return false, fmt.Errorf("migration %s has no transactional metadata for %s", migrationID, dialect)
	}
	return value, nil
}

func CheckCompatibility(ctx context.Context, db *sql.DB, dialect Dialect, migrationFS fs.FS) error {
	return CheckCompatibilityForModules(ctx, db, dialect, migrationFS, nil)
}

func CheckCompatibilityForModules(ctx context.Context, db *sql.DB, dialect Dialect, migrationFS fs.FS, enabled map[string]bool) error {
	ctx = nonNilContext(ctx)
	if db == nil {
		return errors.New("database is required")
	}
	if err := validateMigrationDialect(dialect); err != nil {
		return err
	}
	migrations, err := discoverMigrations(migrationFS)
	if err != nil {
		return fmt.Errorf("failed to discover migrations: %w", err)
	}
	allMigrations := append([]string(nil), migrations...)
	known := make(map[string]struct{}, len(allMigrations))

	for _, migrationID := range allMigrations {
		known[migrationID] = struct{}{}
		if enabled != nil {
			selected, err := migrationSelected(migrationFS, migrationID, enabled)
			if err != nil {
				return err
			}
			if !selected {
				continue
			}
		}
		path := fmt.Sprintf("%s/%s.sql", migrationID, dialect)
		content, err := fs.ReadFile(migrationFS, path)
		if err != nil {
			return fmt.Errorf("missing migration script %s: %w", path, err)
		}
		want, err := migrationChecksum(migrationFS, migrationID, dialect, content)
		if err != nil {
			return err
		}

		var got string
		if err := db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE id = "+placeholder(dialect, 1), migrationID).Scan(&got); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("migration %s has not been applied yet; run \"hydro migrate\" first", migrationID)
			}
			return fmt.Errorf("schema is not ready for migration %s: %w", migrationID, err)
		}
		if got != want {
			return fmt.Errorf("migration %s checksum mismatch", migrationID)
		}
	}

	// An older binary must not serve a database that has already been advanced
	// by a newer binary. Checking only the migrations known to this binary would
	// make that rollback look compatible and could expose old code to a newer
	// schema. Treat the ledger as authoritative and fail closed on unknown IDs.
	if err := rejectUnknownMigrations(ctx, db, known); err != nil {
		return err
	}
	return nil
}

type migrationMetadata struct {
	Module          string   `yaml:"module"`
	RequiresModules []string `yaml:"requires_modules"`
}

type sqliteShadowMetadata struct {
	Table     string                       `yaml:"table"`
	CreateSQL string                       `yaml:"create_sql"`
	Columns   []storagesqlite.ShadowColumn `yaml:"columns"`
	BatchSize int                          `yaml:"batch_size"`
	AfterSwap []string                     `yaml:"after_swap"`
	Checksum  string                       `yaml:"checksum_sha256"`
}

func (m sqliteShadowMetadata) spec() storagesqlite.ShadowMigration {
	return storagesqlite.ShadowMigration{Table: m.Table, CreateSQL: m.CreateSQL, Columns: m.Columns, BatchSize: m.BatchSize, AfterSwap: m.AfterSwap}
}

func readSQLiteShadowManifest(migrationFS fs.FS, migrationID string) (*sqliteShadowMetadata, error) {
	data, err := fs.ReadFile(migrationFS, migrationID+"/migration.yaml")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var manifest struct {
		SQLiteShadow *sqliteShadowMetadata `yaml:"sqlite_shadow"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse SQLite shadow metadata %s: %w", migrationID, err)
	}
	if manifest.SQLiteShadow == nil {
		return nil, nil
	}
	actual, err := storagesqlite.Checksum(manifest.SQLiteShadow.spec())
	if err != nil {
		return nil, fmt.Errorf("checksum SQLite shadow metadata %s: %w", migrationID, err)
	}
	if strings.TrimSpace(manifest.SQLiteShadow.Checksum) == "" || !strings.EqualFold(manifest.SQLiteShadow.Checksum, actual) {
		return nil, fmt.Errorf("SQLite shadow metadata checksum mismatch for migration %s", migrationID)
	}
	return manifest.SQLiteShadow, nil
}

func migrationChecksum(migrationFS fs.FS, migrationID string, dialect Dialect, script []byte) (string, error) {
	digest := sha256.Sum256(script)
	scriptChecksum := hex.EncodeToString(digest[:])
	if dialect != SQLite {
		return scriptChecksum, nil
	}
	manifest, err := readSQLiteShadowManifest(migrationFS, migrationID)
	if err != nil || manifest == nil {
		return scriptChecksum, err
	}
	combined := append(append([]byte(nil), script...), 0)
	combined = append(combined, manifest.Checksum...)
	digest = sha256.Sum256(combined)
	return hex.EncodeToString(digest[:]), nil
}

func readMigrationMetadata(migrationFS fs.FS, migrationID string) (migrationMetadata, error) {
	data, err := fs.ReadFile(migrationFS, migrationID+"/migration.yaml")
	if errors.Is(err, fs.ErrNotExist) {
		return migrationMetadata{Module: "core"}, nil
	}
	if err != nil {
		return migrationMetadata{}, fmt.Errorf("read migration metadata %s: %w", migrationID, err)
	}
	var metadata migrationMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return migrationMetadata{}, fmt.Errorf("parse migration metadata %s: %w", migrationID, err)
	}
	metadata.Module = strings.TrimSpace(metadata.Module)
	if metadata.Module == "" {
		metadata.Module = "core"
	}
	for index := range metadata.RequiresModules {
		metadata.RequiresModules[index] = strings.TrimSpace(metadata.RequiresModules[index])
		if metadata.RequiresModules[index] == "" {
			return migrationMetadata{}, fmt.Errorf("migration %s has an empty required module", migrationID)
		}
		if metadata.RequiresModules[index] == metadata.Module {
			return migrationMetadata{}, fmt.Errorf("migration %s requires its owning module %q", migrationID, metadata.Module)
		}
		for previous := 0; previous < index; previous++ {
			if metadata.RequiresModules[previous] == metadata.RequiresModules[index] {
				return migrationMetadata{}, fmt.Errorf("migration %s requires module %q more than once", migrationID, metadata.RequiresModules[index])
			}
		}
	}
	return metadata, nil
}

func migrationModule(migrationFS fs.FS, migrationID string) (string, error) {
	metadata, err := readMigrationMetadata(migrationFS, migrationID)
	if err != nil {
		return "", err
	}
	return metadata.Module, nil
}

func migrationSelected(migrationFS fs.FS, migrationID string, enabled map[string]bool) (bool, error) {
	metadata, err := readMigrationMetadata(migrationFS, migrationID)
	if err != nil {
		return false, err
	}
	if metadata.Module != "core" && !enabled[metadata.Module] {
		return false, nil
	}
	for _, required := range metadata.RequiresModules {
		if required != "" && !enabled[required] {
			return false, nil
		}
	}
	return true, nil
}

func rejectUnknownMigrations(ctx context.Context, queryer migrationQueryer, known map[string]struct{}) error {
	rows, err := queryer.QueryContext(ctx, "SELECT id FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("read migration ledger: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("read migration ledger row: %w", err)
		}
		if _, ok := known[id]; !ok {
			return fmt.Errorf("%w: %s", ErrUnknownMigration, id)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read migration ledger rows: %w", err)
	}
	return nil
}

func validateMigrationDialect(dialect Dialect) error {
	if dialect != SQLite && dialect != Postgres && dialect != MySQL {
		return fmt.Errorf("unsupported database dialect %q", dialect)
	}
	return nil
}

func placeholder(dialect Dialect, position int) string {
	if dialect == Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}

func isNoRows(err error) bool {
	return err == sql.ErrNoRows
}

const (
	pagesFirstMigration   = "202609060001_transaction_contract"
	coreAuditEventsMarker = "202610210002_core_audit_events"
)

// rejectPagesOnCoreCreatedDatabase stops before any SQL runs when the pages
// chain would start on a database whose audit/outbox tables core already
// created: the first pages migration creates them again and fails, and on
// MySQL (no transactional DDL) the earlier statements would stay applied.
// Published migrations cannot be edited, so the fix is the baseline of ADR 0009;
// until then the operator gets the reason instead of a raw SQL error.
func rejectPagesOnCoreCreatedDatabase(pending []string, applied map[string]string) error {
	if _, ok := applied[coreAuditEventsMarker]; !ok {
		return nil
	}
	if _, ok := applied[pagesFirstMigration]; ok {
		return nil
	}
	for _, id := range pending {
		if id == pagesFirstMigration {
			return fmt.Errorf("%w: its audit and outbox tables already exist from core (ADR 0009); create the project with pages enabled from the start, or migrate the content into a new database", ErrPagesNeedsFreshDatabase)
		}
	}
	return nil
}
