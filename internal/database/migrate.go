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

	"gopkg.in/yaml.v3"
)

func discoverMigrations(migrationFS fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(migrationFS, ".")
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

func ApplyMigrations(ctx context.Context, db *sql.DB, dialect Dialect, migrationFS fs.FS) error {
	if db == nil {
		return errors.New("database is required")
	}
	migrations, err := discoverMigrations(migrationFS)
	if err != nil {
		return fmt.Errorf("failed to discover migrations: %w", err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("open migration connection: %w", err)
	}
	defer conn.Close()
	if dialect == Postgres {
		if err := acquireMigrationLock(ctx, conn); err != nil {
			return err
		}
		defer releaseMigrationLock(context.Background(), conn)
	}
	if err := ensureMigrationLedger(ctx, conn); err != nil {
		return err
	}
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

var ErrUnknownMigration = errors.New("database contains a migration unknown to this binary")

type migrationTransactionStarter interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

func withinMigrationTransaction(ctx context.Context, starter migrationTransactionStarter, fn func(Executor) error) error {
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

func ensureMigrationLedger(ctx context.Context, db migrationTransactionStarter) error {
	return withinMigrationTransaction(ctx, db, func(tx Executor) error {
		_, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
            id TEXT PRIMARY KEY,
            checksum TEXT NOT NULL
        )`)
		return err
	})
}

func applyMigration(ctx context.Context, executor Executor, dialect Dialect, migrationFS fs.FS, migrationID string) error {
	path := fmt.Sprintf("%s/%s.sql", migrationID, dialect)
	content, err := fs.ReadFile(migrationFS, path)
	if err != nil {
		return fmt.Errorf("missing migration script %s: %w", path, err)
	}
	digest := sha256.Sum256(content)
	checksum := hex.EncodeToString(digest[:])
	var existing string
	err = executor.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE id = "+placeholder(dialect, 1), migrationID).Scan(&existing)
	switch {
	case err == nil && existing != checksum:
		return fmt.Errorf("migration %s checksum mismatch", migrationID)
	case err == nil:
		return nil
	case !isNoRows(err):
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

func migrationIsTransactional(migrationFS fs.FS, migrationID string, dialect Dialect) (bool, error) {
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
	migrations, err := discoverMigrations(migrationFS)
	if err != nil {
		return fmt.Errorf("failed to discover migrations: %w", err)
	}
	known := make(map[string]struct{}, len(migrations))

	for _, migrationID := range migrations {
		known[migrationID] = struct{}{}
		path := fmt.Sprintf("%s/%s.sql", migrationID, dialect)
		content, err := fs.ReadFile(migrationFS, path)
		if err != nil {
			return fmt.Errorf("missing migration script %s: %w", path, err)
		}
		digest := sha256.Sum256(content)
		want := hex.EncodeToString(digest[:])

		var got string
		if err := db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE id = "+placeholder(dialect, 1), migrationID).Scan(&got); err != nil {
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
	rows, err := db.QueryContext(ctx, "SELECT id FROM schema_migrations")
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

func placeholder(dialect Dialect, position int) string {
	if dialect == Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}

func isNoRows(err error) bool {
	return err == sql.ErrNoRows
}
