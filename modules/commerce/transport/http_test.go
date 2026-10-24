package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/notify"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
	"github.com/ardaozyildirim/hydrogo/modules/commerce/repository/sqlstore"
)

func newTestHandler(t *testing.T) (Handler, database.Executor) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "commerce-http.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sqlstore.Store{Dialect: database.SQLite}
	svc := commerce.Service{Transactions: database.NewTransactor(db), Repository: store}
	return Handler{Service: svc, DB: db}, db
}

func decodeBody(t *testing.T, recorder *httptest.ResponseRecorder, into any) {
	t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v (%s)", err, recorder.Body.String())
	}
	if err := json.Unmarshal(envelope.Data, into); err != nil {
		t.Fatalf("decode data: %v (%s)", err, envelope.Data)
	}
}

// TestAnonymousVisitorGetsADurableCartCookie proves the one piece of
// genuinely new infrastructure this phase needed: nothing else in the
// codebase gives an anonymous visitor a cookie-based identity they can use
// across requests (confirmed by research before writing this -- see
// docs/decisions). The same cookie, replayed, must return the same cart.
func TestAnonymousVisitorGetsADurableCartCookie(t *testing.T) {
	h, _ := newTestHandler(t)

	first := httptest.NewRequest(http.MethodGet, "/api/v1/commerce/cart", nil)
	firstRec := httptest.NewRecorder()
	h.getCart(firstRec, first)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first cart request: %d %s", firstRec.Code, firstRec.Body)
	}
	cookies := firstRec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != cartCookieName || cookies[0].Value == "" {
		t.Fatalf("expected a cart cookie to be set, got %+v", cookies)
	}
	var firstCart cartResponse
	decodeBody(t, firstRec, &firstCart)

	second := httptest.NewRequest(http.MethodGet, "/api/v1/commerce/cart", nil)
	second.AddCookie(cookies[0])
	secondRec := httptest.NewRecorder()
	h.getCart(secondRec, second)
	var secondCart cartResponse
	decodeBody(t, secondRec, &secondCart)

	if secondCart.ID != firstCart.ID {
		t.Fatalf("replaying the same cart cookie should return the same cart, got %q then %q", firstCart.ID, secondCart.ID)
	}
	// No new cookie should be set once one already exists.
	if len(secondRec.Result().Cookies()) != 0 {
		t.Fatalf("a request carrying a valid cart cookie should not receive a new one: %+v", secondRec.Result().Cookies())
	}
}

