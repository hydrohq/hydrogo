package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"strings"
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
	var actor, resource, created string
	if err := db.QueryRowContext(ctx, "SELECT actor_id, resource_type, created_at FROM audit_events WHERE id = ?", "audit-1").Scan(&actor, &resource, &created); err != nil {
		t.Fatal(err)
	}
	if actor != "user-1" || resource != "auth" || !strings.Contains(created, "T") {
		t.Fatalf("audit metadata actor=%q resource=%q created=%q", actor, resource, created)
	}
}

func TestRecordAutomaticallyScopesTenantContext(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, "file:audit-tenant-scope?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE audit_events (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, action TEXT NOT NULL, target_id TEXT NOT NULL, request_id TEXT, actor_id TEXT NOT NULL DEFAULT '', resource_type TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL DEFAULT '', before_json TEXT NOT NULL DEFAULT '{}', after_json TEXT NOT NULL DEFAULT '{}')`); err != nil {
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

func TestRecordChangeBoundsAndPersistsJSONDiff(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, "file:audit-change?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := Store{Dialect: database.SQLite}
	before, after := []byte(`{"version":1}`), []byte(`{"version":2}`)
	if err := database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
		return store.RecordChange(ctx, tx, "audit-diff", "pages.updated", "page-1", before, after)
	}); err != nil {
		t.Fatal(err)
	}
	var gotBefore, gotAfter string
	if err := db.QueryRowContext(ctx, "SELECT before_json, after_json FROM audit_events WHERE id = ?", "audit-diff").Scan(&gotBefore, &gotAfter); err != nil {
		t.Fatal(err)
	}
	if gotBefore != string(before) || gotAfter != string(after) {
		t.Fatalf("stored diff = %s -> %s", gotBefore, gotAfter)
	}
	err = database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
		return store.RecordChange(ctx, tx, "audit-bad", "pages.updated", "page-1", []byte("not-json"), after)
	})
	if err == nil {
		t.Fatal("accepted invalid JSON diff")
	}
}

func TestAuditExplorerFiltersAndPaginatesWithinTenant(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, "file:audit-explorer?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := Store{Dialect: database.SQLite}
	for _, item := range []struct{ id, action, target string }{{"event-a", "pages.updated", "page-1"}, {"event-b", "news.published", "news-2"}} {
		if err := database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
			return store.Record(ctx, tx, item.id, item.action, item.target)
		}); err != nil {
			t.Fatal(err)
		}
	}
	explorer := Store{DB: db, Dialect: database.SQLite}
	result, err := explorer.Explore(ctx, ExplorerFilter{Resource: "pages", Action: "updated", Target: "page-1", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(result.Events) != 1 || result.Events[0].ID != "event-a" || result.Events[0].ResourceType != "pages" {
		t.Fatalf("explorer result = %+v", result)
	}
	if _, err := (Store{DB: db, Dialect: database.SQLite, RequireTenantScope: true}).Explore(ctx, ExplorerFilter{}); !errors.Is(err, tenant.ErrMissingScope) {
		t.Fatalf("missing tenant scope error = %v", err)
	}
}

func TestRecordRequiresTenantInFailClosedMode(t *testing.T) {
	store := Store{Dialect: database.SQLite, RequireTenantScope: true}
	if err := store.Record(context.Background(), nil, "id", "action", "target"); !errors.Is(err, tenant.ErrMissingScope) {
		t.Fatalf("missing tenant scope = %v", err)
	}
}
