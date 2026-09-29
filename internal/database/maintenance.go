package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"
)

type MaintenanceReport struct {
	Dialect               Dialect
	StartedAt             time.Time
	Duration              time.Duration
	ReclaimedBytes        int64
	WALCheckpointDeferred bool
}

// Maintain reclaims SQLite free pages in WAL mode or refreshes PostgreSQL
// planner statistics. SQLite VACUUM runs outside a transaction so WAL readers
// retain their snapshots while the replacement database is built.
func Maintain(ctx context.Context, db *sql.DB, dialect Dialect, sqlitePath string) (MaintenanceReport, error) {
	report := MaintenanceReport{Dialect: dialect, StartedAt: time.Now().UTC()}
	if db == nil {
		return report, ErrDatabaseRequired
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var before int64
	if dialect == SQLite {
		if sqlitePath == "" {
			return report, errors.New("SQLite maintenance requires a file path")
		}
		before = sqliteFootprint(sqlitePath)
		if _, err := db.ExecContext(ctx, "VACUUM"); err != nil {
			return report, fmt.Errorf("vacuum SQLite database: %w", err)
		}
		var busy, logFrames, checkpointed int
		if err := db.QueryRowContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &logFrames, &checkpointed); err != nil {
			return report, fmt.Errorf("checkpoint SQLite WAL after vacuum: %w", err)
		}
		report.WALCheckpointDeferred = busy != 0
		if reclaimed := before - sqliteFootprint(sqlitePath); reclaimed > 0 {
			report.ReclaimedBytes = reclaimed
		}
	} else if dialect == Postgres {
		if _, err := db.ExecContext(ctx, "ANALYZE"); err != nil {
			return report, fmt.Errorf("analyze PostgreSQL tables: %w", err)
		}
	} else {
		return report, fmt.Errorf("database maintenance is unsupported for %q", dialect)
	}
	report.Duration = time.Since(report.StartedAt)
	return report, nil
}

func sqliteFootprint(path string) int64 {
	var total int64
	for _, suffix := range []string{"", "-wal"} {
		if info, err := os.Stat(path + suffix); err == nil {
			total += info.Size()
		}
	}
	return total
}

// NextWeeklyMaintenance returns the next Sunday at 03:00 UTC.
func NextWeeklyMaintenance(now time.Time) time.Time {
	utc := now.UTC()
	days := (int(time.Sunday) - int(utc.Weekday()) + 7) % 7
	next := time.Date(utc.Year(), utc.Month(), utc.Day()+days, 3, 0, 0, 0, time.UTC)
	if !next.After(utc) {
		next = next.AddDate(0, 0, 7)
	}
	return next
}

func RunWeeklyMaintenance(ctx context.Context, maintain func(context.Context)) {
	if maintain == nil {
		return
	}
	for {
		delay := time.Until(NextWeeklyMaintenance(time.Now()))
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			maintain(ctx)
		}
	}
}
