package commerce_test

import (
	"context"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

func TestReconcileCleanWhenNothingWrong(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	if err := svc.Repository.EnsureStock(ctx, db, "variant-clean", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-clean", 1, 5000)
	if _, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-reconcile-clean",
	}); err != nil {
		t.Fatal(err)
	}

	report, err := svc.Reconcile(ctx, db, time.Now().UTC(), commerce.ReconcileConfig{}, "mock-provider", 100)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Clean() {
		t.Fatalf("expected a clean report for a freshly checked-out order, got %+v", report)
	}
}

func TestReconcileFlagsStalePendingOrder(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	if err := svc.Repository.EnsureStock(ctx, db, "variant-stale", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-stale", 1, 5000)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-reconcile-stale",
	})
	if err != nil {
		t.Fatal(err)
	}

	// The order was just created, so it is not stale "now". Simulate the
	// worker's expiry sweep never having run by asking Reconcile about a
	// point far enough in the future that Hold+StaleAfter has elapsed.
	future := time.Now().UTC().Add(2 * time.Hour)
	report, err := svc.Reconcile(ctx, db, future, commerce.ReconcileConfig{}, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.StalePendingOrders) != 1 || report.StalePendingOrders[0].ID != order.ID {
		t.Fatalf("expected the pending order to be flagged as stale, got %+v", report.StalePendingOrders)
	}
}

func TestReconcileFlagsNotifiedButNotPaid(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	if err := svc.Repository.EnsureStock(ctx, db, "variant-notified", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-notified", 1, 5000)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-reconcile-notified",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Record the provider's notification directly, without going through
	// ConfirmPayment -- simulating the exact gap this check exists to
	// catch: the provider's callback landed (so a naive "did the provider
	// ever call back" check would look fine), but the order's own
	// payment_status transition never actually applied.
	if _, err := svc.Repository.RecordPaymentNotification(ctx, db, "mock-provider", order.ID); err != nil {
		t.Fatal(err)
	}

	report, err := svc.Reconcile(ctx, db, time.Now().UTC(), commerce.ReconcileConfig{}, "mock-provider", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.NotifiedButNotPaid) != 1 || report.NotifiedButNotPaid[0].ID != order.ID {
		t.Fatalf("expected the order to be flagged as notified-but-not-paid, got %+v", report.NotifiedButNotPaid)
	}

	// With no provider configured, this check is skipped entirely rather
	// than reporting every unconfirmed order as an anomaly.
	emptyProviderReport, err := svc.Reconcile(ctx, db, time.Now().UTC(), commerce.ReconcileConfig{}, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(emptyProviderReport.NotifiedButNotPaid) != 0 {
		t.Fatalf("expected no notified-but-not-paid check with no provider given, got %+v", emptyProviderReport.NotifiedButNotPaid)
	}
}

func TestReconcileFlagsReservationDrift(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	if err := svc.Repository.EnsureStock(ctx, db, "variant-drift", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-drift", 2, 5000)
	if _, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-reconcile-drift",
	}); err != nil {
		t.Fatal(err)
	}

	// Release the reservation directly, bypassing ExpireOrder/ConfirmPayment
	// -- simulating a release that ran without the order's own status
	// transition applying (a bug elsewhere, or manual DB surgery), which is
	// exactly the drift this check exists to surface: the order is still
	// "pending" with 2 units' worth of items, but stock now shows 0 reserved.
	if err := svc.Repository.ReleaseReservation(ctx, db, "variant-drift", 2); err != nil {
		t.Fatal(err)
	}

	report, err := svc.Reconcile(ctx, db, time.Now().UTC(), commerce.ReconcileConfig{}, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.ReservationDrift) != 1 {
		t.Fatalf("expected exactly one drifted variant, got %+v", report.ReservationDrift)
	}
	drift := report.ReservationDrift[0]
	if drift.VariantID != "variant-drift" || drift.Reserved != 0 || drift.Expected != 2 {
		t.Fatalf("unexpected drift row: %+v", drift)
	}
}
