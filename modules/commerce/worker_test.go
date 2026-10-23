package commerce_test

import (
	"context"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

func TestWorkerExpireDueReleasesStaleReservationsOnly(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	store := svc.Repository

	if err := store.EnsureStock(ctx, db, "variant-worker-stale", 5); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureStock(ctx, db, "variant-worker-fresh", 5); err != nil {
		t.Fatal(err)
	}

	staleCart := seedCartWithItem(t, svc, db, "variant-worker-stale", 1, 100)
	staleOrder, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: staleCart.ID, GuestEmail: "stale@example.com", IdempotencyKey: "worker-stale-1"})
	if err != nil {
		t.Fatal(err)
	}
	// Back-date the order so it looks like it has been pending well past
	// the hold window, without needing to actually sleep in a test.
	backdateOrder(t, db, staleOrder.ID, time.Now().UTC().Add(-2*time.Hour))

	freshCart := seedCartWithItem(t, svc, db, "variant-worker-fresh", 1, 100)
	if _, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: freshCart.ID, GuestEmail: "fresh@example.com", IdempotencyKey: "worker-fresh-1"}); err != nil {
		t.Fatal(err)
	}

	worker := commerce.Worker{Service: svc, Hold: 30 * time.Minute}
	expired, err := worker.ExpireDue(ctx, db, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if expired != 1 {
		t.Fatalf("expected exactly 1 order expired, got %d", expired)
	}

	staleStock, err := store.GetStock(ctx, db, "variant-worker-stale")
	if err != nil {
		t.Fatal(err)
	}
	if staleStock.Available != 5 || staleStock.Reserved != 0 {
		t.Fatalf("stale order's stock should be released, got %+v", staleStock)
	}
	freshStock, err := store.GetStock(ctx, db, "variant-worker-fresh")
	if err != nil {
		t.Fatal(err)
	}
	if freshStock.Available != 4 || freshStock.Reserved != 1 {
		t.Fatalf("fresh order's reservation must survive the sweep, got %+v", freshStock)
	}
}

func backdateOrder(t *testing.T, db database.Executor, orderID string, when time.Time) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), "UPDATE commerce_customer_orders SET created_at = ? WHERE id = ?", when.Format(time.RFC3339Nano), orderID); err != nil {
		t.Fatal(err)
	}
}
