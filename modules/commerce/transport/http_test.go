package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/internal/mailer"
	"github.com/hydrohq/hydrogo/internal/notify"
	"github.com/hydrohq/hydrogo/migrations"
	"github.com/hydrohq/hydrogo/modules/commerce"
	"github.com/hydrohq/hydrogo/modules/commerce/repository/sqlstore"
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
	svc := commerce.Service{Transactions: database.NewTransactor(db), Repository: store, Prices: commerce.StaticPrices{}}
	return Handler{Service: svc}, db
}

type blockingOrderRepository struct {
	commerce.Repository
	entered chan struct{}
	release chan struct{}
}

func (r blockingOrderRepository) GetOrder(ctx context.Context, _ database.Executor, _ string) (commerce.Order, error) {
	close(r.entered)
	<-r.release
	return commerce.Order{PaymentStatus: commerce.PaymentPending}, nil
}

func TestPendingTrackerDoesNotHoldGlobalLockDuringOrderRead(t *testing.T) {
	h, _ := newTestHandler(t)
	entered, release := make(chan struct{}), make(chan struct{})
	svc := commerce.Service{Transactions: h.Service.Transactions, Repository: blockingOrderRepository{entered: entered, release: release}}
	tracker := NewPendingOrderTracker()
	tracker.mu.Lock()
	tracker.orders["slow-ip"] = []pendingOrderEntry{{reservation: "existing", orderID: "existing", createdAt: time.Now()}}
	tracker.mu.Unlock()
	done := make(chan struct{})
	go func() {
		tracker.CountPending(context.Background(), "slow-ip", svc, time.Now(), time.Minute)
		close(done)
	}()
	<-entered
	if _, ok := tracker.TryReserve("other-ip", 1); !ok {
		t.Fatal("other IP was blocked by a slow order read")
	}
	close(release)
	<-done
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

func TestPendingTrackerReservesSlotsAtomically(t *testing.T) {
	tracker := NewPendingOrderTracker()
	var wg sync.WaitGroup
	var mu sync.Mutex
	reserved := 0
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if token, ok := tracker.TryReserve("same-ip", 5); ok {
				mu.Lock()
				reserved++
				mu.Unlock()
				tracker.Confirm("same-ip", token, "order")
			}
		}()
	}
	wg.Wait()
	if reserved != 5 {
		t.Fatalf("reserved %d slots, want 5", reserved)
	}
}

func TestPendingTrackerLazilyRemovesExpiredIPs(t *testing.T) {
	tracker := NewPendingOrderTracker()
	tracker.mu.Lock()
	tracker.orders["expired"] = []pendingOrderEntry{{reservation: "old", createdAt: time.Unix(1, 0)}}
	tracker.mu.Unlock()
	if got := tracker.CountPending(context.Background(), "new-ip", commerce.Service{}, time.Now(), time.Minute); got != 0 {
		t.Fatalf("pending count = %d, want 0", got)
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if _, ok := tracker.orders["expired"]; ok {
		t.Fatal("expired IP entry was retained")
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

	h.Service.Prices.(commerce.StaticPrices)["variant-http-1"] = commerce.Price{Cents: 1500}
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
	WaitNotifications()

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

	h.Service.Prices.(commerce.StaticPrices)["variant-http-titled"] = commerce.Price{Cents: 1500}
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
	WaitNotifications()
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

	h.Service.Prices.(commerce.StaticPrices)["variant-http-soldout"] = commerce.Price{Cents: 100}
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

	h.Service.Prices.(commerce.StaticPrices)["variant-http-noemail"] = commerce.Price{Cents: 100}
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
	mu    sync.Mutex
	calls []recordedOrderMail
	err   error
}

func (f *fakeMailer) SendOrderConfirmation(_ context.Context, email, orderID, total string, items []mailer.OrderConfirmationItem) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, recordedOrderMail{email: email, orderID: orderID, total: total, items: items})
	return f.err
}

type fakeNotifier struct {
	mu    sync.Mutex
	calls []notify.Message
}

func (f *fakeNotifier) Send(_ context.Context, msg notify.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, msg)
	return nil
}

type fakeOrderReader struct {
	order commerce.Order
	items []commerce.OrderItem
	err   error
}

func (f fakeOrderReader) OrderWithItems(_ context.Context, _ string) (commerce.Order, []commerce.OrderItem, error) {
	return f.order, f.items, f.err
}

func TestNotifyOrderPaid(t *testing.T) {
	notifier := &fakeNotifier{}
	reader := fakeOrderReader{
		order: commerce.Order{ID: "ord-test-1", TotalCents: 4500, Currency: "TRY"},
		items: []commerce.OrderItem{
			{Title: "Tişört", Quantity: 2},
			{Title: "Şapka", Quantity: 1},
		},
	}
	NotifyOrderPaid(context.Background(), notifier, reader, "ord-test-1", "mock-provider", nil)
	WaitNotifications()
	if len(notifier.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(notifier.calls))
	}
	msg := notifier.calls[0]
	if msg.Title != "Yeni sipariş (ödendi)" {
		t.Fatalf("unexpected title: %q", msg.Title)
	}
	wantBody := "ord-test-1 — 45.00 TRY (mock-provider, 2 kalem (Tişört x2, Şapka x1))"
	if msg.Body != wantBody {
		t.Fatalf("body = %q, want %q", msg.Body, wantBody)
	}
}

