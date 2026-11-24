package contract_test

import (
	"context"
	"testing"
	"time"

	adminstudiosql "github.com/hydrohq/hydrogo/internal/adminstudio/sqlstore"
	analyticssql "github.com/hydrohq/hydrogo/internal/analytics/sqlstore"
	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/internal/tenant"
	"github.com/hydrohq/hydrogo/migrations"
)

// The analytics and admin studio stores must keep working under the
// fail-closed policies: single-tenant rows ("default" analytics, unscoped
// notifications) stay readable and writable, and named tenants stay apart.
func TestPostgresRLSStoresBindTheirTenant(t *testing.T) {
	ctx := context.Background()
	db := openIsolatedPostgres(t, ctx)
	if err := database.ApplyMigrations(ctx, db, database.Postgres, migrations.FS); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	day := time.Now().UTC()

	analytics := analyticssql.Store{Executor: db, Dialect: database.Postgres}
	if err := analytics.RecordFeedback(ctx, "default", "news", "n1", "up", day); err != nil {
		t.Fatalf("single-tenant feedback write: %v", err)
	}
	if err := analytics.RecordFeedback(ctx, "tenant-alpha", "news", "n1", "down", day); err != nil {
		t.Fatalf("tenant feedback write: %v", err)
	}
	totals, err := analytics.FeedbackTotals(ctx, "default", "news", "n1")
	if err != nil || totals.Up != 1 || totals.Down != 0 {
		t.Fatalf("default feedback totals = %+v, err %v; want up=1 down=0", totals, err)
	}
	if count, err := analytics.IncrementContentView(ctx, "", "news", "n1", "tr"); err != nil || count != 1 {
		t.Fatalf("single-tenant content view = %d, err %v; want 1", count, err)
	}
	if count, err := analytics.ContentViewTotal(ctx, "tenant-alpha", "news", "n1"); err != nil || count != 0 {
		t.Fatalf("tenant-alpha sees %d default views (err %v); want 0", count, err)
	}

	notes := adminstudiosql.Store{Dialect: database.Postgres}
	if err := notes.Add(ctx, db, "user-1", "Single", "single-tenant note", "/admin"); err != nil {
		t.Fatalf("single-tenant notification add: %v", err)
	}
	alpha, err := tenant.NewScope("tenant-alpha", "org-alpha")
	if err != nil {
		t.Fatal(err)
	}
	alphaCtx := tenant.WithScope(ctx, alpha)
	if err := notes.Add(alphaCtx, db, "user-1", "Alpha", "alpha note", "/admin"); err != nil {
		t.Fatalf("tenant notification add: %v", err)
	}
	single, err := notes.List(ctx, db, "user-1", 10)
	if err != nil || len(single) != 1 || single[0].Title != "Single" {
		t.Fatalf("single-tenant list = %+v, err %v; want only Single", single, err)
	}
	scoped, err := notes.List(alphaCtx, db, "user-1", 10)
	if err != nil || len(scoped) != 1 || scoped[0].Title != "Alpha" {
		t.Fatalf("tenant-alpha list = %+v, err %v; want only Alpha", scoped, err)
	}
	if n, err := notes.UnreadCount(alphaCtx, db, "user-1"); err != nil || n != 1 {
		t.Fatalf("tenant-alpha unread = %d, err %v; want 1", n, err)
	}
}
