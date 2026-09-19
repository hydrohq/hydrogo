package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSQLiteDataVersionMonitorClearsOnExternalVersionChange(t *testing.T) {
	versions := []int64{7, 7, 8}
	reads := 0
	changes := 0
	var delays []time.Duration
	monitor := SQLiteDataVersionMonitor{
		ReadVersion: func(context.Context) (int64, error) {
			value := versions[reads]
			reads++
			return value, nil
		},
		Interval:    100 * time.Millisecond,
		MaxInterval: 400 * time.Millisecond,
		OnChange:    func() { changes++ },
		sleep: func(ctx context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			if len(delays) == 3 {
				return context.Canceled
			}
			return nil
		},
	}
	if err := monitor.Run(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("monitor error = %v", err)
	}
	if changes != 1 || reads != 3 || len(delays) != 3 || delays[0] != 100*time.Millisecond || delays[1] != 100*time.Millisecond || delays[2] != 100*time.Millisecond {
		t.Fatalf("reads=%d changes=%d delays=%v", reads, changes, delays)
	}
}

func TestSQLiteDataVersionMonitorBacksOffReadErrors(t *testing.T) {
	reads := 0
	var delays []time.Duration
	monitor := SQLiteDataVersionMonitor{
		ReadVersion: func(context.Context) (int64, error) {
			reads++
			return 0, errors.New("database is locked")
		},
		Interval:    100 * time.Millisecond,
		MaxInterval: 400 * time.Millisecond,
		OnChange:    func() {},
		sleep: func(ctx context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			if len(delays) == 4 {
				return context.Canceled
			}
			return nil
		},
	}
	if err := monitor.Run(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("monitor error = %v", err)
	}
	if reads != 4 || len(delays) != 4 || delays[0] != 100*time.Millisecond || delays[1] != 200*time.Millisecond || delays[2] != 400*time.Millisecond || delays[3] != 400*time.Millisecond {
		t.Fatalf("reads=%d delays=%v", reads, delays)
	}
}

func TestSQLiteDataVersionMonitorRejectsUnboundedConfig(t *testing.T) {
	monitor := SQLiteDataVersionMonitor{ReadVersion: func(context.Context) (int64, error) { return 0, nil }, OnChange: func() {}, Interval: time.Millisecond, MaxInterval: time.Second}
	if err := monitor.Run(context.Background()); !errors.Is(err, ErrInvalidSQLiteMonitor) {
		t.Fatalf("invalid monitor error = %v", err)
	}
}
