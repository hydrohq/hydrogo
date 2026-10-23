package commerce_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
	"github.com/ardaozyildirim/hydrogo/modules/commerce/repository/sqlstore"
)

func newTestService(t *testing.T) (commerce.Service, database.Executor) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "commerce-service.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sqlstore.Store{Dialect: database.SQLite}
	svc := commerce.Service{Transactions: database.NewTransactor(db), Repository: store}
	return svc, db
}

func seedCartWithItem(t *testing.T, svc commerce.Service, db database.Executor, variantID string, quantity int, unitPriceCents int64) commerce.Cart {
	t.Helper()
	ctx := context.Background()
	cart, err := svc.GetOrCreateCart(ctx, db, "", "anon-"+variantID, "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AddItem(ctx, db, cart.ID, variantID, quantity, unitPriceCents); err != nil {
		t.Fatal(err)
	}
	return cart
}

func TestCheckoutReservesStockAndCreatesOrder(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	store := svc.Repository

	if err := store.EnsureStock(ctx, db, "variant-belt", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-belt", 2, 10000)

	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-happy-1",
		ItemTitles: map[string]string{"variant-belt": "Deri Kemer"}, ItemSKUs: map[string]string{"variant-belt": "KMR-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.TotalCents != 20000 || order.PaymentStatus != commerce.PaymentPending {
		t.Fatalf("unexpected order: %+v", order)
	}
	stock, err := store.GetStock(ctx, db, "variant-belt")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Available != 3 || stock.Reserved != 2 {
		t.Fatalf("stock after checkout = %+v, want available=3 reserved=2", stock)
	}
}

// TestCheckoutRollsBackWhenAnyItemIsOutOfStock is the end-to-end proof of
// the whole module's reason to exist, at the orchestration layer: a cart
// with one available item and one sold-out item must reserve NEITHER --
// the already-reserved item's hold must roll back with the transaction,
// not leak out as a real reservation for an order that never got created.
func TestCheckoutRollsBackWhenAnyItemIsOutOfStock(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	store := svc.Repository

	if err := store.EnsureStock(ctx, db, "variant-plenty", 10); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureStock(ctx, db, "variant-soldout", 0); err != nil {
		t.Fatal(err)
	}
	cart, err := svc.GetOrCreateCart(ctx, db, "", "anon-mixed", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AddItem(ctx, db, cart.ID, "variant-plenty", 1, 5000); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddItem(ctx, db, cart.ID, "variant-soldout", 1, 5000); err != nil {
		t.Fatal(err)
	}

	_, err = svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-mixed-1"})
	if !errors.Is(err, commerce.ErrOutOfStock) {
		t.Fatalf("expected ErrOutOfStock, got %v", err)
	}

	plenty, err := store.GetStock(ctx, db, "variant-plenty")
	if err != nil {
		t.Fatal(err)
	}
	if plenty.Available != 10 || plenty.Reserved != 0 {
		t.Fatalf("the in-stock item's reservation should have rolled back entirely, got %+v", plenty)
	}
}

func TestCheckoutIsIdempotentAtServiceLevel(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	store := svc.Repository

	if err := store.EnsureStock(ctx, db, "variant-retry", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-retry", 1, 1000)

	req := commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-retry-1"}
	first, err := svc.Checkout(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Checkout(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("retried checkout should return the same order, got %q then %q", first.ID, second.ID)
	}
	stock, err := store.GetStock(ctx, db, "variant-retry")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Reserved != 1 {
		t.Fatalf("retried checkout must not reserve stock twice, reserved = %d, want 1", stock.Reserved)
	}
}

func TestConfirmPaymentCommitsReservationExactlyOnce(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	store := svc.Repository

	if err := store.EnsureStock(ctx, db, "variant-pay", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-pay", 2, 1000)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-pay-1"})
	if err != nil {
		t.Fatal(err)
	}

	applied, err := svc.ConfirmPayment(ctx, db, "paytr", "evt-1", order.ID)
	if err != nil || !applied {
		t.Fatalf("applied=%v err=%v", applied, err)
	}
	stock, err := store.GetStock(ctx, db, "variant-pay")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Available != 3 || stock.Reserved != 0 {
		t.Fatalf("stock after payment = %+v, want available=3 reserved=0", stock)
	}

	// A redelivered webhook for the same event must not commit twice.
	appliedAgain, err := svc.ConfirmPayment(ctx, db, "paytr", "evt-1", order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if appliedAgain {
		t.Fatal("a redelivered payment event must not be applied twice")
	}
	stock, err = store.GetStock(ctx, db, "variant-pay")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Available != 3 || stock.Reserved != 0 {
		t.Fatalf("stock must be unchanged by a duplicate webhook, got %+v", stock)
	}
}

func TestExpireOrderReleasesStockAndIsIdempotent(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	store := svc.Repository

	if err := store.EnsureStock(ctx, db, "variant-expire", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-expire", 2, 1000)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-expire-1"})
	if err != nil {
		t.Fatal(err)
	}

	expired, err := svc.ExpireOrder(ctx, db, order.ID)
	if err != nil || !expired {
		t.Fatalf("expired=%v err=%v", expired, err)
	}
	stock, err := store.GetStock(ctx, db, "variant-expire")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Available != 5 || stock.Reserved != 0 {
		t.Fatalf("stock after expiry = %+v, want available=5 reserved=0", stock)
	}

	expiredAgain, err := svc.ExpireOrder(ctx, db, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if expiredAgain {
		t.Fatal("expiring an already-expired order a second time should be a no-op")
	}
	stock, err = store.GetStock(ctx, db, "variant-expire")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Available != 5 {
		t.Fatalf("a second expiry sweep must not release stock twice, available = %d, want 5", stock.Available)
	}
}

func TestExpireOrderDoesNotUndoACompletedPayment(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	store := svc.Repository

	if err := store.EnsureStock(ctx, db, "variant-paid-then-expire", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-paid-then-expire", 1, 1000)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-paid-expire-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmPayment(ctx, db, "paytr", "evt-paid-1", order.ID); err != nil {
		t.Fatal(err)
	}

	// The expiry worker sweeping in after payment already succeeded (a slow
	// sweep, a race with the webhook) must find nothing to expire.
	expired, err := svc.ExpireOrder(ctx, db, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if expired {
		t.Fatal("a paid order must never be expired")
	}
	stock, err := store.GetStock(ctx, db, "variant-paid-then-expire")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Available != 4 || stock.Reserved != 0 {
		t.Fatalf("a paid order's committed stock must stay committed, got %+v", stock)
	}
}

type recordedOutboxEvent struct {
	topic       string
	aggregateID string
}

type fakeOutbox struct {
	events []recordedOutboxEvent
}

func (f *fakeOutbox) Enqueue(_ context.Context, _ database.Executor, _, topic, aggregateID string, _ int) error {
	f.events = append(f.events, recordedOutboxEvent{topic: topic, aggregateID: aggregateID})
	return nil
}

func TestCheckoutEmitsOrderCreatedExactlyOnceEvenWhenRetried(t *testing.T) {
	svc, db := newTestService(t)
	outbox := &fakeOutbox{}
	svc.Outbox = outbox
	ctx := context.Background()
	store := svc.Repository

	if err := store.EnsureStock(ctx, db, "variant-event-order", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-event-order", 1, 1000)
	req := commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-event-1"}

	order, err := svc.Checkout(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Checkout(ctx, req); err != nil {
		t.Fatal(err)
	}

	if len(outbox.events) != 1 || outbox.events[0].topic != "order.created" || outbox.events[0].aggregateID != order.ID {
		t.Fatalf("expected exactly one order.created event for %s, got %+v", order.ID, outbox.events)
	}
}

func TestConfirmPaymentEmitsOrderPaidOnlyWhenTheTransitionApplies(t *testing.T) {
	svc, db := newTestService(t)
	outbox := &fakeOutbox{}
	svc.Outbox = outbox
	ctx := context.Background()
	store := svc.Repository

	if err := store.EnsureStock(ctx, db, "variant-event-paid", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-event-paid", 1, 1000)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-event-2"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.ConfirmPayment(ctx, db, "paytr", "evt-event-1", order.ID); err != nil {
		t.Fatal(err)
	}
	// A redelivered webhook must not emit a second order.paid event.
	if _, err := svc.ConfirmPayment(ctx, db, "paytr", "evt-event-1", order.ID); err != nil {
		t.Fatal(err)
	}

	var paidEvents []recordedOutboxEvent
	for _, event := range outbox.events {
		if event.topic == "order.paid" {
			paidEvents = append(paidEvents, event)
		}
	}
	if len(paidEvents) != 1 || paidEvents[0].aggregateID != order.ID {
		t.Fatalf("expected exactly one order.paid event for %s, got %+v", order.ID, paidEvents)
	}
}
