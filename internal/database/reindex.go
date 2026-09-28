package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrReindexUnsupported = errors.New("database reindex is supported only for SQLite and PostgreSQL")

// Reindex rebuilds database indexes without wrapping PostgreSQL's concurrent
// operation in a transaction. SQLite's external-content FTS index is rebuilt
// when that optional module table exists.
func Reindex(ctx context.Context, db *sql.DB, dialect Dialect) error {
	if db == nil {
		return ErrDatabaseRequired
	}
	switch dialect {
	case SQLite:
		if _, err := db.ExecContext(ctx, "REINDEX"); err != nil {
			return fmt.Errorf("reindex SQLite indexes: %w", err)
		}
		var ftsExists bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='hydro_search_fts')`).Scan(&ftsExists); err != nil {
			return fmt.Errorf("inspect SQLite full-text index: %w", err)
		}
		if ftsExists {
			if _, err := db.ExecContext(ctx, `INSERT INTO hydro_search_fts(hydro_search_fts) VALUES ('rebuild')`); err != nil {
				return fmt.Errorf("rebuild SQLite full-text index: %w", err)
			}
		}
		var busy, logFrames, checkpointed int
		if err := db.QueryRowContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &logFrames, &checkpointed); err != nil {
			return fmt.Errorf("checkpoint SQLite WAL: %w", err)
		}
		if busy != 0 {
			return fmt.Errorf("SQLite WAL checkpoint is busy (log frames %d, checkpointed %d)", logFrames, checkpointed)
		}
		return nil
	case Postgres:
		var name string
		if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&name); err != nil {
			return fmt.Errorf("read PostgreSQL database name: %w", err)
		}
		if _, err := db.ExecContext(ctx, postgresReindexStatement(name)); err != nil {
			return fmt.Errorf("reindex PostgreSQL database: %w", err)
		}
		return nil
	default:
		return ErrReindexUnsupported
	}
}

func postgresReindexStatement(databaseName string) string {
	return "REINDEX DATABASE CONCURRENTLY \"" + strings.ReplaceAll(databaseName, "\"", "\"\"") + "\""
}
