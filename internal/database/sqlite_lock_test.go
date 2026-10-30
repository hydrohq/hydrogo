package database

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteMigrationConcurrencyLock(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "concurrent_migrate.sqlite")
	ctx := context.Background()

	// Spin up 5 concurrent workers attempting to apply migrations on the same SQLite database file.
	const workerCount = 5
	var wg sync.WaitGroup
	errs := make([]error, workerCount)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			db, err := Open(ctx, SQLite, dbPath)
			if err != nil {
				errs[idx] = err
				return
			}
			defer db.Close()

			errs[idx] = ApplyMigrations(ctx, db, SQLite, migrations.FS)
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("worker %d failed migration: %v", i, err)
		}
	}

	// Verify that migrations ledger has correct count
	db, err := Open(ctx, SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("query schema_migrations count: %v", err)
	}
	if count == 0 {
		t.Fatal("expected schema_migrations to contain records")
	}
}

func TestSQLiteMigrationLockTimeout(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lock_timeout.sqlite")
	ctx := context.Background()

	db, err := Open(ctx, SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Pre-create database by running migration once
	if err := ApplyMigrations(ctx, db, SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	lockPath := dbPath + ".lock"
	// Manually hold file lock
	fl, err := acquireFileLock(ctx, lockPath, 5*time.Second)
	if err != nil {
		t.Fatalf("acquireFileLock: %v", err)
	}
	defer fl.Release()

	// Running migrations with a short timeout context should cleanly timeout waiting for the lock
	shortCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = ApplyMigrations(shortCtx, db, SQLite, migrations.FS)
	if err == nil {
		t.Fatal("expected migration to fail when lock is held by another process")
	}
}