func TestCheckoutSendsOrderConfirmationEmailAndDoesNotNotifyOperatorUntilPaid(t *testing.T) {
	h, db := newTestHandler(t)
	mail := &fakeMailer{}
	notifier := &fakeNotifier{}
	h.Mailer = mail
	h.Notifier = notifier
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-notify", 3); err != nil {
		t.Fatal(err)
	}

	h.Service.Prices.(commerce.StaticPrices)["variant-http-notify"] = commerce.Price{Cents: 1500}
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
	WaitNotifications()

	if len(mail.calls) != 1 {
		t.Fatalf("expected exactly 1 order confirmation email, got %d", len(mail.calls))
	}
	if mail.calls[0].email != "guest@example.com" || len(mail.calls[0].items) != 1 || mail.calls[0].items[0].Title != "Deri Kemer" {
		t.Fatalf("unexpected mail call: %+v", mail.calls[0])
	}
	if len(notifier.calls) != 0 {
		t.Fatalf("checkout alone must not send operator notification, got %d: %+v", len(notifier.calls), notifier.calls)
	}
}

func TestCheckoutStillSucceedsWhenMailerFails(t *testing.T) {
	h, db := newTestHandler(t)
	h.Mailer = &fakeMailer{err: errors.New("smtp down")}
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-notify-fail", 3); err != nil {
		t.Fatal(err)
	}

	h.Service.Prices.(commerce.StaticPrices)["variant-http-notify-fail"] = commerce.Price{Cents: 1500}
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
	WaitNotifications()
}

