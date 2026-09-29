package database

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMaintainPostgres(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	db, err := Open(context.Background(), Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	report, err := Maintain(context.Background(), db, Postgres, "")
	if err != nil {
		t.Fatal(err)
	}
	if report.Duration <= 0 || report.ReclaimedBytes != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestMaintainSQLitePreservesReadersAndReclaimsSpace(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "maintenance.sqlite")
	db, err := OpenWithOptions(ctx, SQLite, path, OpenOptions{MaxOpenConns: 2, MaxIdleConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE maintenance_test (value TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	payload := strings.Repeat("x", 4096)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for i := 0; i < 512; i++ {
		if _, err := tx.ExecContext(ctx, "INSERT INTO maintenance_test(value) VALUES (?)", payload); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM maintenance_test`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	reader, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Rollback()
	var count int
	if err := reader.QueryRowContext(ctx, `SELECT COUNT(*) FROM maintenance_test`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("reader count=%d err=%v", count, err)
	}
	report, err := Maintain(ctx, db, SQLite, path)
	if err != nil {
		t.Fatal(err)
	}
	if report.Duration <= 0 {
		t.Fatalf("maintenance duration=%s", report.Duration)
	}
	if err := reader.QueryRowContext(ctx, `SELECT COUNT(*) FROM maintenance_test`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("active reader was interrupted: count=%d err=%v", count, err)
	}
	if err := reader.Commit(); err != nil {
		t.Fatal(err)
	}
	report, err = Maintain(ctx, db, SQLite, path)
	if err != nil {
		t.Fatal(err)
	}
	if report.ReclaimedBytes <= 0 {
		t.Fatalf("vacuum reclaimed %d bytes", report.ReclaimedBytes)
	}
}

func TestNextWeeklyMaintenance(t *testing.T) {
	for _, tc := range []struct{ now, want string }{
		{"2026-09-27T02:59:59Z", "2026-09-27T03:00:00Z"},
		{"2026-09-27T03:00:00Z", "2026-10-04T03:00:00Z"},
		{"2026-09-28T12:00:00-04:00", "2026-10-04T03:00:00Z"},
	} {
		now, _ := time.Parse(time.RFC3339, tc.now)
		if got := NextWeeklyMaintenance(now).Format(time.RFC3339); got != tc.want {
			t.Errorf("NextWeeklyMaintenance(%s)=%s want %s", tc.now, got, tc.want)
		}
	}
}
