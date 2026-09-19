package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestRecordPersistsRequestIDFromContext(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, "file:audit-request-id?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	requestCtx := security.WithRequestID(ctx, "req-audit-123")
	store := Store{Dialect: database.SQLite}
	if err := database.NewTransactor(db).Within(requestCtx, func(tx database.Executor) error {
		return store.Record(requestCtx, tx, "audit-1", "auth.login", "user-1")
	}); err != nil {
		t.Fatal(err)
	}
	var got sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT request_id FROM audit_events WHERE id = ?", "audit-1").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if !got.Valid || got.String != "req-audit-123" {
		t.Fatalf("request_id = %#v", got)
	}
}

func TestRecordAutomaticallyScopesTenantContext(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, "file:audit-tenant-scope?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE audit_events (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, action TEXT NOT NULL, target_id TEXT NOT NULL, request_id TEXT)`); err != nil {
		t.Fatal(err)
	}
	scope, err := tenant.NewScope("tenant-a", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	requestCtx := security.WithRequestID(tenant.WithScope(ctx, scope), "req-tenant")
	store := Store{Dialect: database.SQLite, RequireTenantScope: true}
	if err := database.NewTransactor(db).Within(requestCtx, func(tx database.Executor) error {
		return store.Record(requestCtx, tx, "audit-tenant", "auth.login", "user-1")
	}); err != nil {
		t.Fatal(err)
	}
	var tenantID, requestID string
	if err := db.QueryRowContext(ctx, "SELECT tenant_id, request_id FROM audit_events WHERE id = ?", "audit-tenant").Scan(&tenantID, &requestID); err != nil {
		t.Fatal(err)
	}
	if tenantID != "tenant-a" || requestID != "req-tenant" {
		t.Fatalf("tenant=%q request=%q", tenantID, requestID)
	}
}

func TestRecordRequiresTenantInFailClosedMode(t *testing.T) {
	store := Store{Dialect: database.SQLite, RequireTenantScope: true}
	if err := store.Record(context.Background(), nil, "id", "action", "target"); !errors.Is(err, tenant.ErrMissingScope) {
		t.Fatalf("missing tenant scope = %v", err)
	}
}
