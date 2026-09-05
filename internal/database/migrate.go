package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
)

const migrationID = "202609060001_transaction_contract"

func ApplyMigrations(ctx context.Context, db *sql.DB, dialect Dialect, migrationFS fs.FS) error {
	path := fmt.Sprintf("%s/%s.sql", migrationID, dialect)
	content, err := fs.ReadFile(migrationFS, path)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(content)
	checksum := hex.EncodeToString(digest[:])

	return NewTransactor(db).Within(ctx, func(tx Executor) error {
		if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
            id TEXT PRIMARY KEY,
            checksum TEXT NOT NULL
        )`); err != nil {
			return err
		}

		var existing string
		err := tx.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE id = "+placeholder(dialect, 1), migrationID).Scan(&existing)
		switch {
		case err == nil && existing != checksum:
			return fmt.Errorf("migration %s checksum mismatch", migrationID)
		case err == nil:
			return nil
		case !isNoRows(err):
			return err
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			"INSERT INTO schema_migrations (id, checksum) VALUES ("+placeholder(dialect, 1)+", "+placeholder(dialect, 2)+")",
			migrationID, checksum,
		)
		return err
	})
}

func CheckCompatibility(ctx context.Context, db *sql.DB, dialect Dialect, migrationFS fs.FS) error {
	path := fmt.Sprintf("%s/%s.sql", migrationID, dialect)
	content, err := fs.ReadFile(migrationFS, path)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(content)
	want := hex.EncodeToString(digest[:])

	var got string
	if err := db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE id = "+placeholder(dialect, 1), migrationID).Scan(&got); err != nil {
		return fmt.Errorf("schema is not ready: %w", err)
	}
	if got != want {
		return fmt.Errorf("migration %s checksum mismatch", migrationID)
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
