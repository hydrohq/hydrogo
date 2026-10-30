package audit_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/audit"
	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}

func TestVerifyAuditChainIntegrityAndTamperDetection(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	auditstore.SetChain(true)
	defer auditstore.SetChain(false)

	store := auditstore.Store{DB: db, Dialect: database.SQLite}
	svc := audit.Service{Store: store, DB: db, Dialect: database.SQLite}

	// 1. Record events
	if err := store.Record(ctx, db, "evt-1", "page.create", "page-1"); err != nil {
		t.Fatalf("record 1: %v", err)
	}
	if err := store.Record(ctx, db, "evt-2", "page.update", "page-1"); err != nil {
		t.Fatalf("record 2: %v", err)
	}
	if err := store.Record(ctx, db, "evt-3", "page.publish", "page-1"); err != nil {
		t.Fatalf("record 3: %v", err)
	}

	// 2. Verify untampered chain
	result, err := svc.VerifyAuditChain(ctx, "")
	if err != nil {
		t.Fatalf("verify untampered: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid audit chain, got problems: %v", result.Problems)
	}
	if result.TotalLinks != 3 {
		t.Fatalf("expected 3 links, got %d", result.TotalLinks)
	}

	// 3. Tamper directly with the database (as a DBA dropping immutability trigger)
	_, _ = db.ExecContext(ctx, "DROP TRIGGER IF EXISTS audit_events_immutable_update")
	_, err = db.ExecContext(ctx, "UPDATE audit_events SET action = 'page.tampered' WHERE id = 'evt-2'")
	if err != nil {
		t.Fatalf("tamper update: %v", err)
	}

	// 4. Verify tampering is detected
	tamperedResult, err := svc.VerifyAuditChain(ctx, "")
	if err != nil {
		t.Fatalf("verify after tamper: %v", err)
	}
	if tamperedResult.Valid {
		t.Fatal("expected audit chain verification to fail on tampered record, but it reported valid")
	}
	if len(tamperedResult.Problems) == 0 {
		t.Fatal("expected problems reported for tampered row, got 0")
	}
}
