package database

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

// Concurrent read-then-write transactions on a multi-connection SQLite pool
// must wait for the write lock (busy_timeout), not fail with SQLITE_BUSY.
func TestSQLiteConcurrentReadWriteTransactionsDoNotFailBusy(t *testing.T) {
	ctx := context.Background()
	db, err := OpenWithOptions(ctx, SQLite, "file:"+filepath.Join(t.TempDir(), "txlock.db"), OpenOptions{MaxOpenConns: 10, MaxIdleConns: 10, SQLiteProfile: SQLiteVPSProfile})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `CREATE TABLE stock(id TEXT PRIMARY KEY, available INTEGER NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO stock VALUES('v', 1000)`); err != nil {
		t.Fatal(err)
	}
	transactor := NewTenantTransactor(db, SQLite, false)
	const workers = 100
	var failed atomic.Int64
	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := transactor.Within(ctx, func(tx Executor) error {
				var available int
				if err := tx.QueryRowContext(ctx, `SELECT available FROM stock WHERE id = 'v'`).Scan(&available); err != nil {
					return err
				}
				_, err := tx.ExecContext(ctx, `UPDATE stock SET available = available - 1 WHERE id = 'v' AND available >= 1`)
				return err
			})
			if err != nil {
				failed.Add(1)
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if n := failed.Load(); n > 0 {
		t.Fatalf("%d of %d concurrent transactions failed; first: %v", n, workers, firstErr)
	}
	var left int
	if err := db.QueryRowContext(ctx, `SELECT available FROM stock WHERE id = 'v'`).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 1000-workers {
		t.Fatalf("available = %d, want %d", left, 1000-workers)
	}
}