func TestAddItemThenCheckoutEndToEnd(t *testing.T) {
	h, db := newTestHandler(t)
	store := h.Service.Repository
	ctx := context.Background()
	if err := store.EnsureStock(ctx, db, "variant-http-1", 3); err != nil {
		t.Fatal(err)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(`{"variant_id":"variant-http-1","quantity":2,"unit_price_cents":1500}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	h.addItem(addRec, addReq)
	if addRec.Code != http.StatusOK {
		t.Fatalf("add item: %d %s", addRec.Code, addRec.Body)
	}
	cookie := addRec.Result().Cookies()[0]

	checkoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(`{"guest_email":"guest@example.com","idempotency_key":"http-idem-1"}`))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(cookie)
	checkoutRec := httptest.NewRecorder()
	h.checkout(checkoutRec, checkoutReq)
	if checkoutRec.Code != http.StatusCreated {
		t.Fatalf("checkout: %d %s", checkoutRec.Code, checkoutRec.Body)
	}

	stock, err := store.GetStock(ctx, db, "variant-http-1")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Available != 1 || stock.Reserved != 2 {
		t.Fatalf("stock after checkout = %+v, want available=1 reserved=2", stock)
	}
}

func TestCheckoutAcceptsItemTitlesAndSKUsSoTheOrderReceiptIsNotBlank(t *testing.T) {
	h, db := newTestHandler(t)
	store := h.Service.Repository
	ctx := context.Background()
	if err := store.EnsureStock(ctx, db, "variant-http-titled", 3); err != nil {
		t.Fatal(err)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(`{"variant_id":"variant-http-titled","quantity":1,"unit_price_cents":1500}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	h.addItem(addRec, addReq)
	cookie := addRec.Result().Cookies()[0]

	checkoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(
		`{"guest_email":"guest@example.com","idempotency_key":"http-idem-titled-1","item_titles":{"variant-http-titled":"Deri Kemer"},"item_skus":{"variant-http-titled":"KMR-001"}}`,
	))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(cookie)
	checkoutRec := httptest.NewRecorder()
	h.checkout(checkoutRec, checkoutReq)
	if checkoutRec.Code != http.StatusCreated {
		t.Fatalf("checkout: %d %s", checkoutRec.Code, checkoutRec.Body)
	}
	var envelope struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(checkoutRec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	items, err := h.Service.Repository.ListOrderItems(ctx, db, envelope.Data.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "Deri Kemer" || items[0].SKU != "KMR-001" {
		t.Fatalf("expected the submitted title/sku to reach the order item, got %+v", items)
	}
}

func TestCheckoutReturns409WhenOutOfStock(t *testing.T) {
	h, db := newTestHandler(t)
	store := h.Service.Repository
	ctx := context.Background()
	if err := store.EnsureStock(ctx, db, "variant-http-soldout", 0); err != nil {
		t.Fatal(err)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(`{"variant_id":"variant-http-soldout","quantity":1,"unit_price_cents":100}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	h.addItem(addRec, addReq)
	cookie := addRec.Result().Cookies()[0]

	checkoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(`{"guest_email":"guest@example.com","idempotency_key":"http-idem-2"}`))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(cookie)
	checkoutRec := httptest.NewRecorder()
	h.checkout(checkoutRec, checkoutReq)
	if checkoutRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for out-of-stock checkout, got %d %s", checkoutRec.Code, checkoutRec.Body)
	}
}

func TestCheckoutRejectsGuestWithoutEmail(t *testing.T) {
	h, db := newTestHandler(t)
	store := h.Service.Repository
	ctx := context.Background()
	if err := store.EnsureStock(ctx, db, "variant-http-noemail", 5); err != nil {
		t.Fatal(err)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(`{"variant_id":"variant-http-noemail","quantity":1,"unit_price_cents":100}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	h.addItem(addRec, addReq)
	cookie := addRec.Result().Cookies()[0]

	checkoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(`{"idempotency_key":"http-idem-3"}`))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(cookie)
	checkoutRec := httptest.NewRecorder()
	h.checkout(checkoutRec, checkoutReq)
	if checkoutRec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for a guest checkout without an email, got %d %s", checkoutRec.Code, checkoutRec.Body)
	}
}

func TestSetStockEndpointUpsertsAvailable(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/stock", bytes.NewBufferString(`{"variant_id":"variant-admin-1","available":7}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.setStock(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("set stock: %d %s", rec.Code, rec.Body)
	}
	stock, err := h.Service.Repository.GetStock(ctx, db, "variant-admin-1")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Available != 7 {
		t.Fatalf("available = %d, want 7", stock.Available)
	}
}

func TestGetCurrenciesReturnsTheConfiguredDisplayCurrenciesAndRates(t *testing.T) {
	h, _ := newTestHandler(t)
	h.DisplayCurrencies = []string{"USD", "EUR"}
	h.CurrencyRates = map[string]float64{"USD": 0.03, "EUR": 0.028}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/commerce/currencies", nil)
	rec := httptest.NewRecorder()
	h.getCurrencies(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get currencies: %d %s", rec.Code, rec.Body)
	}
	var envelope struct {
		Data struct {
			Default string             `json:"default"`
			Display []string           `json:"display"`
			Rates   map[string]float64 `json:"rates"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	if envelope.Data.Default != "TRY" || len(envelope.Data.Display) != 2 || envelope.Data.Rates["USD"] != 0.03 {
		t.Fatalf("unexpected currencies response: %+v", envelope.Data)
	}
}

func TestGetCheckoutConfigReturnsTheConfiguredTaxAndShipping(t *testing.T) {
	h, _ := newTestHandler(t)
	h.Service.Pricing = commerce.PricingConfig{TaxRate: 0.20, TaxInclusive: true, ShippingFlatRateCents: 500, ShippingFreeAboveCents: 50000}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/commerce/checkout-config", nil)
	rec := httptest.NewRecorder()
	h.getCheckoutConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get checkout config: %d %s", rec.Code, rec.Body)
	}
	var envelope struct {
		Data struct {
			TaxRate                float64 `json:"tax_rate"`
			TaxInclusive           bool    `json:"tax_inclusive"`
			ShippingFlatRateCents  int64   `json:"shipping_flat_rate_cents"`
			ShippingFreeAboveCents int64   `json:"shipping_free_above_cents"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	if envelope.Data.TaxRate != 0.20 || !envelope.Data.TaxInclusive || envelope.Data.ShippingFlatRateCents != 500 || envelope.Data.ShippingFreeAboveCents != 50000 {
		t.Fatalf("unexpected checkout config: %+v", envelope.Data)
	}
}

func TestGetStockReturnsAvailableButNotReservedCount(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-public-stock", 5); err != nil {
		t.Fatal(err)
	}
	if ok, err := h.Service.Repository.ReserveStock(ctx, db, "variant-public-stock", 2); err != nil || !ok {
		t.Fatalf("reserve: ok=%v err=%v", ok, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/commerce/stock/variant-public-stock", nil)
	req.SetPathValue("variantID", "variant-public-stock")
	rec := httptest.NewRecorder()
	h.getStock(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get stock: %d %s", rec.Code, rec.Body)
	}
	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	if _, hasReserved := envelope.Data["reserved"]; hasReserved {
		t.Fatalf("public stock read must not expose the reserved count, got %s", rec.Body.String())
	}
	var available int
	if err := json.Unmarshal(envelope.Data["available"], &available); err != nil {
		t.Fatal(err)
	}
	if available != 3 {
		t.Fatalf("available = %d, want 3 (5 - 2 reserved)", available)
	}
}

func TestGetStockReturnsZeroForAnUnknownVariantInsteadOfAnError(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/commerce/stock/variant-never-stocked", nil)
	req.SetPathValue("variantID", "variant-never-stocked")
	rec := httptest.NewRecorder()
	h.getStock(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get stock: %d %s", rec.Code, rec.Body)
	}
}

type fakePaymentProvider struct {
	err   error
	start commerce.PaymentStart
}

func (f fakePaymentProvider) StartPayment(ctx context.Context, order commerce.Order, items []commerce.OrderItem) (commerce.PaymentStart, error) {
	if f.err != nil {
		return commerce.PaymentStart{}, f.err
	}
	return f.start, nil
}

type recordedOrderMail struct {
	email, orderID, total string
	items                 []mailer.OrderConfirmationItem
}

type fakeMailer struct {
	calls []recordedOrderMail
	err   error
}

func (f *fakeMailer) SendOrderConfirmation(_ context.Context, email, orderID, total string, items []mailer.OrderConfirmationItem) error {
	f.calls = append(f.calls, recordedOrderMail{email: email, orderID: orderID, total: total, items: items})
	return f.err
}

type fakeNotifier struct{ calls []notify.Message }

func (f *fakeNotifier) Send(_ context.Context, msg notify.Message) error {
	f.calls = append(f.calls, msg)
	return nil
}

func TestCheckoutSendsOrderConfirmationEmailAndOperatorNotification(t *testing.T) {
	h, db := newTestHandler(t)
	mail := &fakeMailer{}
	notifier := &fakeNotifier{}
	h.Mailer = mail
	h.Notifier = notifier
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-notify", 3); err != nil {
		t.Fatal(err)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(`{"variant_id":"variant-http-notify","quantity":1,"unit_price_cents":1500}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	h.addItem(addRec, addReq)
	cookie := addRec.Result().Cookies()[0]

	checkoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(
		`{"guest_email":"guest@example.com","idempotency_key":"http-idem-notify-1","item_titles":{"variant-http-notify":"Deri Kemer"}}`,
	))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(cookie)
	checkoutRec := httptest.NewRecorder()
	h.checkout(checkoutRec, checkoutReq)
	if checkoutRec.Code != http.StatusCreated {
		t.Fatalf("checkout: %d %s", checkoutRec.Code, checkoutRec.Body)
	}

	if len(mail.calls) != 1 {
		t.Fatalf("expected exactly 1 order confirmation email, got %d", len(mail.calls))
	}
	if mail.calls[0].email != "guest@example.com" || len(mail.calls[0].items) != 1 || mail.calls[0].items[0].Title != "Deri Kemer" {
		t.Fatalf("unexpected mail call: %+v", mail.calls[0])
	}
	if len(notifier.calls) != 1 || notifier.calls[0].Title != "Yeni sipariş" {
		t.Fatalf("unexpected notifier calls: %+v", notifier.calls)
	}
}

func TestCheckoutStillSucceedsWhenMailerFails(t *testing.T) {
	h, db := newTestHandler(t)
	h.Mailer = &fakeMailer{err: errors.New("smtp down")}
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-notify-fail", 3); err != nil {
		t.Fatal(err)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(`{"variant_id":"variant-http-notify-fail","quantity":1,"unit_price_cents":1500}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	h.addItem(addRec, addReq)
	cookie := addRec.Result().Cookies()[0]

	checkoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(`{"guest_email":"guest@example.com","idempotency_key":"http-idem-notify-fail-1"}`))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(cookie)
	checkoutRec := httptest.NewRecorder()
	h.checkout(checkoutRec, checkoutReq)
	if checkoutRec.Code != http.StatusCreated {
		t.Fatalf("checkout must still succeed when the mailer fails: %d %s", checkoutRec.Code, checkoutRec.Body)
	}
}

func TestCheckoutIncludesThePaymentProvidersRedirectWhenOneIsConfigured(t *testing.T) {
	h, db := newTestHandler(t)
	h.PaymentProvider = fakePaymentProvider{start: commerce.PaymentStart{RedirectURL: "https://pay.example.test/abc"}}
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-pay", 3); err != nil {
		t.Fatal(err)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(`{"variant_id":"variant-http-pay","quantity":1,"unit_price_cents":1500}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	h.addItem(addRec, addReq)
	cookie := addRec.Result().Cookies()[0]

	checkoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(`{"guest_email":"guest@example.com","idempotency_key":"http-idem-pay-1"}`))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(cookie)
	checkoutRec := httptest.NewRecorder()
	h.checkout(checkoutRec, checkoutReq)
	if checkoutRec.Code != http.StatusCreated {
		t.Fatalf("checkout: %d %s", checkoutRec.Code, checkoutRec.Body)
	}

	var envelope struct {
		Data struct {
			Payment struct {
				RedirectURL string `json:"redirect_url"`
			} `json:"payment"`
		} `json:"data"`
	}
	if err := json.Unmarshal(checkoutRec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v (%s)", err, checkoutRec.Body.String())
	}
	if envelope.Data.Payment.RedirectURL != "https://pay.example.test/abc" {
		t.Fatalf("payment redirect url = %q, want the provider's url (%s)", envelope.Data.Payment.RedirectURL, checkoutRec.Body.String())
	}
}

func TestCheckoutStillSucceedsWhenThePaymentProviderFails(t *testing.T) {
	h, db := newTestHandler(t)
	h.PaymentProvider = fakePaymentProvider{err: errors.New("provider unreachable")}
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-pay-fail", 3); err != nil {
		t.Fatal(err)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(`{"variant_id":"variant-http-pay-fail","quantity":1,"unit_price_cents":1500}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	h.addItem(addRec, addReq)
	cookie := addRec.Result().Cookies()[0]

	checkoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(`{"guest_email":"guest@example.com","idempotency_key":"http-idem-pay-fail-1"}`))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(cookie)
	checkoutRec := httptest.NewRecorder()
	h.checkout(checkoutRec, checkoutReq)
	if checkoutRec.Code != http.StatusCreated {
		t.Fatalf("checkout must still succeed when the payment provider fails: %d %s", checkoutRec.Code, checkoutRec.Body)
	}
}

func TestStartOrderPaymentRetriesAfterAnEarlierProviderFailure(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-retry", 3); err != nil {
		t.Fatal(err)
	}
	cart, err := h.Service.GetOrCreateCart(ctx, db, "", "anon-retry", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Service.AddItem(ctx, db, cart.ID, "variant-http-retry", 1, 1500); err != nil {
		t.Fatal(err)
	}
	order, err := h.Service.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "retry-idem-1"})
	if err != nil {
		t.Fatal(err)
	}

	h.PaymentProvider = fakePaymentProvider{start: commerce.PaymentStart{RedirectURL: "https://pay.example.test/retry"}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/orders/"+order.ID+"/payment", nil)
	req.SetPathValue("id", order.ID)
	rec := httptest.NewRecorder()
	h.startOrderPayment(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("start order payment: %d %s", rec.Code, rec.Body)
	}
	var envelope struct {
		Data commerce.PaymentStart `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	if envelope.Data.RedirectURL != "https://pay.example.test/retry" {
		t.Fatalf("redirect url = %q", envelope.Data.RedirectURL)
	}
}

func TestStartOrderPaymentRejectsAnAlreadyPaidOrder(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-paid", 3); err != nil {
		t.Fatal(err)
	}
	cart, err := h.Service.GetOrCreateCart(ctx, db, "", "anon-paid", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Service.AddItem(ctx, db, cart.ID, "variant-http-paid", 1, 1500); err != nil {
		t.Fatal(err)
	}
	order, err := h.Service.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "paid-idem-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Service.ConfirmPayment(ctx, db, "paytr", "evt-1", order.ID); err != nil {
		t.Fatal(err)
	}

	h.PaymentProvider = fakePaymentProvider{start: commerce.PaymentStart{RedirectURL: "https://pay.example.test/should-not-be-used"}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/orders/"+order.ID+"/payment", nil)
	req.SetPathValue("id", order.ID)
	rec := httptest.NewRecorder()
	h.startOrderPayment(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for an already-paid order, got %d %s", rec.Code, rec.Body)
	}
}

func TestGetOrderReturnsTheOrderAndItsItemsByID(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-get-order", 3); err != nil {
		t.Fatal(err)
	}
	cart, err := h.Service.GetOrCreateCart(ctx, db, "", "anon-get-order", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Service.AddItem(ctx, db, cart.ID, "variant-http-get-order", 2, 1500); err != nil {
		t.Fatal(err)
	}
	order, err := h.Service.Checkout(ctx, commerce.CheckoutRequest{
		CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "get-order-idem-1",
		ItemTitles: map[string]string{"variant-http-get-order": "Cüzdan"},
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/commerce/orders/"+order.ID, nil)
	req.SetPathValue("id", order.ID)
	rec := httptest.NewRecorder()
	h.getOrder(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get order: %d %s", rec.Code, rec.Body)
	}
	var envelope struct {
		Data struct {
			ID    string `json:"id"`
			Items []struct {
				Title    string `json:"title"`
				Quantity int    `json:"quantity"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	if envelope.Data.ID != order.ID || len(envelope.Data.Items) != 1 || envelope.Data.Items[0].Title != "Cüzdan" || envelope.Data.Items[0].Quantity != 2 {
		t.Fatalf("unexpected order view: %+v", envelope.Data)
	}
}

func TestGetOrderReturns404ForAnUnknownID(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/commerce/orders/does-not-exist", nil)
	req.SetPathValue("id", "does-not-exist")
	rec := httptest.NewRecorder()
	h.getOrder(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d %s", rec.Code, rec.Body)
	}
}

func TestStartOrderPaymentRequiresAConfiguredProvider(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-noprovider", 3); err != nil {
		t.Fatal(err)
	}
	cart, err := h.Service.GetOrCreateCart(ctx, db, "", "anon-noprovider", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Service.AddItem(ctx, db, cart.ID, "variant-http-noprovider", 1, 1500); err != nil {
		t.Fatal(err)
	}
	order, err := h.Service.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "noprovider-idem-1"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/orders/"+order.ID+"/payment", nil)
	req.SetPathValue("id", order.ID)
	rec := httptest.NewRecorder()
	h.startOrderPayment(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 with no configured provider, got %d %s", rec.Code, rec.Body)
	}
}

func TestCreateDiscountCodeThenUseItAtCheckout(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/discount-codes", bytes.NewBufferString(`{"code":"SAVE10","kind":"percent","value":10}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	h.createDiscountCode(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create discount code: %d %s", createRec.Code, createRec.Body)
	}

	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-discount", 3); err != nil {
		t.Fatal(err)
	}
	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(`{"variant_id":"variant-http-discount","quantity":1,"unit_price_cents":10000}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	h.addItem(addRec, addReq)
	cookie := addRec.Result().Cookies()[0]

	checkoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(`{"guest_email":"guest@example.com","idempotency_key":"http-idem-discount-1","discount_code":"SAVE10"}`))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(cookie)
	checkoutRec := httptest.NewRecorder()
	h.checkout(checkoutRec, checkoutReq)
	if checkoutRec.Code != http.StatusCreated {
		t.Fatalf("checkout: %d %s", checkoutRec.Code, checkoutRec.Body)
	}
	var envelope struct {
		Data struct {
			DiscountCode  string `json:"discount_code"`
			DiscountCents int64  `json:"discount_cents"`
			TotalCents    int64  `json:"total_cents"`
		} `json:"data"`
	}
	if err := json.Unmarshal(checkoutRec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v (%s)", err, checkoutRec.Body.String())
	}
	if envelope.Data.DiscountCode != "SAVE10" || envelope.Data.DiscountCents != 1000 || envelope.Data.TotalCents != 9000 {
		t.Fatalf("unexpected checkout response: %+v", envelope.Data)
	}
}

func TestCheckoutReturns422ForAnInvalidDiscountCode(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-baddiscount", 3); err != nil {
		t.Fatal(err)
	}
	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(`{"variant_id":"variant-http-baddiscount","quantity":1,"unit_price_cents":10000}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	h.addItem(addRec, addReq)
	cookie := addRec.Result().Cookies()[0]

	checkoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(`{"guest_email":"guest@example.com","idempotency_key":"http-idem-baddiscount-1","discount_code":"NOPE"}`))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(cookie)
	checkoutRec := httptest.NewRecorder()
	h.checkout(checkoutRec, checkoutReq)
	if checkoutRec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d %s", checkoutRec.Code, checkoutRec.Body)
	}
}