func TestCheckoutIncludesThePaymentProvidersRedirectWhenOneIsConfigured(t *testing.T) {
	h, db := newTestHandler(t)
	h.PaymentProvider = fakePaymentProvider{start: commerce.PaymentStart{RedirectURL: "https://pay.example.test/abc"}}
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-pay", 3); err != nil {
		t.Fatal(err)
	}

	h.Service.Prices.(commerce.StaticPrices)["variant-http-pay"] = commerce.Price{Cents: 1500}
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
	WaitNotifications()

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

	h.Service.Prices.(commerce.StaticPrices)["variant-http-pay-fail"] = commerce.Price{Cents: 1500}
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
	if err := addPriced(h.Service, ctx, db, cart, "variant-http-retry", 1, 1500); err != nil {
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
	if err := addPriced(h.Service, ctx, db, cart, "variant-http-paid", 1, 1500); err != nil {
		t.Fatal(err)
	}
	order, err := h.Service.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "paid-idem-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pay(h.Service, ctx, "mock-provider", "evt-1", order.ID); err != nil {
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
	if err := addPriced(h.Service, ctx, db, cart, "variant-http-get-order", 2, 1500); err != nil {
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
	if err := addPriced(h.Service, ctx, db, cart, "variant-http-noprovider", 1, 1500); err != nil {
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
	h.Service.Prices.(commerce.StaticPrices)["variant-http-discount"] = commerce.Price{Cents: 10000}
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
	WaitNotifications()
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
	h.Service.Prices.(commerce.StaticPrices)["variant-http-baddiscount"] = commerce.Price{Cents: 10000}
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

type fakeShippingProvider struct {
	name   string
	result commerce.ShipmentResult
	err    error
}

func (f fakeShippingProvider) Name() string { return f.name }

func (f fakeShippingProvider) CreateShipment(ctx context.Context, order commerce.Order, items []commerce.OrderItem) (commerce.ShipmentResult, error) {
	if f.err != nil {
		return commerce.ShipmentResult{}, f.err
	}
	return f.result, nil
}

func TestCreateShipmentEndpointRecordsTracking(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-ship", 3); err != nil {
		t.Fatal(err)
	}
	cart, err := h.Service.GetOrCreateCart(ctx, db, "", "anon-ship", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := addPriced(h.Service, ctx, db, cart, "variant-http-ship", 1, 1500); err != nil {
		t.Fatal(err)
	}
	order, err := h.Service.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "ship-idem-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pay(h.Service, ctx, "mock-provider", "evt-ship-1", order.ID); err != nil {
		t.Fatal(err)
	}

	h.ShippingProvider = fakeShippingProvider{name: "carrier-a", result: commerce.ShipmentResult{TrackingNumber: "TRK123", TrackingURL: "https://example.com/track/TRK123"}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/orders/"+order.ID+"/shipment", nil)
	req.SetPathValue("id", order.ID)
	rec := httptest.NewRecorder()
	h.createShipment(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create shipment: %d %s", rec.Code, rec.Body)
	}
	var shipped commerce.Order
	decodeBody(t, rec, &shipped)
	if shipped.ShippingTrackingNumber != "TRK123" || shipped.FulfillmentStatus != commerce.FulfillmentShipped {
		t.Fatalf("unexpected order after shipment: %+v", shipped)
	}
}

func TestCreateShipmentEndpointReturns503WithNoProviderConfigured(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/orders/does-not-matter/shipment", nil)
	req.SetPathValue("id", "does-not-matter")
	rec := httptest.NewRecorder()
	h.createShipment(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d %s", rec.Code, rec.Body)
	}
}

func TestCreateShipmentEndpointReturns409ForAnUnpaidOrder(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-unpaid-ship", 3); err != nil {
		t.Fatal(err)
	}
	cart, err := h.Service.GetOrCreateCart(ctx, db, "", "anon-unpaid-ship", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := addPriced(h.Service, ctx, db, cart, "variant-http-unpaid-ship", 1, 1500); err != nil {
		t.Fatal(err)
	}
	order, err := h.Service.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "unpaid-ship-idem-1"})
	if err != nil {
		t.Fatal(err)
	}

	h.ShippingProvider = fakeShippingProvider{name: "carrier-a"}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/orders/"+order.ID+"/shipment", nil)
	req.SetPathValue("id", order.ID)
	rec := httptest.NewRecorder()
	h.createShipment(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d %s", rec.Code, rec.Body)
	}
}

func TestListOrdersAdminTrimsToLimitAndReportsHasMore(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	for i, key := range []string{"list-a", "list-b", "list-c"} {
		variant := "variant-" + key
		if err := h.Service.Repository.EnsureStock(ctx, db, variant, 3); err != nil {
			t.Fatal(err)
		}
		cart, err := h.Service.GetOrCreateCart(ctx, db, "", "anon-"+key, "TRY")
		if err != nil {
			t.Fatal(err)
		}
		if err := addPriced(h.Service, ctx, db, cart, variant, 1, int64(1000*(i+1))); err != nil {
			t.Fatal(err)
		}
		if _, err := h.Service.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "idem-" + key}); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/commerce/orders?limit=2", nil)
	rec := httptest.NewRecorder()
	h.listOrdersAdmin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list orders: %d %s", rec.Code, rec.Body)
	}
	var envelope struct {
		Data     []commerce.Order `json:"data"`
		Metadata struct {
			Limit   int  `json:"limit"`
			HasMore bool `json:"has_more"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	if len(envelope.Data) != 2 || !envelope.Metadata.HasMore || envelope.Metadata.Limit != 2 {
		t.Fatalf("expected 2 orders and has_more=true, got %d orders, metadata %+v", len(envelope.Data), envelope.Metadata)
	}

	lastReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/commerce/orders?limit=2&offset=2", nil)
	lastRec := httptest.NewRecorder()
	h.listOrdersAdmin(lastRec, lastReq)
	var last struct {
		Data     []commerce.Order `json:"data"`
		Metadata struct {
			HasMore bool `json:"has_more"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(lastRec.Body.Bytes(), &last); err != nil {
		t.Fatalf("decode: %v (%s)", err, lastRec.Body.String())
	}
	if len(last.Data) != 1 || last.Metadata.HasMore {
		t.Fatalf("expected the last page to have 1 order and has_more=false, got %d, %+v", len(last.Data), last.Metadata)
	}
}

func TestListOrdersAdminReturnsAnEmptyArrayNotNull(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/commerce/orders", nil)
	rec := httptest.NewRecorder()
	h.listOrdersAdmin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list orders: %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"data":[]`) {
		t.Fatalf("an empty order list must serialize as [] so the admin can iterate it, got %s", rec.Body.String())
	}
}

func TestCheckoutPersistsTheGuestDistrict(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-district", 3); err != nil {
		t.Fatal(err)
	}
	h.Service.Prices.(commerce.StaticPrices)["variant-http-district"] = commerce.Price{Cents: 1000}
	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(`{"variant_id":"variant-http-district","quantity":1,"unit_price_cents":1000}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	h.addItem(addRec, addReq)
	cookie := addRec.Result().Cookies()[0]

	checkoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(`{"guest_email":"guest@example.com","guest_city":"İstanbul","guest_district":"Kadıköy","idempotency_key":"http-idem-district-1"}`))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(cookie)
	checkoutRec := httptest.NewRecorder()
	h.checkout(checkoutRec, checkoutReq)
	if checkoutRec.Code != http.StatusCreated {
		t.Fatalf("checkout: %d %s", checkoutRec.Code, checkoutRec.Body)
	}
	WaitNotifications()
	var created commerce.Order
	decodeBody(t, checkoutRec, &created)
	stored, err := h.Service.Repository.GetOrder(ctx, db, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GuestDistrict != "Kadıköy" {
		t.Fatalf("guest_district = %q, want Kadıköy", stored.GuestDistrict)
	}
}

func TestAClientSentUnitPriceIsIgnored(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-price", 3); err != nil {
		t.Fatal(err)
	}
	h.Service.Prices.(commerce.StaticPrices)["variant-http-price"] = commerce.Price{Cents: 49990}
	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(`{"variant_id":"variant-http-price","quantity":1,"unit_price_cents":1}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	h.addItem(addRec, addReq)
	if addRec.Code != http.StatusOK {
		t.Fatalf("add item: %d %s", addRec.Code, addRec.Body)
	}
	cookie := addRec.Result().Cookies()[0]
	checkoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(`{"guest_email":"guest@example.com","idempotency_key":"http-idem-price"}`))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(cookie)
	checkoutRec := httptest.NewRecorder()
	h.checkout(checkoutRec, checkoutReq)
	if checkoutRec.Code != http.StatusCreated {
		t.Fatalf("checkout: %d %s", checkoutRec.Code, checkoutRec.Body)
	}
	WaitNotifications()
	var order commerce.Order
	decodeBody(t, checkoutRec, &order)
	if order.TotalCents != 49990 {
		t.Fatalf("total = %d, want the catalog price 49990 (client sent 1)", order.TotalCents)
	}
}

func TestARetriedCheckoutDoesNotSendASecondReceipt(t *testing.T) {
	h, db := newTestHandler(t)
	mail := &fakeMailer{}
	h.Mailer = mail
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-http-once", 3); err != nil {
		t.Fatal(err)
	}
	h.Service.Prices.(commerce.StaticPrices)["variant-http-once"] = commerce.Price{Cents: 1500}
	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(`{"variant_id":"variant-http-once","quantity":1}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	h.addItem(addRec, addReq)
	cookie := addRec.Result().Cookies()[0]
	for attempt := 0; attempt < 2; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(`{"guest_email":"guest@example.com","idempotency_key":"http-idem-once"}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		h.checkout(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("attempt %d: %d %s", attempt, rec.Code, rec.Body)
		}
	}
	WaitNotifications()
	if len(mail.calls) != 1 {
		t.Fatalf("receipts sent = %d, want 1 for one order", len(mail.calls))
	}
}

func TestAdminRoutesDeclareAPermission(t *testing.T) {
	for _, route := range (Handler{}).Routes() {
		if route.Protected && route.Permission == "" {
			t.Errorf("%s is protected but declares no permission; any signed-in staff user could call it", route.Pattern)
		}
	}
}

type fakeRefunder struct {
	calls int
	err   error
}

func (f *fakeRefunder) Refund(_ context.Context, _ commerce.Order, _ commerce.PaymentCharge) error {
	f.calls++
	return f.err
}

func TestRefundsRequiredAdminEndpoints(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()

	// Seed a refund requirement
	rrq := commerce.RefundRequired{
		ID:          "rrq-test-1",
		OrderID:     "ord-1",
		Provider:    "paytr",
		AmountCents: 5000,
		Currency:    "TRY",
		Reason:      "payment currency mismatch",
		CreatedAt:   time.Now().UTC(),
	}
	if err := h.Service.RecordRefundRequired(ctx, rrq); err != nil {
		t.Fatalf("failed to record refund required: %v", err)
	}

	// 1. GET /api/v1/admin/commerce/refunds-required
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/admin/commerce/refunds-required?unresolved_only=true", nil)
	recList := httptest.NewRecorder()
	h.listRefundsRequired(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200: %s", recList.Code, recList.Body.String())
	}
	var listResp struct {
		Data []commerce.RefundRequired `json:"data"`
	}
	if err := json.Unmarshal(recList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(listResp.Data) != 1 || listResp.Data[0].ID != "rrq-test-1" {
		t.Fatalf("unexpected list response: %+v", listResp.Data)
	}

	// 1b. GET /api/v1/admin/commerce/refunds-required/count -> 1
	reqCount := httptest.NewRequest(http.MethodGet, "/api/v1/admin/commerce/refunds-required/count", nil)
	recCount := httptest.NewRecorder()
	h.countRefundsRequired(recCount, reqCount)
	if recCount.Code != http.StatusOK {
		t.Fatalf("count status = %d, want 200: %s", recCount.Code, recCount.Body.String())
	}
	var countResp struct {
		Data struct {
			Count int `json:"count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recCount.Body.Bytes(), &countResp); err != nil {
		t.Fatalf("unmarshal count error: %v", err)
	}
	if countResp.Data.Count != 1 {
		t.Fatalf("expected count 1, got %d", countResp.Data.Count)
	}

	// 2. POST /api/v1/admin/commerce/refunds-required/{id}/resolve
	resolveBody := `{"note":"refund sent via bank wire"}`
	reqResolve := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/refunds-required/rrq-test-1/resolve", strings.NewReader(resolveBody))
	reqResolve.Header.Set("Content-Type", "application/json")
	reqResolve.SetPathValue("id", "rrq-test-1")
	recResolve := httptest.NewRecorder()
	h.resolveRefundRequired(recResolve, reqResolve)

	if recResolve.Code != http.StatusOK {
		t.Fatalf("resolve status = %d, want 200: %s", recResolve.Code, recResolve.Body.String())
	}

	// 3. GET again with unresolved_only=true should now return 0 items
	recListAfter := httptest.NewRecorder()
	h.listRefundsRequired(recListAfter, reqList)
	var listAfter struct {
		Data []commerce.RefundRequired `json:"data"`
	}
	_ = json.Unmarshal(recListAfter.Body.Bytes(), &listAfter)
	if len(listAfter.Data) != 0 {
		t.Fatalf("expected 0 unresolved items after resolve, got %d", len(listAfter.Data))
	}

	// 3b. GET /api/v1/admin/commerce/refunds-required/count -> 0
	recCountAfter := httptest.NewRecorder()
	h.countRefundsRequired(recCountAfter, reqCount)
	if recCountAfter.Code != http.StatusOK {
		t.Fatalf("count status = %d, want 200: %s", recCountAfter.Code, recCountAfter.Body.String())
	}
	var countAfterResp struct {
		Data struct {
			Count int `json:"count"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recCountAfter.Body.Bytes(), &countAfterResp)
	if countAfterResp.Data.Count != 0 {
		t.Fatalf("expected count 0 after resolve, got %d", countAfterResp.Data.Count)
	}
	_ = db
}

func TestRefundOrderAdminWithShippedStatus(t *testing.T) {
	h, db := newTestHandler(t)
	refunder := &fakeRefunder{}
	h.Refunder = refunder
	ctx := context.Background()

	// Ensure stock, seed cart, and checkout
	if err := h.Service.Repository.EnsureStock(ctx, db, "var-shipped-ref", 10); err != nil {
		t.Fatal(err)
	}
	h.Service.Prices.(commerce.StaticPrices)["var-shipped-ref"] = commerce.Price{Cents: 1500}
	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(`{"variant_id":"var-shipped-ref","quantity":2}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	h.addItem(addRec, addReq)
	if addRec.Code != http.StatusOK {
		t.Fatalf("add item: %d %s", addRec.Code, addRec.Body)
	}
	cookie := addRec.Result().Cookies()[0]

	checkoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(`{"guest_email":"guest@example.com","idempotency_key":"idem-shipped-ref"}`))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(cookie)
	checkoutRec := httptest.NewRecorder()
	h.checkout(checkoutRec, checkoutReq)
	if checkoutRec.Code != http.StatusCreated {
		t.Fatalf("checkout: %d %s", checkoutRec.Code, checkoutRec.Body)
	}
	WaitNotifications()
	var created struct {
		Data commerce.Order `json:"data"`
	}
	if err := json.Unmarshal(checkoutRec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	order := created.Data

	// Confirm payment
	if _, err := h.Service.ConfirmPayment(ctx, commerce.Payment{Provider: "paytr", EventID: order.ID, OrderID: order.ID, AmountCents: order.TotalCents}); err != nil {
		t.Fatal(err)
	}

	// Mark shipped
	shipErr := h.Service.Transactions.Within(ctx, func(tx database.Executor) error {
		claimed, err := h.Service.Repository.ClaimShipment(ctx, tx, order.ID, time.Time{})
		if err != nil || !claimed {
			return err
		}
		shipped, err := h.Service.Repository.RecordShipment(ctx, tx, order.ID, commerce.ShipmentResult{Carrier: "yurtici", TrackingNumber: "TRK-999", TrackingURL: "http://track"}, time.Now().UTC())
		if err != nil || !shipped {
			return err
		}
		return nil
	})
	if shipErr != nil {
		t.Fatal(shipErr)
	}

	// 1. Calling refund without allow_shipped: should fail with 409 Conflict
	reqRefuse := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/orders/"+order.ID+"/refund", nil)
	reqRefuse.SetPathValue("id", order.ID)
	recRefuse := httptest.NewRecorder()
	h.refundOrder(recRefuse, reqRefuse)

	if recRefuse.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 Conflict for shipped order without approval", recRefuse.Code)
	}
	if refunder.calls != 0 {
		t.Fatalf("provider must not be called when shipment permission check fails")
	}

	// 2. Calling refund WITH allow_shipped: true: should succeed
	reqApprove := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/orders/"+order.ID+"/refund", strings.NewReader(`{"allow_shipped":true}`))
	reqApprove.Header.Set("Content-Type", "application/json")
	reqApprove.SetPathValue("id", order.ID)
	recApprove := httptest.NewRecorder()
	h.refundOrder(recApprove, reqApprove)

	if recApprove.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 OK: %s", recApprove.Code, recApprove.Body.String())
	}
	if refunder.calls != 1 {
		t.Fatalf("expected 1 refunder call, got %d", refunder.calls)
	}
}

func TestAddItemHonorsConfiguredLineQuantity(t *testing.T) {
	h, db := newTestHandler(t)
	h.Service.Prices.(commerce.StaticPrices)["variant-limit"] = commerce.Price{Cents: 100}
	if err := h.Service.Repository.EnsureStock(context.Background(), db, "variant-limit", 500); err != nil {
		t.Fatal(err)
	}

	// Default (0 -> 100): 100 accepted, 101 rejected
	for _, tc := range []struct {
		quantity int
		want     int
	}{{101, http.StatusUnprocessableEntity}, {100, http.StatusOK}} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(fmt.Sprintf(`{"variant_id":"variant-limit","quantity":%d}`, tc.quantity)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.addItem(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("default max quantity %d: status=%d body=%s", tc.quantity, rec.Code, rec.Body)
		}
	}

	// Override to 2: 3 rejected, 2 accepted
	h.MaxLineQuantity = 2
	var cookie *http.Cookie
	for _, tc := range []struct {
		quantity int
		want     int
	}{{3, http.StatusUnprocessableEntity}, {2, http.StatusOK}} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/cart/items", bytes.NewBufferString(fmt.Sprintf(`{"variant_id":"variant-limit","quantity":%d}`, tc.quantity)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.addItem(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("override max quantity %d: status=%d body=%s", tc.quantity, rec.Code, rec.Body)
		}
		if rec.Code == http.StatusOK {
			cookie = rec.Result().Cookies()[0]
		}
	}

	// Lower MaxLineQuantity to 1 after cart already holds 2: checkout must reject with 422
	h.MaxLineQuantity = 1
	checkoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", bytes.NewBufferString(`{"guest_email":"shopper@example.com","idempotency_key":"idem-line-limit-1"}`))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(cookie)
	checkoutRec := httptest.NewRecorder()
	h.checkout(checkoutRec, checkoutReq)
	if checkoutRec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("checkout exceeding max_line_quantity: status=%d body=%s", checkoutRec.Code, checkoutRec.Body)
	}
}

