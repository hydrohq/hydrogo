package persistence

import (
	"context"
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
	"os"
	"testing"
)

func openStore(t *testing.T) (*sql.DB, Store) {
	t.Helper()
	dialect := SQLite
	dsn := "file:construction-persistence?mode=memory&cache=shared"
	driver := "sqlite"
	if url := os.Getenv("HYDRO_CONSTRUCTION_POSTGRES_URL"); url != "" {
		dialect = PostgreSQL
		dsn = url
		driver = "pgx"
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db, Store{Dialect: dialect}
}

func TestConstructionPersistenceApprovalIsScopedAndAudited(t *testing.T) {
	db, store := openStore(t)
	ctx := context.Background()
	if err := store.ApplySchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO construction_projects(id,organization_id,name,created_at) VALUES ('p1','o1','Project','now')`); err != nil {
		t.Fatal(err)
	}
	if err := store.AddMembership(ctx, db, "p1", "manager-1", "manager"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddMembership(ctx, db, "p1", "viewer-1", "viewer"); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateDocument(ctx, db, "d1", "p1", "tenants/t1/quarantine/x", "sha", "author"); err != nil {
		t.Fatal(err)
	}
	if err := store.ApproveDocument(ctx, db, "p1", "d1", "viewer-1", 1); err != ErrDenied {
		t.Fatalf("viewer approval=%v", err)
	}
	if err := store.ApproveDocument(ctx, db, "p1", "d1", "manager-1", 1); err != nil {
		t.Fatal(err)
	}
	if err := store.ApproveDocument(ctx, db, "p1", "d1", "manager-1", 1); err != ErrConflict {
		t.Fatalf("stale approval=%v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM construction_audit_events WHERE action='document.approved'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("audit=%d/%v", count, err)
	}
}
