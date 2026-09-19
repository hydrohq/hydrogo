package cache

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrInvalidSQLiteMonitor = errors.New("invalid sqlite cache monitor configuration")

// SQLiteDataVersionMonitor observes writes made by another SQLite connection.
// Application mutations should still invalidate immediately; this watcher is
// the bounded safety net for operator scripts/cron jobs that bypass HydroGo.
type SQLiteDataVersionMonitor struct {
	ReadVersion func(context.Context) (int64, error)
	Interval    time.Duration
	MaxInterval time.Duration
	OnChange    func()
	OnError     func(error)
	sleep       func(context.Context, time.Duration) error
}

// NewSQLiteDataVersionMonitor binds the watcher to a database handle. Compact
// deployments should use a dedicated DB handle/connection policy so the PRAGMA
// observes a stable SQLite connection.
func NewSQLiteDataVersionMonitor(db *sql.DB) SQLiteDataVersionMonitor {
	return SQLiteDataVersionMonitor{ReadVersion: func(ctx context.Context) (int64, error) {
		var version int64
		if db == nil {
			return 0, ErrInvalidSQLiteMonitor
		}
		err := db.QueryRowContext(ctx, "PRAGMA data_version").Scan(&version)
		return version, err
	}}
}

// Run polls SQLite data_version with bounded exponential backoff on query
// errors. It exits on context cancellation and never spins on a broken DB.
func (m SQLiteDataVersionMonitor) Run(ctx context.Context) error {
	base, max, err := m.bounds()
	if err != nil {
		return err
	}
	if m.ReadVersion == nil || m.OnChange == nil {
		return ErrInvalidSQLiteMonitor
	}
	sleep := m.sleep
	if sleep == nil {
		sleep = sqliteSleepContext
	}
	previous, err := m.ReadVersion(ctx)
	if err != nil {
		if m.OnError != nil {
			m.OnError(err)
		}
		previous = -1
	}
	interval := base
	for {
		if err := sleep(ctx, interval); err != nil {
			return err
		}
		current, readErr := m.ReadVersion(ctx)
		if readErr != nil {
			if m.OnError != nil {
				m.OnError(readErr)
			}
			interval = sqliteNextPollInterval(interval, max)
			continue
		}
		if previous >= 0 && current != previous {
			m.OnChange()
		}
		previous = current
		interval = base
	}
}

func sqliteSleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func sqliteNextPollInterval(current, max time.Duration) time.Duration {
	if current >= max {
		return max
	}
	next := current * 2
	if next < current || next > max {
		return max
	}
	return next
}

func (m SQLiteDataVersionMonitor) bounds() (time.Duration, time.Duration, error) {
	base := m.Interval
	if base <= 0 {
		base = time.Second
	}
	max := m.MaxInterval
	if max <= 0 {
		max = 30 * time.Second
	}
	if base < 100*time.Millisecond || base > max || max > 10*time.Minute {
		return 0, 0, ErrInvalidSQLiteMonitor
	}
	return base, max, nil
}
