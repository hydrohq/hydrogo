package commerce_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/migrations"
	"github.com/hydrohq/hydrogo/modules/commerce"
	"github.com/hydrohq/hydrogo/modules/commerce/repository/sqlstore"
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
	svc := commerce.Service{Transactions: database.NewTransactor(db), Repository: store, Prices: commerce.StaticPrices{}}
	return svc, db
}

func seedCartWithItem(t *testing.T, svc commerce.Service, db database.Executor, variantID string, quantity int, unitPriceCents int64) commerce.Cart {
	t.Helper()
	ctx := context.Background()
	cart, err := svc.GetOrCreateCart(ctx, db, "", "anon-"+variantID, "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := addPriced(svc, ctx, db, cart, variantID, quantity, unitPriceCents); err != nil {
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
	if err := addPriced(svc, ctx, db, cart, "variant-plenty", 1, 5000); err != nil {
		t.Fatal(err)
	}
	if err := addPriced(svc, ctx, db, cart, "variant-soldout", 1, 5000); err != nil {
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

	applied, err := pay(svc, ctx, "mock-provider", "evt-1", order.ID)
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
	appliedAgain, err := pay(svc, ctx, "mock-provider", "evt-1", order.ID)
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

	expired, err := svc.ExpireOrder(ctx, order.ID)
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

	expiredAgain, err := svc.ExpireOrder(ctx, order.ID)
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
	if _, err := pay(svc, ctx, "mock-provider", "evt-paid-1", order.ID); err != nil {
		t.Fatal(err)
	}

	// The expiry worker sweeping in after payment already succeeded (a slow
	// sweep, a race with the webhook) must find nothing to expire.
	expired, err := svc.ExpireOrder(ctx, order.ID)
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

	if _, err := pay(svc, ctx, "mock-provider", "evt-event-1", order.ID); err != nil {
		t.Fatal(err)
	}
	// A redelivered webhook must not emit a second order.paid event.
	if _, err := pay(svc, ctx, "mock-provider", "evt-event-1", order.ID); err != nil {
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

func TestCheckoutAppliesConfiguredTaxAndShipping(t *testing.T) {
	svc, db := newTestService(t)
	svc.Pricing = commerce.PricingConfig{TaxRate: 0.20, ShippingFlatRateCents: 500}
	ctx := context.Background()
	if err := svc.Repository.EnsureStock(ctx, db, "variant-pricing", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-pricing", 1, 10000)

	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-pricing-1"})
	if err != nil {
		t.Fatal(err)
	}
	if order.SubtotalCents != 10000 {
		t.Fatalf("subtotal = %d, want 10000", order.SubtotalCents)
	}
	if order.TaxCents != 2000 {
		t.Fatalf("tax = %d, want 2000 (20%%)", order.TaxCents)
	}
	if order.ShippingCents != 500 {
		t.Fatalf("shipping = %d, want 500", order.ShippingCents)
	}
	if order.TotalCents != 12500 {
		t.Fatalf("total = %d, want 12500 (10000 + 2000 tax + 500 shipping)", order.TotalCents)
	}
}

func TestCheckoutAppliesAValidDiscountCodeAndRedeemsItExactlyOnce(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	if err := svc.Repository.EnsureStock(ctx, db, "variant-discount", 5); err != nil {
		t.Fatal(err)
	}
	store := sqlstore.Store{Dialect: database.SQLite}
	if err := store.CreateDiscountCode(ctx, db, commerce.DiscountCode{
		ID: "disc-checkout-1", Code: "SAVE10", Kind: commerce.DiscountPercent, Value: 10, Active: true, MaxRedemptions: 1,
	}); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-discount", 1, 10000)

	req := commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-discount-1", DiscountCode: "SAVE10"}
	order, err := svc.Checkout(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if order.DiscountCode != "SAVE10" || order.DiscountCents != 1000 || order.TotalCents != 9000 {
		t.Fatalf("unexpected order: %+v", order)
	}

	// A retried checkout (same idempotency key) must not redeem the code a
	// second time -- it's the same order being returned, not a new one.
	if _, err := svc.Checkout(ctx, req); err != nil {
		t.Fatal(err)
	}
	code, err := store.GetDiscountCodeByCode(ctx, db, "SAVE10")
	if err != nil {
		t.Fatal(err)
	}
	if code.RedeemedCount != 1 {
		t.Fatalf("redeemed_count = %d, want 1 (one real checkout, one idempotent retry)", code.RedeemedCount)
	}
}

func TestCheckoutRejectsAnInvalidDiscountCodeWithoutCreatingAnOrder(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	if err := svc.Repository.EnsureStock(ctx, db, "variant-baddiscount", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-baddiscount", 1, 10000)

	_, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-baddiscount-1", DiscountCode: "DOES-NOT-EXIST"})
	if !errors.Is(err, commerce.ErrDiscountCodeInvalid) {
		t.Fatalf("expected ErrDiscountCodeInvalid, got %v", err)
	}
	stock, err := svc.Repository.GetStock(ctx, db, "variant-baddiscount")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Available != 5 || stock.Reserved != 0 {
		t.Fatalf("a rejected discount code must leave stock untouched (rolled back), got %+v", stock)
	}
}

type fakeShippingProvider struct {
	name  string
	calls int
	err   error
}

func (f *fakeShippingProvider) Name() string { return f.name }

func (f *fakeShippingProvider) CreateShipment(ctx context.Context, order commerce.Order, items []commerce.OrderItem) (commerce.ShipmentResult, error) {
	f.calls++
	if f.err != nil {
		return commerce.ShipmentResult{}, f.err
	}
	return commerce.ShipmentResult{TrackingNumber: "TRK-" + order.ID, TrackingURL: "https://example.com/track/" + order.ID}, nil
}

func payOrder(t *testing.T, svc commerce.Service, db database.Executor, orderID string) {
	t.Helper()
	if _, err := pay(svc, context.Background(), "mock-provider", "evt-"+orderID, orderID); err != nil {
		t.Fatal(err)
	}
}

func TestCreateShipmentRecordsTrackingAndMovesOrderToShipped(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	if err := svc.Repository.EnsureStock(ctx, db, "variant-ship", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-ship", 1, 10000)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-ship-1"})
	if err != nil {
		t.Fatal(err)
	}
	payOrder(t, svc, db, order.ID)

	provider := &fakeShippingProvider{name: "carrier-a"}
	shipped, err := svc.CreateShipment(ctx, provider, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if shipped.FulfillmentStatus != commerce.FulfillmentShipped {
		t.Fatalf("fulfillment_status = %q, want shipped", shipped.FulfillmentStatus)
	}
	if shipped.ShippingCarrier != "carrier-a" || shipped.ShippingTrackingNumber != "TRK-"+order.ID {
		t.Fatalf("unexpected shipping fields: %+v", shipped)
	}
	if provider.calls != 1 {
		t.Fatalf("provider called %d times, want 1", provider.calls)
	}
}

func TestCreateShipmentRejectsAnUnpaidOrder(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	if err := svc.Repository.EnsureStock(ctx, db, "variant-unpaid", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-unpaid", 1, 10000)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-unpaid-1"})
	if err != nil {
		t.Fatal(err)
	}

	provider := &fakeShippingProvider{name: "carrier-a"}
	_, err = svc.CreateShipment(ctx, provider, order.ID)
	if !errors.Is(err, commerce.ErrOrderNotShippable) {
		t.Fatalf("expected ErrOrderNotShippable, got %v", err)
	}
	if provider.calls != 0 {
		t.Fatal("the carrier must never be called for an unpaid order")
	}
}

// TestCreateShipmentRejectsAnAlreadyShippedOrderWithoutCallingTheCarrierAgain
// is the real-world-side-effect-safe version of
// TestConfirmPaymentCommitsReservationExactlyOnce's guard proof: unlike a
// payment webhook dedup (which protects a write after an external event
// already happened), the carrier call here IS the external event, so a
// retried request must never reach the carrier a second time, not just
// avoid a second database write.
func TestCreateShipmentRejectsAnAlreadyShippedOrderWithoutCallingTheCarrierAgain(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	if err := svc.Repository.EnsureStock(ctx, db, "variant-reship", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-reship", 1, 10000)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-reship-1"})
	if err != nil {
		t.Fatal(err)
	}
	payOrder(t, svc, db, order.ID)

	provider := &fakeShippingProvider{name: "carrier-a"}
	if _, err := svc.CreateShipment(ctx, provider, order.ID); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 {
		t.Fatalf("provider called %d times after the first shipment, want 1", provider.calls)
	}

	_, err = svc.CreateShipment(ctx, provider, order.ID)
	if !errors.Is(err, commerce.ErrOrderNotShippable) {
		t.Fatalf("expected ErrOrderNotShippable on a retry, got %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("provider called %d times after a retry, want 1 (the carrier must never be called twice for the same order)", provider.calls)
	}
}

func TestStocksByVariantIDs(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	// Empty list returns empty slice without error
	stocks, err := svc.StocksByVariantIDs(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stocks) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(stocks))
	}

	// Seed stock rows
	if err := svc.Repository.EnsureStock(ctx, db, "var-svc-1", 10); err != nil {
		t.Fatal(err)
	}
	if err := svc.Repository.EnsureStock(ctx, db, "var-svc-2", 20); err != nil {
		t.Fatal(err)
	}
	// Reserve some stock on var-svc-1
	if ok, err := svc.Repository.ReserveStock(ctx, db, "var-svc-1", 3); err != nil || !ok {
		t.Fatalf("reserve stock: ok=%v err=%v", ok, err)
	}

	// Query known and unknown variants
	stocks, err = svc.StocksByVariantIDs(ctx, []string{"var-svc-1", "var-svc-unknown", "var-svc-2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(stocks) != 2 {
		t.Fatalf("expected 2 stock items (unknown omitted), got %d", len(stocks))
	}
	stockMap := make(map[string]commerce.Stock)
	for _, s := range stocks {
		stockMap[s.VariantID] = s
	}

	s1, ok1 := stockMap["var-svc-1"]
	if !ok1 {
		t.Fatal("var-svc-1 missing")
	}
	if s1.Available != 7 || s1.Reserved != 3 {
		t.Fatalf("var-svc-1 stock mismatch: available=%d (want 7), reserved=%d (want 3)", s1.Available, s1.Reserved)
	}

	s2, ok2 := stockMap["var-svc-2"]
	if !ok2 {
		t.Fatal("var-svc-2 missing")
	}
	if s2.Available != 20 || s2.Reserved != 0 {
		t.Fatalf("var-svc-2 stock mismatch: available=%d (want 20), reserved=%d (want 0)", s2.Available, s2.Reserved)
	}
}

type trackingProvider struct {
	attempts []commerce.Order
}

func (p *trackingProvider) StartPayment(_ context.Context, order commerce.Order, _ []commerce.OrderItem) (commerce.PaymentStart, error) {
	p.attempts = append(p.attempts, order)
	if len(p.attempts) == 1 {
		return commerce.PaymentStart{Method: "gateway", ProviderRef: "ref-attempt-1", RedirectURL: "https://pay/1"}, nil
	}
	return commerce.PaymentStart{Method: "gateway", ProviderRef: "ref-attempt-2", RedirectURL: "https://pay/2"}, nil
}

func TestStartPaymentPersistsMethodAndProviderRef(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	if err := svc.Repository.EnsureStock(ctx, db, "var-start-ref", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "var-start-ref", 1, 2500)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "shop@example.com", IdempotencyKey: "idem-start-ref-1"})
	if err != nil {
		t.Fatal(err)
	}

	p := &trackingProvider{}
	if _, err := svc.StartPayment(ctx, p, order.ID); err != nil {
		t.Fatal(err)
	}
	stored1, err := svc.Order(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored1.PaymentMethod != "gateway" || stored1.PaymentRef != "ref-attempt-1" {
		t.Fatalf("after attempt 1 got method=%q ref=%q", stored1.PaymentMethod, stored1.PaymentRef)
	}

	if _, err := svc.StartPayment(ctx, p, order.ID); err != nil {
		t.Fatal(err)
	}
	if len(p.attempts) != 2 || p.attempts[1].PaymentRef != "ref-attempt-1" {
		t.Fatalf("second attempt should see previous PaymentRef, got %+v", p.attempts)
	}
	stored2, err := svc.Order(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored2.PaymentRef != "ref-attempt-2" {
		t.Fatalf("after attempt 2 got ref=%q, want ref-attempt-2", stored2.PaymentRef)
	}
	if stored2.HoldUntil.IsZero() || stored2.HoldUntil.Before(time.Now().UTC().Add(20*time.Minute)) {
		t.Fatalf("expected StartPayment retry to extend HoldUntil >= 20m, got %v", stored2.HoldUntil)
	}
}

func TestCheckoutPreventsConcurrentTwoTabDuplicateOrdersFromOneCart(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	if err := svc.Repository.EnsureStock(ctx, db, "variant-two-tab", 10); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-two-tab", 1, 5000)

	// Run two concurrent checkouts with distinct idempotency keys against the same cart.
	type outcome struct {
		order commerce.Order
		err   error
	}
	results := make(chan outcome, 2)
	start := make(chan struct{})
	for _, key := range []string{"tab-1", "tab-2"} {
		go func(idem string) {
			<-start
			o, err := svc.Checkout(ctx, commerce.CheckoutRequest{
				CartID: cart.ID, GuestEmail: "shopper@example.test", IdempotencyKey: idem,
			})
			results <- outcome{order: o, err: err}
		}(key)
	}
	close(start)

	var successes int
	var emptyErrs int
	for i := 0; i < 2; i++ {
		res := <-results
		if res.err == nil {
			successes++
		} else if errors.Is(res.err, commerce.ErrEmptyCart) {
			emptyErrs++
		} else {
			t.Fatalf("unexpected concurrent checkout error: %v", res.err)
		}
	}
	if successes != 1 || emptyErrs != 1 {
		t.Fatalf("got successes=%d emptyErrs=%d, want 1 winner and 1 ErrEmptyCart", successes, emptyErrs)
	}
}

func TestCheckoutNormalizesPromoCaseAndAutoSettlesZeroTotalAndRefundReleasesCoupon(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	store := sqlstore.Store{Dialect: database.SQLite}
	if err := svc.Repository.EnsureStock(ctx, db, "variant-promo-zero", 5); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateDiscountCode(ctx, db, commerce.DiscountCode{
		ID: "disc-free-100", Code: "FREE100", Kind: commerce.DiscountPercent, Value: 100, Active: true, MaxRedemptions: 2,
	}); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-promo-zero", 1, 5000)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: cart.ID, GuestEmail: "guest@example.test", IdempotencyKey: "idem-free-1",
		DiscountCode: "  free100 ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.TotalCents != 0 || order.PaymentStatus != commerce.PaymentPaid {
		t.Fatalf("expected 100%% coupon order to auto-settle as paid with total=0, got status=%s total=%d", order.PaymentStatus, order.TotalCents)
	}
	stock, err := svc.Repository.GetStock(ctx, db, "variant-promo-zero")
	if err != nil || stock.Available != 4 || stock.Reserved != 0 {
		t.Fatalf("expected stock committed (avail=4, res=0), got %+v err=%v", stock, err)
	}

	// Now test unshipped refund releasing discount code redemption.
	if err := store.CreateDiscountCode(ctx, db, commerce.DiscountCode{
		ID: "disc-half-50", Code: "HALF50", Kind: commerce.DiscountPercent, Value: 50, Active: true, MaxRedemptions: 1,
	}); err != nil {
		t.Fatal(err)
	}
	cart2 := seedCartWithItem(t, svc, db, "variant-promo-zero", 1, 5000)
	order2, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: cart2.ID, GuestEmail: "guest@example.test", IdempotencyKey: "idem-half-1", DiscountCode: "half50",
	})
	if err != nil {
		t.Fatal(err)
	}
	payOrder(t, svc, db, order2.ID)
	if _, err := svc.RefundOrder(ctx, &fakeRefunder{}, order2.ID); err != nil {
		t.Fatal(err)
	}
	code, err := store.GetDiscountCodeByCode(ctx, db, "HALF50")
	if err != nil {
		t.Fatal(err)
	}
	if code.RedeemedCount != 0 {
		t.Fatalf("redeemed_count after unshipped refund = %d, want 0", code.RedeemedCount)
	}
}
