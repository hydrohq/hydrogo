package commerce_test

import (
	"context"
	"testing"
	"time"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/modules/commerce"
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
	expired, err := worker.ExpireDue(ctx, time.Now().UTC())
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

func TestWorkerExpireDueHoldsBankTransferAndExtendedHoldOrders(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	store := svc.Repository

	for _, v := range []string{"variant-w-card", "variant-w-havale", "variant-w-fraud"} {
		if err := store.EnsureStock(ctx, db, v, 5); err != nil {
			t.Fatal(err)
		}
	}

	now := time.Now().UTC()

	// Card order 45 minutes old: should expire with 30m Hold.
	cardCart := seedCartWithItem(t, svc, db, "variant-w-card", 1, 100)
	cardOrder, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: cardCart.ID, GuestEmail: "card@example.com", IdempotencyKey: "worker-card-1", PaymentMethod: "paytr",
	})
	if err != nil {
		t.Fatal(err)
	}
	backdateOrder(t, db, cardOrder.ID, now.Add(-45*time.Minute))

	// Havale order 2 hours old: must NOT expire until 48h BankTransferHold passes.
	havaleCart := seedCartWithItem(t, svc, db, "variant-w-havale", 1, 100)
	havaleOrder, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: havaleCart.ID, GuestEmail: "havale@example.com", IdempotencyKey: "worker-havale-1", PaymentMethod: "havale",
	})
	if err != nil {
		t.Fatal(err)
	}
	backdateOrder(t, db, havaleOrder.ID, now.Add(-2*time.Hour))

	// Card order under 24h fraud review hold: must NOT expire at +2h.
	fraudCart := seedCartWithItem(t, svc, db, "variant-w-fraud", 1, 100)
	fraudOrder, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: fraudCart.ID, GuestEmail: "fraud@example.com", IdempotencyKey: "worker-fraud-1", PaymentMethod: "iyzico",
	})
	if err != nil {
		t.Fatal(err)
	}
	backdateOrder(t, db, fraudOrder.ID, now.Add(-2*time.Hour))
	if err := svc.ExtendOrderHold(ctx, fraudOrder.ID, now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	worker := commerce.Worker{Service: svc, Hold: 30 * time.Minute, BankTransferHold: 48 * time.Hour}
	expired, err := worker.ExpireDue(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if expired != 1 {
		t.Fatalf("expected 1 expired order (card only), got %d", expired)
	}
	if got, _ := store.GetOrder(ctx, db, cardOrder.ID); got.PaymentStatus != commerce.PaymentFailed {
		t.Fatalf("card order = %q, want failed", got.PaymentStatus)
	}
	if got, _ := store.GetOrder(ctx, db, havaleOrder.ID); got.PaymentStatus != commerce.PaymentPending {
		t.Fatalf("havale order = %q, want pending", got.PaymentStatus)
	}
	if got, _ := store.GetOrder(ctx, db, fraudOrder.ID); got.PaymentStatus != commerce.PaymentPending {
		t.Fatalf("fraud-review order = %q, want pending", got.PaymentStatus)
	}

	// Fast-forward past 48h and 24h hold_until: both remaining orders now expire.
	expired, err = worker.ExpireDue(ctx, now.Add(49*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if expired != 2 {
		t.Fatalf("expected remaining 2 orders to expire at +49h, got %d", expired)
	}
}

func backdateOrder(t *testing.T, db database.Executor, orderID string, when time.Time) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), "UPDATE commerce_customer_orders SET created_at = ? WHERE id = ?", when.Format(time.RFC3339Nano), orderID); err != nil {
		t.Fatal(err)
	}
}
