package commerce_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
	"github.com/ardaozyildirim/hydrogo/modules/commerce/repository/sqlstore"
)

func placeOne(t *testing.T, svc commerce.Service, db database.Executor, anon, variant string, price int64, key string) (commerce.Cart, commerce.Order) {
	t.Helper()
	ctx := context.Background()
	if err := svc.Repository.EnsureStock(ctx, db, variant, 10); err != nil {
		t.Fatal(err)
	}
	cart, err := svc.GetOrCreateCart(ctx, db, "", anon, "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := addPriced(svc, ctx, db, cart, variant, 1, price); err != nil {
		t.Fatal(err)
	}
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "g@example.com", IdempotencyKey: key, Currency: "TRY"})
	if err != nil {
		t.Fatal(err)
	}
	return cart, order
}

// The catalog, not the cart, sets the price that is charged -- even when
// the price changes after the item went into the cart.
func TestCheckoutChargesTheCatalogPriceNotTheStoredCartPrice(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	if err := svc.Repository.EnsureStock(ctx, db, "variant-reprice", 5); err != nil {
		t.Fatal(err)
	}
	cart, err := svc.GetOrCreateCart(ctx, db, "", "anon-reprice", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := addPriced(svc, ctx, db, cart, "variant-reprice", 2, 1000); err != nil {
		t.Fatal(err)
	}
	svc.Prices.(commerce.StaticPrices)["variant-reprice"] = commerce.Price{Cents: 2500}
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "g@example.com", IdempotencyKey: "idem-reprice"})
	if err != nil {
		t.Fatal(err)
	}
	if order.SubtotalCents != 5000 {
		t.Fatalf("subtotal = %d, want 5000 (2 x catalog price 2500)", order.SubtotalCents)
	}
}

func TestCheckoutRefusesWithoutAPriceSource(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	cart, err := svc.GetOrCreateCart(ctx, db, "", "anon-noprices", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	svc.Prices = nil
	if err := svc.AddItem(ctx, db, cart, "variant-x", 1); !errors.Is(err, commerce.ErrNoPriceSource) {
		t.Fatalf("AddItem without a price source: %v, want ErrNoPriceSource", err)
	}
}

func TestAddItemRejectsAnUnknownVariantAndAForeignCurrency(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	cart, err := svc.GetOrCreateCart(ctx, db, "", "anon-unknown", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AddItem(ctx, db, cart, "variant-not-in-catalog", 1); !errors.Is(err, commerce.ErrUnknownVariant) {
		t.Fatalf("unknown variant: %v", err)
	}
	svc.Prices.(commerce.StaticPrices)["variant-usd"] = commerce.Price{Cents: 100, Currency: "USD"}
	if err := svc.AddItem(ctx, db, cart, "variant-usd", 1); !errors.Is(err, commerce.ErrCurrencyMismatch) {
		t.Fatalf("USD item in a TRY cart: %v", err)
	}
}

func TestCheckoutUsesTheRequestedCurrencyAndEmptiesTheCart(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	if err := svc.Repository.EnsureStock(ctx, db, "variant-usd-cart", 5); err != nil {
		t.Fatal(err)
	}
	cart, err := svc.GetOrCreateCart(ctx, db, "", "anon-usd", "USD")
	if err != nil {
		t.Fatal(err)
	}
	svc.Prices.(commerce.StaticPrices)["variant-usd-cart"] = commerce.Price{Cents: 1999, Currency: "USD"}
	if err := svc.AddItem(ctx, db, cart, "variant-usd-cart", 1); err != nil {
		t.Fatal(err)
	}
	order, created, err := svc.PlaceOrder(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "g@example.com", IdempotencyKey: "idem-usd", Currency: "USD"})
	if err != nil {
		t.Fatal(err)
	}
	if !created || order.Currency != "USD" {
		t.Fatalf("created=%v currency=%q, want true/USD", created, order.Currency)
	}
	items, err := svc.Repository.ListItems(ctx, db, cart.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("cart still has %d items after checkout; a second click would order them again", len(items))
	}
}

func TestIdempotentRetryFromTheSameCartReturnsTheOrderButNotAsNew(t *testing.T) {
	svc, db := newTestService(t)
	cart, first := placeOne(t, svc, db, "anon-retry2", "variant-retry2", 1000, "idem-retry2")
	again, created, err := svc.PlaceOrder(context.Background(), commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "g@example.com", IdempotencyKey: "idem-retry2"})
	if err != nil {
		t.Fatal(err)
	}
	if created || again.ID != first.ID {
		t.Fatalf("retry: created=%v id=%s, want false/%s", created, again.ID, first.ID)
	}
}

