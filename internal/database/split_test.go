package database

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestSplitPoolConcurrentReadsAndWrites(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "split_concurrency.sqlite")

	// Initialize table with single connection first
	initDB, err := Open(ctx, SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := initDB.ExecContext(ctx, `CREATE TABLE items (id INTEGER PRIMARY KEY, val TEXT)`); err != nil {
		t.Fatal(err)
	}
	_ = initDB.Close()

	// Open with SplitPool
	pool, err := OpenSplit(ctx, SQLite, dbPath, OpenOptions{MaxOpenConns: 10})
	if err != nil {
		t.Fatalf("OpenSplit: %v", err)
	}
	defer pool.Close()

	transactor := NewTransactor(pool)

	const (
		numReaders = 50
		numWriters = 5
		iterations = 20
	)

	var wg sync.WaitGroup
	errs := make(chan error, numReaders*iterations+numWriters*iterations)

	// Launch 50 concurrent readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				var count int
				err := pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM items").Scan(&count)
				if err != nil {
					errs <- fmt.Errorf("reader %d failed at iter %d: %w", readerID, j, err)
					return
				}
				time.Sleep(500 * time.Microsecond)
			}
		}(i)
	}

	// Launch 5 concurrent writers using transactions
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				id := writerID*1000 + j
				err := transactor.Within(ctx, func(tx Executor) error {
					_, err := tx.ExecContext(ctx, "INSERT INTO items (id, val) VALUES (?, ?)", id, fmt.Sprintf("val_%d_%d", writerID, j))
					return err
				})
				if err != nil {
					errs <- fmt.Errorf("writer %d failed at iter %d: %w", writerID, j, err)
					return
				}
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	var collectedErrs []error
	for err := range errs {
		collectedErrs = append(collectedErrs, err)
	}
	if len(collectedErrs) > 0 {
		t.Fatalf("encountered %d concurrency errors, first: %v", len(collectedErrs), collectedErrs[0])
	}

	// Verify all records written
	var finalCount int
	if err := pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM items").Scan(&finalCount); err != nil {
		t.Fatalf("count: %v", err)
	}
	if finalCount != numWriters*iterations {
		t.Fatalf("final count want %d, got %d", numWriters*iterations, finalCount)
	}
}

func TestSplitPoolRoutingAndReadOnlyEnforcement(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "split_routing.sqlite")

	initDB, err := Open(ctx, SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := initDB.ExecContext(ctx, `CREATE TABLE logs (id INTEGER PRIMARY KEY, msg TEXT)`); err != nil {
		t.Fatal(err)
	}
	_ = initDB.Close()

	pool, err := OpenSplit(ctx, SQLite, dbPath, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	// 1. Direct write works via ExecContext
	if _, err := pool.ExecContext(ctx, "INSERT INTO logs (id, msg) VALUES (1, 'ok')"); err != nil {
		t.Fatalf("pool.ExecContext write: %v", err)
	}

	// 2. QueryContext on SELECT works
	rows, err := pool.QueryContext(ctx, "SELECT id, msg FROM logs")
	if err != nil {
		t.Fatalf("pool.QueryContext SELECT: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("expected 1 row")
	}

	// 3. Reader pool directly rejects writes because it has query_only pragma
	if _, err := pool.Reader().ExecContext(ctx, "INSERT INTO logs (id, msg) VALUES (2, 'fail')"); err == nil {
		t.Fatalf("expected reader pool to reject write, but it succeeded")
	}
}
