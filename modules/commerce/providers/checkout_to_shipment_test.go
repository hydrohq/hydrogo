// Package providers_test chains checkout, payment confirmation and carrier
// shipment creation through the real commerce.Service pipeline.
package providers_test

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

type mockPaymentProvider struct {
	token string
}

func (m mockPaymentProvider) StartPayment(ctx context.Context, order commerce.Order, items []commerce.OrderItem) (commerce.PaymentStart, error) {
	return commerce.PaymentStart{Token: m.token}, nil
}

type mockShippingProvider struct {
	carrier string
	refuse  bool
}

func (m mockShippingProvider) Name() string { return m.carrier }

func (m mockShippingProvider) CreateShipment(ctx context.Context, order commerce.Order, items []commerce.OrderItem) (commerce.ShipmentResult, error) {
	if m.refuse {
		return commerce.ShipmentResult{}, errors.New("carrier refused shipment")
	}
	return commerce.ShipmentResult{
		TrackingNumber: order.ID,
		TrackingURL:    "https://tracking.example.test/?code=" + order.ID,
	}, nil
}

func TestCheckoutThroughPaymentToShipmentAgainstMockProviders(t *testing.T) {
	ctx := commerce.WithClientIP(context.Background(), "203.0.113.9")
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "checkout-to-shipment.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sqlstore.Store{Dialect: database.SQLite}
	svc := commerce.Service{
		Transactions: database.NewTransactor(db),
		Repository:   store,
		Prices:       commerce.StaticPrices{"variant-boot": {Cents: 45000, Currency: "TRY"}},
	}
	if err := store.EnsureStock(ctx, db, "variant-boot", 3); err != nil {
		t.Fatal(err)
	}

	cart, err := svc.GetOrCreateCart(ctx, db, "", "anon-pipeline", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AddItem(ctx, db, cart, "variant-boot", 1); err != nil {
		t.Fatal(err)
	}

	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: cart.ID, IdempotencyKey: "pipeline-1",
		GuestName: "Ayşe Yılmaz", GuestEmail: "aysee@example.test", GuestPhone: "0532 111 22 33",
		GuestAddress: "Örnek Mah. No:5", GuestCity: "İstanbul", GuestDistrict: "Kadıköy",
	})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if order.PaymentStatus != commerce.PaymentPending {
		t.Fatalf("new order payment_status = %q, want pending", order.PaymentStatus)
	}

	// 1. Start a payment with a mock payment provider.
	paymentProvider := mockPaymentProvider{token: "mock-token-1"}
	start, err := svc.StartPayment(ctx, paymentProvider, order.ID)
	if err != nil {
		t.Fatalf("start payment: %v", err)
	}
	if start.Token != "mock-token-1" {
		t.Fatalf("payment token = %q", start.Token)
	}

	// 2. Confirm payment.
	if _, err := svc.ConfirmPayment(ctx, commerce.Payment{
		Provider: "mock-provider", EventID: "evt-pipeline-1", OrderID: order.ID, AmountCents: order.TotalCents,
	}); err != nil {
		t.Fatalf("confirm payment: %v", err)
	}

	paid, err := svc.Repository.GetOrder(ctx, db, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if paid.PaymentStatus != commerce.PaymentPaid {
		t.Fatalf("payment_status after callback = %q, want paid", paid.PaymentStatus)
	}

	// 3. Hand the now-paid order to a carrier shippingProvider
	// and confirm the order comes out shipped with tracking info.
	shippingProvider := mockShippingProvider{carrier: "test-carrier"}
	shipped, err := svc.CreateShipment(ctx, shippingProvider, order.ID)
	if err != nil {
		t.Fatalf("create shipment: %v", err)
	}
	if shipped.FulfillmentStatus != commerce.FulfillmentShipped {
		t.Fatalf("fulfillment_status = %q, want shipped", shipped.FulfillmentStatus)
	}
	if shipped.ShippingCarrier != "test-carrier" || shipped.ShippingTrackingNumber != order.ID {
		t.Fatalf("carrier/tracking = %q/%q", shipped.ShippingCarrier, shipped.ShippingTrackingNumber)
	}
}

// TestCreateShipmentReleasesTheClaimWhenTheCarrierRefuses chains checkout
// and payment the same way, then proves a carrier-side refusal (not a
// transport error) still releases the shipment claim instead of leaving
// the order stuck in "preparing" forever.
func TestCreateShipmentReleasesTheClaimWhenTheCarrierRefuses(t *testing.T) {
	ctx := commerce.WithClientIP(context.Background(), "203.0.113.9")
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "checkout-to-shipment-refused.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sqlstore.Store{Dialect: database.SQLite}
	svc := commerce.Service{
		Transactions: database.NewTransactor(db),
		Repository:   store,
		Prices:       commerce.StaticPrices{"variant-boot": {Cents: 45000, Currency: "TRY"}},
	}
	if err := store.EnsureStock(ctx, db, "variant-boot", 3); err != nil {
		t.Fatal(err)
	}
	cart, err := svc.GetOrCreateCart(ctx, db, "", "anon-pipeline-2", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AddItem(ctx, db, cart, "variant-boot", 1); err != nil {
		t.Fatal(err)
	}
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID: cart.ID, IdempotencyKey: "pipeline-2",
		GuestName: "Ayşe Yılmaz", GuestEmail: "aysee@example.test", GuestCity: "İstanbul",
	})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if _, err := svc.ConfirmPayment(ctx, commerce.Payment{Provider: "mock-provider", EventID: order.ID, OrderID: order.ID, AmountCents: order.TotalCents}); err != nil {
		t.Fatalf("confirm payment: %v", err)
	}

	shippingProvider := mockShippingProvider{carrier: "test-carrier", refuse: true}
	if _, err := svc.CreateShipment(ctx, shippingProvider, order.ID); err == nil {
		t.Fatal("expected an error when the carrier refuses the shipment")
	}

	after, err := svc.Repository.GetOrder(ctx, db, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.FulfillmentStatus != commerce.FulfillmentUnfulfilled {
		t.Fatalf("fulfillment_status after a carrier refusal = %q, want unfulfilled (claim released, retryable)", after.FulfillmentStatus)
	}

	// Retrying against a provider that now accepts must succeed -- proves the
	// released claim, not just the error, is correct.
	retryProvider := mockShippingProvider{carrier: "test-carrier", refuse: false}
	if _, err := svc.CreateShipment(ctx, retryProvider, order.ID); err != nil {
		t.Fatalf("retry after a released claim: %v", err)
	}
}