func TestListStockAdmin(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()

	// 1. Check route registration
	var found bool
	for _, r := range h.Routes() {
		if r.Pattern == "GET /api/v1/admin/commerce/stock" {
			found = true
			if !r.Protected {
				t.Fatalf("expected protected route")
			}
			if r.Permission != PermissionRead {
				t.Fatalf("expected PermissionRead, got %s", r.Permission)
			}
		}
	}
	if !found {
		t.Fatal("route GET /api/v1/admin/commerce/stock not found in Routes()")
	}

	// 2. Empty or invalid query -> 400
	for _, raw := range []string{"", "variant_ids=", "variant_ids=%20%20", "variant_ids=v1,,v2", "variant_ids=,"} {
		url := "/api/v1/admin/commerce/stock"
		if raw != "" {
			url += "?" + raw
		}
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		h.listStockAdmin(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("query %q: want status 400, got %d (%s)", raw, rec.Code, rec.Body.String())
		}
	}

	// 3. More than 500 IDs -> 400
	tooManyIDs := make([]string, 501)
	for i := range tooManyIDs {
		tooManyIDs[i] = fmt.Sprintf("v-%d", i)
	}
	req501 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/commerce/stock?variant_ids="+strings.Join(tooManyIDs, ","), nil)
	rec501 := httptest.NewRecorder()
	h.listStockAdmin(rec501, req501)
	if rec501.Code != http.StatusBadRequest {
		t.Fatalf("501 IDs: want status 400, got %d (%s)", rec501.Code, rec501.Body.String())
	}

	// 4. Seed variants
	if err := h.Service.Repository.EnsureStock(ctx, db, "v-admin-1", 10); err != nil {
		t.Fatal(err)
	}
	if err := h.Service.Repository.EnsureStock(ctx, db, "v-admin-2", 20); err != nil {
		t.Fatal(err)
	}
	if ok, err := h.Service.Repository.ReserveStock(ctx, db, "v-admin-1", 4); err != nil || !ok {
		t.Fatalf("reserve: ok=%v err=%v", ok, err)
	}

	// 5. Query: v-admin-1, v-admin-unknown, v-admin-2
	// Expected: unknown omitted, reserved/available returned
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/commerce/stock?variant_ids=v-admin-1,v-admin-unknown,v-admin-2", nil)
	rec := httptest.NewRecorder()
	h.listStockAdmin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var envelope struct {
		Data []struct {
			VariantID string `json:"variant_id"`
			Available int    `json:"available"`
			Reserved  int    `json:"reserved"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v (%s)", err, rec.Body.String())
	}

	if len(envelope.Data) != 2 {
		t.Fatalf("expected 2 stock rows (unknown omitted), got %d: %+v", len(envelope.Data), envelope.Data)
	}

	rowMap := make(map[string]struct{ Available, Reserved int })
	for _, row := range envelope.Data {
		rowMap[row.VariantID] = struct{ Available, Reserved int }{row.Available, row.Reserved}
	}

	r1, ok1 := rowMap["v-admin-1"]
	if !ok1 {
		t.Fatal("v-admin-1 missing from response")
	}
	if r1.Available != 6 || r1.Reserved != 4 {
		t.Fatalf("v-admin-1 got available=%d reserved=%d, want available=6 reserved=4", r1.Available, r1.Reserved)
	}

	r2, ok2 := rowMap["v-admin-2"]
	if !ok2 {
		t.Fatal("v-admin-2 missing from response")
	}
	if r2.Available != 20 || r2.Reserved != 0 {
		t.Fatalf("v-admin-2 got available=%d reserved=%d, want available=20 reserved=0", r2.Available, r2.Reserved)
	}

	// 6. Query only unknown variants -> empty data array
	reqNone := httptest.NewRequest(http.MethodGet, "/api/v1/admin/commerce/stock?variant_ids=unknown-x,unknown-y", nil)
	recNone := httptest.NewRecorder()
	h.listStockAdmin(recNone, reqNone)
	if recNone.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recNone.Code, recNone.Body.String())
	}
	var envNone struct {
		Data []any `json:"data"`
	}
	if err := json.Unmarshal(recNone.Body.Bytes(), &envNone); err != nil {
		t.Fatalf("decode response: %v (%s)", err, recNone.Body.String())
	}
	if len(envNone.Data) != 0 {
		t.Fatalf("expected empty array, got %d items", len(envNone.Data))
	}
}

func TestCancelOrderFailsPendingOrderReleasesStockAndRejectsPaid(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-cancel", 3); err != nil {
		t.Fatal(err)
	}
	place := func(anon, key string) commerce.Order {
		cart, err := h.Service.GetOrCreateCart(ctx, db, "", anon, "TRY")
		if err != nil {
			t.Fatal(err)
		}
		if err := addPriced(h.Service, ctx, db, cart, "variant-cancel", 1, 1500); err != nil {
			t.Fatal(err)
		}
		order, err := h.Service.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "g@example.com", IdempotencyKey: key})
		if err != nil {
			t.Fatal(err)
		}
		return order
	}
	cancel := func(id string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/orders/"+id+"/cancel", nil)
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		h.cancelOrder(rec, req)
		return rec.Code
	}

	pending := place("anon-cancel-1", "cancel-idem-1")
	if code := cancel(pending.ID); code != http.StatusOK {
		t.Fatalf("cancel pending: got %d", code)
	}
	if got, _ := h.Service.Order(ctx, pending.ID); got.PaymentStatus != commerce.PaymentFailed {
		t.Fatalf("expected failed, got %s", got.PaymentStatus)
	}
	if st, _ := h.Service.Stock(ctx, "variant-cancel"); st.Reserved != 0 || st.Available != 3 {
		t.Fatalf("stock not released: %+v", st)
	}

	if code := cancel(pending.ID); code != http.StatusOK {
		t.Fatalf("second cancel must be idempotent, got %d", code)
	}

	paid := place("anon-cancel-2", "cancel-idem-2")
	if _, err := pay(h.Service, ctx, "mock-provider", "evt-cancel", paid.ID); err != nil {
		t.Fatal(err)
	}
	if code := cancel(paid.ID); code != http.StatusConflict {
		t.Fatalf("cancel paid: expected 409, got %d", code)
	}
	if code := cancel("nope"); code != http.StatusNotFound {
		t.Fatalf("cancel unknown: expected 404, got %d", code)
	}
}

func TestAdminDiscountCodesListAndToggle(t *testing.T) {
	h, _ := newTestHandler(t)
	create := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/discount-codes", strings.NewReader(`{"code":"YAZ10","kind":"percent","value":10}`))
	rec := httptest.NewRecorder()
	h.createDiscountCode(rec, create)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var created commerce.DiscountCode
	decodeBody(t, rec, &created)

	rec = httptest.NewRecorder()
	h.listDiscountCodes(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/commerce/discount-codes", nil))
	var listed []commerce.DiscountCode
	decodeBody(t, rec, &listed)
	if len(listed) != 1 || listed[0].Code != "YAZ10" || !listed[0].Active {
		t.Fatalf("unexpected list: %+v", listed)
	}

	toggle := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"active":false}`))
	toggle.SetPathValue("id", created.ID)
	rec = httptest.NewRecorder()
	h.setDiscountCodeActive(rec, toggle)
	if rec.Code != http.StatusOK {
		t.Fatalf("toggle: %d %s", rec.Code, rec.Body)
	}
	rec = httptest.NewRecorder()
	h.listDiscountCodes(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	decodeBody(t, rec, &listed)
	if listed[0].Active {
		t.Fatal("code should be inactive")
	}

	missing := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"active":true}`))
	missing.SetPathValue("id", "disc_nope")
	rec = httptest.NewRecorder()
	h.setDiscountCodeActive(rec, missing)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
}

func TestCancelOrderReturnsTheCouponRedemption(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	if err := h.Service.CreateDiscountCode(ctx, commerce.DiscountCode{ID: "disc-deneme", Code: "DENEME10", Kind: commerce.DiscountPercent, Value: 10, MaxRedemptions: 1, Active: true}); err != nil {
		t.Fatal(err)
	}
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-deneme", 3); err != nil {
		t.Fatal(err)
	}
	cart, err := h.Service.GetOrCreateCart(ctx, db, "", "anon-deneme", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := addPriced(h.Service, ctx, db, cart, "variant-deneme", 1, 10000); err != nil {
		t.Fatal(err)
	}
	order, err := h.Service.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "g@example.com", IdempotencyKey: "deneme-idem", DiscountCode: "DENEME10"})
	if err != nil {
		t.Fatal(err)
	}
	if order.DiscountCents != 1000 || order.TotalCents != 9000 {
		t.Fatalf("10%% discount not applied: %+v", order)
	}
	redeemed := func() int {
		codes, err := h.Service.ListDiscountCodes(ctx, 10, 0)
		if err != nil || len(codes) != 1 {
			t.Fatalf("list: %v %v", codes, err)
		}
		return codes[0].RedeemedCount
	}
	if got := redeemed(); got != 1 {
		t.Fatalf("after checkout redeemed_count = %d, want 1", got)
	}
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.SetPathValue("id", order.ID)
	rec := httptest.NewRecorder()
	h.cancelOrder(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("cancel: %d", rec.Code)
	}
	if got := redeemed(); got != 0 {
		t.Fatalf("after cancel redeemed_count = %d, want 0", got)
	}
	// second cancel (other tab) must not release it twice / go negative
	rec = httptest.NewRecorder()
	h.cancelOrder(rec, req)
	if got := redeemed(); rec.Code != http.StatusOK || got != 0 {
		t.Fatalf("second cancel: code %d redeemed %d", rec.Code, got)
	}
}

func TestLowStockListsOnlyScarceVariantsLowestFirst(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	for id, n := range map[string]int{"v-plenty": 50, "v-two": 2, "v-zero": 0} {
		if err := h.Service.Repository.EnsureStock(ctx, db, id, n); err != nil {
			t.Fatal(err)
		}
	}
	rec := httptest.NewRecorder()
	h.listLowStock(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/commerce/stock/low?threshold=3", nil))
	var got []variantStockView
	decodeBody(t, rec, &got)
	if len(got) != 2 || got[0].VariantID != "v-zero" || got[1].VariantID != "v-two" {
		t.Fatalf("unexpected low stock: %+v", got)
	}
}

func TestDiscountCodeNeedsAMemberWhenConfigured(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	h.Service.DiscountRequiresMember = true
	if err := h.Service.CreateDiscountCode(ctx, commerce.DiscountCode{ID: "disc-m", Code: "UYE10", Kind: commerce.DiscountPercent, Value: 10, Active: true}); err != nil {
		t.Fatal(err)
	}
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-m", 5); err != nil {
		t.Fatal(err)
	}
	place := func(memberID, anon, key string) error {
		var cart commerce.Cart
		var err error
		cart, err = h.Service.GetOrCreateCart(ctx, db, memberID, anon, "TRY")
		if err != nil {
			t.Fatal(err)
		}
		if err := addPriced(h.Service, ctx, db, cart, "variant-m", 1, 10000); err != nil {
			t.Fatal(err)
		}
		_, err = h.Service.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, MemberID: memberID, GuestEmail: "g@example.com", IdempotencyKey: key, DiscountCode: "UYE10"})
		return err
	}
	if err := place("", "anon-m", "m-guest"); !errors.Is(err, commerce.ErrDiscountMemberRequired) {
		t.Fatalf("guest with a code: %v, want ErrDiscountMemberRequired", err)
	}
	if err := place("member-1", "", "m-member"); err != nil {
		t.Fatalf("member with a code: %v", err)
	}
}

type fakeFullShippingProvider struct {
	fakeShippingProvider
	offerDraft  commerce.ShipmentOfferDraft
	lastInput   commerce.ShipmentInput
	cancelledID string
}

func (f *fakeFullShippingProvider) CreateShipment(ctx context.Context, _ commerce.Order, _ []commerce.OrderItem) (commerce.ShipmentResult, error) {
	f.lastInput = commerce.ShipmentInputFrom(ctx)
	return f.result, nil
}

func (f *fakeFullShippingProvider) ListShipmentOffers(ctx context.Context, _ commerce.Order, _ []commerce.OrderItem) (commerce.ShipmentOfferDraft, error) {
	f.lastInput = commerce.ShipmentInputFrom(ctx)
	return f.offerDraft, nil
}

func (f *fakeFullShippingProvider) CancelOrderShipment(_ context.Context, order commerce.Order) error {
	f.cancelledID = order.ShippingShipmentID
	return nil
}

func (f *fakeFullShippingProvider) DefaultCarrier() string { return "geliver" }
func (f *fakeFullShippingProvider) CarrierInfos() []commerce.CarrierInfo {
	return []commerce.CarrierInfo{{Key: "geliver", SupportsOffers: true, SupportsCancel: true}}
}

func TestShippingCarriersOffersAndCancelEndpoints(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()
	if err := h.Service.Repository.EnsureStock(ctx, db, "variant-ship-full", 5); err != nil {
		t.Fatal(err)
	}
	cart, err := h.Service.GetOrCreateCart(ctx, db, "", "anon-ship-full", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := addPriced(h.Service, ctx, db, cart, "variant-ship-full", 1, 2500); err != nil {
		t.Fatal(err)
	}
	order, err := h.Service.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "guest@example.com", IdempotencyKey: "ship-full-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pay(h.Service, ctx, "mock-provider", "evt-ship-full-1", order.ID); err != nil {
		t.Fatal(err)
	}

	provider := &fakeFullShippingProvider{
		fakeShippingProvider: fakeShippingProvider{
			name: "geliver",
			result: commerce.ShipmentResult{
				Carrier:        "Yurtiçi Kargo",
				TrackingNumber: "TRK-GEL-1",
				TrackingURL:    "https://track.example/TRK-GEL-1",
				ShipmentID:     "shp_gel_1",
				LabelURL:       "https://labels.example/shp_gel_1.pdf",
			},
		},
		offerDraft: commerce.ShipmentOfferDraft{
			ShipmentID: "shp_gel_1",
			CheapestID: "off_1",
			Offers:     []commerce.ShipmentOffer{{ID: "off_1", Carrier: "YURTICI", CarrierName: "Yurtiçi Kargo", Amount: "99.00", Currency: "TRY"}},
		},
	}
	h.ShippingProvider = provider

	// 1. GET /carriers
	rec := httptest.NewRecorder()
	h.getShippingCarriers(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/commerce/shipping/carriers", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("getShippingCarriers = %d %s", rec.Code, rec.Body.String())
	}
	var carriersPayload struct {
		Configured     bool                   `json:"configured"`
		DefaultCarrier string                 `json:"default_carrier"`
		Carriers       []commerce.CarrierInfo `json:"carriers"`
	}
	decodeBody(t, rec, &carriersPayload)
	if !carriersPayload.Configured || carriersPayload.DefaultCarrier != "geliver" || len(carriersPayload.Carriers) != 1 || !carriersPayload.Carriers[0].SupportsOffers {
		t.Fatalf("unexpected carriers payload: %+v", carriersPayload)
	}

	// 2. POST /shipment/offers
	offerReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/orders/"+order.ID+"/shipment/offers", strings.NewReader(`{"carrier":"geliver","parcel":{"weight_kg":"2.5"}}`))
	offerReq.SetPathValue("id", order.ID)
	offerRec := httptest.NewRecorder()
	h.listShipmentOffers(offerRec, offerReq)
	if offerRec.Code != http.StatusOK {
		t.Fatalf("listShipmentOffers = %d %s", offerRec.Code, offerRec.Body.String())
	}
	var draft commerce.ShipmentOfferDraft
	decodeBody(t, offerRec, &draft)
	if draft.ShipmentID != "shp_gel_1" || provider.lastInput.Parcel.WeightKg != "2.5" {
		t.Fatalf("unexpected offers response/input: %+v, %+v", draft, provider.lastInput)
	}

	// 3. POST /shipment with selected offer_id
	shipReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/orders/"+order.ID+"/shipment", strings.NewReader(`{"carrier":"geliver","offer_id":"off_1","shipment_id":"shp_gel_1"}`))
	shipReq.SetPathValue("id", order.ID)
	shipRec := httptest.NewRecorder()
	h.createShipment(shipRec, shipReq)
	if shipRec.Code != http.StatusOK {
		t.Fatalf("createShipment = %d %s", shipRec.Code, shipRec.Body.String())
	}
	var shipped commerce.Order
	decodeBody(t, shipRec, &shipped)
	if shipped.ShippingShipmentID != "shp_gel_1" || shipped.ShippingLabelURL != "https://labels.example/shp_gel_1.pdf" || provider.lastInput.OfferID != "off_1" {
		t.Fatalf("unexpected shipped order: %+v", shipped)
	}

	// 4. POST /shipment/cancel
	cancelReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/orders/"+order.ID+"/shipment/cancel", nil)
	cancelReq.SetPathValue("id", order.ID)
	cancelRec := httptest.NewRecorder()
	h.cancelShipment(cancelRec, cancelReq)
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("cancelShipment = %d %s", cancelRec.Code, cancelRec.Body.String())
	}
	var afterCancel commerce.Order
	decodeBody(t, cancelRec, &afterCancel)
	if afterCancel.FulfillmentStatus != commerce.FulfillmentUnfulfilled || provider.cancelledID != "shp_gel_1" {
		t.Fatalf("unexpected order after cancel: %+v, cancelledID=%q", afterCancel, provider.cancelledID)
	}
}
