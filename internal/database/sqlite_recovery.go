package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// RecoverSQLite validates the main/WAL view and asks SQLite to restart its WAL
// checkpoint. It deliberately leaves existing -wal/-shm files to SQLite:
// deleting a valid WAL while another process may still own it can lose commits.
func RecoverSQLite(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return ErrDatabaseRequired
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var integrity string
	if err := db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&integrity); err != nil {
		return fmt.Errorf("sqlite quick_check: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(integrity), "ok") {
		return fmt.Errorf("sqlite quick_check reported: %s", strings.TrimSpace(integrity))
	}
	var busy, logFrames, checkpointed int
	if err := db.QueryRowContext(ctx, "PRAGMA wal_checkpoint(RESTART)").Scan(&busy, &logFrames, &checkpointed); err != nil {
		return fmt.Errorf("sqlite WAL checkpoint: %w", err)
	}
	if busy != 0 {
		return errors.New("sqlite WAL checkpoint is busy; another connection may be active")
	}
	return nil
}

// CheckpointSQLite flushes committed WAL frames into the database and truncates
// the WAL. Call it only after application workers and request handlers have
// stopped using the database.
func CheckpointSQLite(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return ErrDatabaseRequired
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var busy, logFrames, checkpointed int
	if err := db.QueryRowContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &logFrames, &checkpointed); err != nil {
		return fmt.Errorf("sqlite WAL shutdown checkpoint: %w", err)
	}
	if busy != 0 {
		return fmt.Errorf("sqlite WAL shutdown checkpoint is busy (log frames %d, checkpointed %d)", logFrames, checkpointed)
	}
	return nil
}
