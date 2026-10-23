package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
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
