package database

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/capacity"
)

func TestCapacityProfilesMapToBoundedDatabasePools(t *testing.T) {
	for _, profile := range []capacity.Profile{capacity.Compact, capacity.Standard, capacity.Distributed} {
		profile := profile
		t.Run(string(profile), func(t *testing.T) {
			budget, err := capacity.Defaults(profile)
			if err != nil {
				t.Fatal(err)
			}
			db, err := OpenWithOptions(context.Background(), SQLite, filepath.Join(t.TempDir(), "profile.sqlite"), OpenOptions{
				MaxOpenConns: budget.DBMaxOpenConns,
				MaxIdleConns: budget.DBMaxIdleConns,
			})
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			stats := db.Stats()
			if stats.MaxOpenConnections != budget.DBMaxOpenConns {
				t.Fatalf("max open connections = %d, want %d", stats.MaxOpenConnections, budget.DBMaxOpenConns)
			}
			if budget.DBMaxIdleConns > stats.MaxOpenConnections {
				t.Fatalf("idle connections %d exceed open connections %d", budget.DBMaxIdleConns, stats.MaxOpenConnections)
			}
		})
	}
}

func TestTransactorCommitsAndRollsBackAtomically(t *testing.T) {
	db, err := Open(context.Background(), SQLite, filepath.Join(t.TempDir(), "transaction.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), "CREATE TABLE entries (id INTEGER PRIMARY KEY, value TEXT NOT NULL)"); err != nil {
		t.Fatal(err)
	}

	rollbackErr := errors.New("abort transaction")
	if err := NewTransactor(db).Within(context.Background(), func(executor Executor) error {
		if _, err := executor.ExecContext(context.Background(), "INSERT INTO entries (value) VALUES (?)", "rolled-back"); err != nil {
			return err
		}
		return rollbackErr
	}); !errors.Is(err, rollbackErr) {
		t.Fatalf("rollback error = %v, want %v", err, rollbackErr)
	}
	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM entries").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rows after rollback = %d, want 0", count)
	}

	if err := NewTransactor(db).Within(context.Background(), func(executor Executor) error {
		_, err := executor.ExecContext(context.Background(), "INSERT INTO entries (value) VALUES (?)", "committed")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM entries").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("rows after commit = %d, want 1", count)
	}
}