func TestAnIdempotencyKeyFromAnotherCartNeverReturnsThatOrder(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	placeOne(t, svc, db, "anon-victim", "variant-victim", 1000, "idem-shared")
	attacker, err := svc.GetOrCreateCart(ctx, db, "", "anon-attacker", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.PlaceOrder(ctx, commerce.CheckoutRequest{CartID: attacker.ID, GuestEmail: "a@example.com", IdempotencyKey: "idem-shared"})
	if !errors.Is(err, commerce.ErrIdempotencyConflict) {
		t.Fatalf("foreign key replay: %v, want ErrIdempotencyConflict", err)
	}
}

// lostRedemption simulates the other checkout winning the last redemption
// between DiscountFor's read and the guarded UPDATE.
type lostRedemption struct{ sqlstore.Store }

func (lostRedemption) RedeemDiscountCode(context.Context, database.Executor, string, int) (bool, error) {
	return false, nil
}

func TestCheckoutFailsWhenTheLastDiscountRedemptionWasTakenConcurrently(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	svc.Repository = lostRedemption{Store: sqlstore.Store{Dialect: database.SQLite}}
	if err := svc.Repository.CreateDiscountCode(ctx, db, commerce.DiscountCode{ID: "disc-last", Code: "LAST", Kind: commerce.DiscountFixed, Value: 100, MaxRedemptions: 1, Active: true}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Repository.EnsureStock(ctx, db, "variant-last", 5); err != nil {
		t.Fatal(err)
	}
	cart, err := svc.GetOrCreateCart(ctx, db, "", "anon-last", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := addPriced(svc, ctx, db, cart, "variant-last", 1, 1000); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "g@example.com", IdempotencyKey: "idem-last", DiscountCode: "LAST"})
	if !errors.Is(err, commerce.ErrDiscountCodeInvalid) {
		t.Fatalf("lost redemption race: %v, want ErrDiscountCodeInvalid", err)
	}
	stock, err := svc.Repository.GetStock(ctx, db, "variant-last")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Reserved != 0 {
		t.Fatalf("the failed checkout must roll back its reservation, reserved=%d", stock.Reserved)
	}
}

func TestConfirmPaymentRejectsAShortAmountAndAcceptsInstallmentInterest(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	_, short := placeOne(t, svc, db, "anon-short", "variant-short", 1000, "idem-short")
	applied, err := svc.ConfirmPayment(ctx, commerce.Payment{Provider: "mock-provider", EventID: "evt-short", OrderID: short.ID, AmountCents: 1})
	if applied || !errors.Is(err, commerce.ErrPaymentAmountMismatch) {
		t.Fatalf("1 kuruş for a 10 TRY order: applied=%v err=%v", applied, err)
	}
	order, _, err := svc.OrderWithItems(ctx, short.ID)
	if err != nil {
		t.Fatal(err)
	}
	if order.PaymentStatus != commerce.PaymentPending {
		t.Fatalf("short payment marked the order %s", order.PaymentStatus)
	}

	_, interest := placeOne(t, svc, db, "anon-interest", "variant-interest", 1000, "idem-interest")
	applied, err = svc.ConfirmPayment(ctx, commerce.Payment{Provider: "mock-provider", EventID: "evt-interest", OrderID: interest.ID, AmountCents: interest.TotalCents + 150})
	if err != nil || !applied {
		t.Fatalf("payment with installment interest: applied=%v err=%v", applied, err)
	}
}

func TestAPaymentForAnExpiredOrderIsReportedNotSilentlyDropped(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	_, order := placeOne(t, svc, db, "anon-late", "variant-late", 1000, "idem-late")
	if _, err := svc.ExpireOrder(ctx, order.ID); err != nil {
		t.Fatal(err)
	}
	applied, err := svc.ConfirmPayment(ctx, commerce.Payment{Provider: "mock-provider", EventID: "evt-late", OrderID: order.ID, AmountCents: order.TotalCents})
	if applied || !errors.Is(err, commerce.ErrLatePayment) {
		t.Fatalf("late payment: applied=%v err=%v, want ErrLatePayment", applied, err)
	}
	applied, err = svc.ConfirmPayment(ctx, commerce.Payment{Provider: "mock-provider", EventID: "evt-late", OrderID: order.ID, AmountCents: order.TotalCents})
	if applied || err != nil {
		t.Fatalf("a redelivery of the same late notification must be quiet: applied=%v err=%v", applied, err)
	}
}

type failingCarrier struct{ calls int }

func (f *failingCarrier) Name() string { return "test-carrier" }
func (f *failingCarrier) CreateShipment(context.Context, commerce.Order, []commerce.OrderItem) (commerce.ShipmentResult, error) {
	f.calls++
	return commerce.ShipmentResult{}, errors.New("carrier down")
}

func TestACarrierFailureGivesTheShipmentClaimBack(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	_, order := placeOne(t, svc, db, "anon-carrier", "variant-carrier", 1000, "idem-carrier")
	payOrder(t, svc, db, order.ID)

	if _, err := svc.CreateShipment(ctx, &failingCarrier{}, order.ID); err == nil {
		t.Fatal("a carrier failure must be returned")
	}
	current, _, err := svc.OrderWithItems(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.FulfillmentStatus != commerce.FulfillmentUnfulfilled {
		t.Fatalf("after a carrier failure the order is %s, want unfulfilled so it can be retried", current.FulfillmentStatus)
	}
	retry := &fakeShippingProvider{name: "test-carrier"}
	if _, err := svc.CreateShipment(ctx, retry, order.ID); err != nil || retry.calls != 1 {
		t.Fatalf("retry after release: err=%v calls=%d", err, retry.calls)
	}
}

func TestAShipmentInFlightBlocksASecondRequestWithoutCallingTheCarrier(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	_, order := placeOne(t, svc, db, "anon-inflight", "variant-inflight", 1000, "idem-inflight")
	payOrder(t, svc, db, order.ID)
	claimed, err := svc.Repository.ClaimShipment(ctx, db, order.ID, time.Now().UTC().Add(-time.Hour))
	if err != nil || !claimed {
		t.Fatalf("first claim: %v %v", claimed, err)
	}
	second := &fakeShippingProvider{name: "test-carrier"}
	if _, err := svc.CreateShipment(ctx, second, order.ID); !errors.Is(err, commerce.ErrOrderNotShippable) {
		t.Fatalf("second request while a claim is live: %v", err)
	}
	if second.calls != 0 {
		t.Fatal("the carrier must not be called while another request holds the claim")
	}
}

// dbPrices reads the database on every lookup, like the real catalog price
// source does through the content-type store.
type dbPrices struct {
	db    *sql.DB
	price int64
}

func (p dbPrices) UnitPrice(ctx context.Context, _ string) (commerce.Price, error) {
	var one int
	if err := p.db.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
		return commerce.Price{}, err
	}
	return commerce.Price{Cents: p.price}, nil
}

// With a one-connection pool (the compact profile) a price lookup made
// inside the cart or checkout transaction would wait forever for the
// connection that transaction holds.
func TestPricingNeverDeadlocksOnAOneConnectionPool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "one-conn.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	store := sqlstore.Store{Dialect: database.SQLite}
	svc := commerce.Service{Transactions: database.NewTransactor(db), Repository: store, Prices: dbPrices{db: db, price: 1234}}
	if err := store.EnsureStock(ctx, db, "variant-onecon", 5); err != nil {
		t.Fatal(err)
	}
	shopper := commerce.Shopper{AnonymousToken: "anon-onecon", Currency: "TRY"}
	cart, _, err := svc.AddToCart(ctx, shopper, "variant-onecon", 2)
	if err != nil {
		t.Fatalf("AddToCart on a one-connection pool: %v", err)
	}
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "g@example.com", IdempotencyKey: "idem-onecon", Currency: "TRY"})
	if err != nil {
		t.Fatalf("Checkout on a one-connection pool: %v", err)
	}
	if order.SubtotalCents != 2468 {
		t.Fatalf("subtotal = %d, want 2468", order.SubtotalCents)
	}
}
