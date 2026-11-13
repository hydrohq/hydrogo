package commerce_test

import (
	"context"
	"testing"
	"time"

	"github.com/hydrohq/hydrogo/modules/commerce"
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

func TestReconcileChecksEveryNamedProvider(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	if err := svc.Repository.EnsureStock(ctx, db, "variant-multi", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-multi", 1, 5000)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-reconcile-multi"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Repository.RecordPaymentNotification(ctx, db, "iyzico", order.ID); err != nil {
		t.Fatal(err)
	}
	report, err := svc.Reconcile(ctx, db, time.Now().UTC(), commerce.ReconcileConfig{}, "paytr, iyzico,havale", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.NotifiedButNotPaid) != 1 || report.NotifiedButNotPaid[0].ID != order.ID {
		t.Fatalf("an iyzico notification must be found when several providers are named, got %+v", report.NotifiedButNotPaid)
	}
}

func TestReconcileDoesNotFlagRefundedOrResolvedOrdersAndHonorsHavaleHold(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	for _, v := range []string{"variant-rec-1", "variant-rec-2", "variant-rec-havale"} {
		if err := svc.Repository.EnsureStock(ctx, db, v, 5); err != nil {
			t.Fatal(err)
		}
	}

	// 1. Paid and then refunded order must NOT appear in NotifiedButNotPaid.
	c1 := seedCartWithItem(t, svc, db, "variant-rec-1", 1, 5000)
	refundedOrder, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: c1.ID, GuestEmail: "r1@example.com", IdempotencyKey: "idem-rec-refunded", PaymentMethod: "paytr",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmPayment(ctx, commerce.Payment{Provider: "paytr", EventID: refundedOrder.ID, OrderID: refundedOrder.ID, AmountCents: refundedOrder.TotalCents}); err != nil {
		t.Fatal(err)
	}
	if ok, err := svc.Repository.UpdatePaymentStatus(ctx, db, refundedOrder.ID, commerce.PaymentPaid, commerce.PaymentRefunded); err != nil || !ok {
		t.Fatalf("mark refunded: ok=%v err=%v", ok, err)
	}

	// 2. Expired order with late payment: flagged while unresolved, clean after resolution.
	c2 := seedCartWithItem(t, svc, db, "variant-rec-2", 1, 5000)
	lateOrder, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: c2.ID, GuestEmail: "r2@example.com", IdempotencyKey: "idem-rec-late", PaymentMethod: "iyzico",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := svc.ExpireOrder(ctx, lateOrder.ID); err != nil || !ok {
		t.Fatalf("expire lateOrder: ok=%v err=%v", ok, err)
	}
	if _, err := svc.ConfirmPayment(ctx, commerce.Payment{Provider: "iyzico", EventID: lateOrder.ID, OrderID: lateOrder.ID, AmountCents: lateOrder.TotalCents}); err == nil {
		t.Fatal("expected ErrLatePayment")
	}

	// 3. Pending havale order is not stale at +2h.
	c3 := seedCartWithItem(t, svc, db, "variant-rec-havale", 1, 5000)
	if _, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: c3.ID, GuestEmail: "r3@example.com", IdempotencyKey: "idem-rec-havale", PaymentMethod: "havale",
	}); err != nil {
		t.Fatal(err)
	}

	future := time.Now().UTC().Add(2 * time.Hour)
	report, err := svc.Reconcile(ctx, db, future, commerce.ReconcileConfig{}, "paytr,iyzico", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.StalePendingOrders) != 0 {
		t.Fatalf("expected 0 stale orders (havale has 48h hold), got %+v", report.StalePendingOrders)
	}
	if len(report.NotifiedButNotPaid) != 1 || report.NotifiedButNotPaid[0].ID != lateOrder.ID {
		t.Fatalf("expected only unresolved lateOrder in NotifiedButNotPaid, got %+v", report.NotifiedButNotPaid)
	}

	// Resolve the refund_required record -> reconcile must now be completely clean.
	items, err := svc.ListRefundsRequired(ctx, true, 10, 0)
	if err != nil || len(items) != 1 {
		t.Fatalf("expected 1 unresolved refund_required, got %d err=%v", len(items), err)
	}
	if ok, err := svc.ResolveRefundRequired(ctx, items[0].ID, "refunded in panel"); err != nil || !ok {
		t.Fatalf("resolve refund_required: ok=%v err=%v", ok, err)
	}
	cleanReport, err := svc.Reconcile(ctx, db, future, commerce.ReconcileConfig{}, "paytr,iyzico", 100)
	if err != nil {
		t.Fatal(err)
	}
	if !cleanReport.Clean() {
		t.Fatalf("expected clean reconcile report after refund resolution, got %+v", cleanReport)
	}
}
