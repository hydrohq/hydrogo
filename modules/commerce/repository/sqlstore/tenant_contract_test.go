package sqlstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/testpostgres"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

// The tenant-scoped store contract runs on every dialect a developer can
// reach. Placeholder numbering bugs only show up on Postgres ($n) and
// argument-count bugs only on ?-dialects, so both must run.
func TestSQLiteTenantScopedStoreContract(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "tenant.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runTenantScopedStoreContract(t, db, database.SQLite)
}

func TestPostgresTenantScopedStoreContract(t *testing.T) {
	db := testpostgres.Open(t)
	if err := database.ApplyMigrations(context.Background(), db, database.Postgres, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runTenantScopedStoreContract(t, db, database.Postgres)
}

func tenantCtx(t *testing.T, id string) context.Context {
	t.Helper()
	scope, err := tenant.NewScope(id, id)
	if err != nil {
		t.Fatal(err)
	}
	return tenant.WithScope(context.Background(), scope)
}

func runTenantScopedStoreContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	store := Store{Dialect: dialect, RequireTenantScope: true}
	a := tenantCtx(t, "tenant-a")
	b := tenantCtx(t, "tenant-b")

	t.Run("stock lifecycle stays inside its tenant", func(t *testing.T) {
		if err := store.EnsureStock(a, db, "variant-shared", 5); err != nil {
			t.Fatal(err)
		}
		if err := store.EnsureStock(b, db, "variant-shared", 9); err != nil {
			t.Fatal(err)
		}
		if err := store.EnsureStock(a, db, "variant-shared", 4); err != nil {
			t.Fatalf("restocking an existing row: %v", err)
		}
		reserved, err := store.ReserveStock(a, db, "variant-shared", 3)
		if err != nil {
			t.Fatal(err)
		}
		if !reserved {
			t.Fatal("reserving 3 of 4 available units must succeed under tenant scope")
		}
		if err := store.ReleaseReservation(a, db, "variant-shared", 1); err != nil {
			t.Fatal(err)
		}
		if err := store.CommitReservation(a, db, "variant-shared", 2); err != nil {
			t.Fatal(err)
		}
		stockA, err := store.GetStock(a, db, "variant-shared")
		if err != nil {
			t.Fatal(err)
		}
		if stockA.Available != 2 || stockA.Reserved != 0 {
			t.Fatalf("tenant A stock = %+v, want available=2 reserved=0", stockA)
		}
		stockB, err := store.GetStock(b, db, "variant-shared")
		if err != nil {
			t.Fatal(err)
		}
		if stockB.Available != 9 || stockB.Reserved != 0 {
			t.Fatalf("tenant B stock must be untouched by tenant A, got %+v", stockB)
		}
	})

	t.Run("reconcile queries run under tenant scope", func(t *testing.T) {
		now := time.Now().UTC()
		order := commerce.Order{ID: "order-tenant-notified", GuestEmail: "g@example.com", IdempotencyKey: "idem-tenant-notified", IdempotencyHash: "h", Currency: "TRY", PaymentStatus: commerce.PaymentPending, FulfillmentStatus: commerce.FulfillmentUnfulfilled, CreatedAt: now, UpdatedAt: now}
		if _, _, err := store.CreateOrder(a, db, order, nil); err != nil {
			t.Fatal(err)
		}
		if _, err := store.RecordPaymentNotification(a, db, "paytr", order.ID); err != nil {
			t.Fatal(err)
		}
		notified, err := store.ListNotifiedOrdersNotPaid(a, db, "paytr", 10)
		if err != nil {
			t.Fatalf("ListNotifiedOrdersNotPaid under tenant scope: %v", err)
		}
		if len(notified) != 1 || notified[0].ID != order.ID {
			t.Fatalf("expected the notified-but-pending order, got %+v", notified)
		}
		other, err := store.ListNotifiedOrdersNotPaid(b, db, "paytr", 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(other) != 0 {
			t.Fatalf("tenant B must not see tenant A's notified orders, got %+v", other)
		}
		if _, err := store.StockReservationDrift(a, db, 10); err != nil {
			t.Fatalf("StockReservationDrift under tenant scope: %v", err)
		}
	})

	t.Run("order listing stays inside its tenant", func(t *testing.T) {
		orders, err := store.ListOrders(b, db, "", "", 10, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, o := range orders {
			if o.ID == "order-tenant-notified" {
				t.Fatal("tenant B listed tenant A's order")
			}
		}
	})
}
