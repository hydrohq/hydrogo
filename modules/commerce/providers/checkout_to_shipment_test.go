// Package providers_test chains PayTR and Yurtiçi Kargo through the real
// commerce.Service pipeline against mock HTTP servers for both -- not
// against the real services (neither has a usable sandbox, see each
// package's own doc comment), but enough to prove checkout, payment
// confirmation and shipment creation hand off to each other correctly end
// to end, which the two providers' own per-method unit tests, each mocking
// only its own call, cannot show on their own.
package providers_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
	"github.com/ardaozyildirim/hydrogo/modules/commerce/providers/paytr"
	"github.com/ardaozyildirim/hydrogo/modules/commerce/providers/yurticikargo"
	"github.com/ardaozyildirim/hydrogo/modules/commerce/repository/sqlstore"
)

// paytrCallbackHash independently reproduces PayTR's documented callback
// signature (merchant_oid + salt + status + total_amount, HMAC-SHA256 with
// the merchant key) instead of calling into the paytr package's own
// unexported sign(), so this test cannot pass merely because the package
// agrees with itself.
func paytrCallbackHash(cfg paytr.Config, orderID, status, totalAmount string) string {
	mac := hmac.New(sha256.New, []byte(cfg.MerchantKey))
	mac.Write([]byte(orderID + cfg.MerchantSalt + status + totalAmount))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestCheckoutThroughPaymentToShipmentAgainstMockPayTRAndYurticiKargo(t *testing.T) {
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

	// 1. Start a payment against a mock PayTR get-token endpoint.
	paytrServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "token": "mock-token-1"})
	}))
	defer paytrServer.Close()
	paymentProvider := paytr.Provider{Config: paytr.Config{
		MerchantID: "merchant-1", MerchantKey: "key-1", MerchantSalt: "salt-1", APIBaseURL: paytrServer.URL,
	}}
	start, err := svc.StartPayment(ctx, paymentProvider, order.ID)
	if err != nil {
		t.Fatalf("start payment: %v", err)
	}
	if start.Token != "mock-token-1" {
		t.Fatalf("payment token = %q", start.Token)
	}

	// 2. Simulate PayTR's server-to-server callback the way PayTR's own
	// infrastructure would send it: a signed POST to the real handler, not
	// a direct VerifyCallback call.
	callbackHandler := paytr.Handler{Service: svc, Provider: paymentProvider}
	form := url.Values{
		"merchant_oid": {order.ID}, "status": {"success"},
		"total_amount": {strconv.FormatInt(order.TotalCents, 10)},
	}
	form.Set("hash", paytrCallbackHash(paymentProvider.Config, order.ID, "success", strconv.FormatInt(order.TotalCents, 10)))
	req := httptest.NewRequest(http.MethodPost, paytr.CallbackPath, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	callbackHandler.Routes()[0].Handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "OK" {
		t.Fatalf("paytr callback: status=%d body=%q", recorder.Code, recorder.Body.String())
	}

	paid, err := svc.Repository.GetOrder(ctx, db, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if paid.PaymentStatus != commerce.PaymentPaid {
		t.Fatalf("payment_status after callback = %q, want paid", paid.PaymentStatus)
	}

	// 3. Hand the now-paid order to a mock Yurtiçi Kargo createShipment
	// endpoint and confirm the order comes out shipped with tracking info.
	kargoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <createShipmentResponse><outFlag>0</outFlag><outResult>Basarili</outResult><jobId>job-pipeline-1</jobId></createShipmentResponse>
  </soapenv:Body>
</soapenv:Envelope>`))
	}))
	defer kargoServer.Close()
	shippingProvider := yurticikargo.Provider{Config: yurticikargo.Config{
		WSUserName: "ws-user", WSPassword: "ws-pass", BaseURL: kargoServer.URL,
	}}
	shipped, err := svc.CreateShipment(ctx, shippingProvider, order.ID)
	if err != nil {
		t.Fatalf("create shipment: %v", err)
	}
	if shipped.FulfillmentStatus != commerce.FulfillmentShipped {
		t.Fatalf("fulfillment_status = %q, want shipped", shipped.FulfillmentStatus)
	}
	if shipped.ShippingCarrier != "yurtici-kargo" || shipped.ShippingTrackingNumber != order.ID {
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
	if _, err := svc.ConfirmPayment(ctx, commerce.Payment{Provider: "paytr", EventID: order.ID, OrderID: order.ID, AmountCents: order.TotalCents}); err != nil {
		t.Fatalf("confirm payment: %v", err)
	}

	kargoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <createShipmentResponse><outFlag>1</outFlag><outResult>Alici adresi gecersiz</outResult></createShipmentResponse>
  </soapenv:Body>
</soapenv:Envelope>`))
	}))
	defer kargoServer.Close()
	shippingProvider := yurticikargo.Provider{Config: yurticikargo.Config{
		WSUserName: "ws-user", WSPassword: "ws-pass", BaseURL: kargoServer.URL,
	}}
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

	// Retrying against a server that now accepts must succeed -- proves the
	// released claim, not just the error, is correct.
	kargoServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <createShipmentResponse><outFlag>0</outFlag><outResult>Basarili</outResult></createShipmentResponse>
  </soapenv:Body>
</soapenv:Envelope>`))
	}))
	defer kargoServer2.Close()
	retryProvider := yurticikargo.Provider{Config: yurticikargo.Config{
		WSUserName: "ws-user", WSPassword: "ws-pass", BaseURL: kargoServer2.URL,
	}}
	if _, err := svc.CreateShipment(ctx, retryProvider, order.ID); err != nil {
		t.Fatalf("retry after a released claim: %v", err)
	}
}
