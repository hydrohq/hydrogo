package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func openGatedForTest(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	db, err := OpenWithOptions(ctx, SQLite, "file:"+filepath.Join(t.TempDir(), "gate.db"), OpenOptions{MaxOpenConns: 10, MaxIdleConns: 10, SQLiteProfile: SQLiteVPSProfile})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(ctx, `CREATE TABLE g(id INTEGER PRIMARY KEY, v INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO g VALUES (1, 0)`); err != nil {
		t.Fatal(err)
	}
	return db
}

// While one write transaction is open, reads and read-only transactions on
// other connections proceed; a second writer waits and honours its context.
func TestSQLiteWriteGateQueuesWritersOnly(t *testing.T) {
	db := openGatedForTest(t)
	ctx := context.Background()
	writer, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.ExecContext(ctx, `UPDATE g SET v = v + 1 WHERE id = 1`); err != nil {
		t.Fatal(err)
	}

	readCtx, cancelRead := context.WithTimeout(ctx, time.Second)
	defer cancelRead()
	var v int
	if err := db.QueryRowContext(readCtx, `SELECT v FROM g WHERE id = 1`).Scan(&v); err != nil || v != 0 {
		t.Fatalf("read during open write = %d, %v; want 0 (snapshot), no wait", v, err)
	}
	ro, err := db.BeginTx(readCtx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatalf("read-only tx blocked by writer: %v", err)
	}
	_ = ro.Rollback()

	waitCtx, cancelWait := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancelWait()
	if _, err := db.BeginTx(waitCtx, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second writer err = %v, want deadline exceeded while the gate is held", err)
	}
	if _, err := db.ExecContext(waitCtx, `UPDATE g SET v = 99`); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("autocommit write err = %v, want deadline exceeded while the gate is held", err)
	}

	if err := writer.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE g SET v = v + 1 WHERE id = 1`); err != nil {
		t.Fatalf("write after commit: %v", err)
	}
}

// A transaction whose context is cancelled is rolled back by database/sql and
// must hand the gate back.
func TestSQLiteWriteGateReleasedOnCancelledTransaction(t *testing.T) {
	db := openGatedForTest(t)
	ctx := context.Background()
	txCtx, cancel := context.WithCancel(ctx)
	tx, err := db.BeginTx(txCtx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(txCtx, `UPDATE g SET v = 5`); err != nil {
		t.Fatal(err)
	}
	cancel()
	nextCtx, cancelNext := context.WithTimeout(ctx, 2*time.Second)
	defer cancelNext()
	if _, err := db.ExecContext(nextCtx, `UPDATE g SET v = v + 1 WHERE id = 1`); err != nil {
		t.Fatalf("gate leaked after cancelled transaction: %v", err)
	}
	var v int
	if err := db.QueryRowContext(ctx, `SELECT v FROM g WHERE id = 1`).Scan(&v); err != nil || v != 1 {
		t.Fatalf("v = %d, %v; want 1 (cancelled update rolled back)", v, err)
	}
}
