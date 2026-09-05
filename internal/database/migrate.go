package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
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
	migrations, err := discoverMigrations(migrationFS)
	if err != nil {
		return fmt.Errorf("failed to discover migrations: %w", err)
	}

	return NewTransactor(db).Within(ctx, func(tx Executor) error {
		if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
            id TEXT PRIMARY KEY,
            checksum TEXT NOT NULL
        )`); err != nil {
			return err
		}

		for _, migrationID := range migrations {
			path := fmt.Sprintf("%s/%s.sql", migrationID, dialect)
			content, err := fs.ReadFile(migrationFS, path)
			if err != nil {
				return fmt.Errorf("missing migration script %s: %w", path, err)
			}
			digest := sha256.Sum256(content)
			checksum := hex.EncodeToString(digest[:])

			var existing string
			err = tx.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE id = "+placeholder(dialect, 1), migrationID).Scan(&existing)
			switch {
			case err == nil && existing != checksum:
				return fmt.Errorf("migration %s checksum mismatch", migrationID)
			case err == nil:
				continue
			case !isNoRows(err):
				return err
			}

			if _, err := tx.ExecContext(ctx, string(content)); err != nil {
				return fmt.Errorf("failed executing migration %s: %w", migrationID, err)
			}
			_, err = tx.ExecContext(ctx,
				"INSERT INTO schema_migrations (id, checksum) VALUES ("+placeholder(dialect, 1)+", "+placeholder(dialect, 2)+")",
				migrationID, checksum,
			)
			if err != nil {
				return fmt.Errorf("failed recording migration %s: %w", migrationID, err)
			}
		}
		return nil
	})
}

func CheckCompatibility(ctx context.Context, db *sql.DB, dialect Dialect, migrationFS fs.FS) error {
	migrations, err := discoverMigrations(migrationFS)
	if err != nil {
		return fmt.Errorf("failed to discover migrations: %w", err)
	}

	for _, migrationID := range migrations {
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
