package transport_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
	commercestore "github.com/ardaozyildirim/hydrogo/modules/commerce/repository/sqlstore"
	commercetransport "github.com/ardaozyildirim/hydrogo/modules/commerce/transport"
)

func setupTestCommerceHandler(t *testing.T) (commercetransport.Handler, commerce.Service, *sql.DB) {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test_abandoned.sqlite")
	db, err := database.Open(ctx, database.SQLite, dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	store := commercestore.Store{Dialect: database.SQLite}
	if err := store.EnsureStock(ctx, db, "var-item-1", 100); err != nil {
		t.Fatalf("ensure stock: %v", err)
	}

	svc := commerce.Service{
		Transactions: database.NewTransactor(db),
		Repository:   store,
		Prices: commerce.StaticPrices{
			"var-item-1": commerce.Price{Cents: 1000, Currency: "TRY"},
		},
	}

	tracker := commercetransport.NewPendingOrderTracker()
	h := commercetransport.Handler{
		Service:               svc,
		MaxPendingOrdersPerIP: 25,
		PendingTracker:        tracker,
	}
	return h, svc, db
}

func performCheckout(h commercetransport.Handler, ip string, idemKey string) *httptest.ResponseRecorder {
	ctx := context.Background()
	shopper := commerce.Shopper{AnonymousToken: "anon-" + idemKey, Currency: "TRY"}
	_, _, err := h.Service.AddToCart(ctx, shopper, "var-item-1", 1)
	if err != nil {
		panic(err)
	}

	body := fmt.Sprintf(`{"guest_email":"test@example.com","idempotency_key":%q}`, idemKey)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":12345"
	req.AddCookie(&http.Cookie{Name: "hydro_commerce_cart", Value: "anon-" + idemKey})

	rec := httptest.NewRecorder()
	for _, route := range h.Routes() {
		if route.Pattern == "POST /api/v1/commerce/checkout" {
			route.Handler.ServeHTTP(rec, req)
			break
		}
	}
	return rec
}

func TestAbandonedCheckoutMaxPendingOrdersPerIP(t *testing.T) {
	h, svc, _ := setupTestCommerceHandler(t)
	const clientIP = "198.51.100.1"

	var orderIDs []string
	// Perform 25 checkouts from the same IP (within 30 mins) -> all should succeed.
	for i := 1; i <= 25; i++ {
		rec := performCheckout(h, clientIP, fmt.Sprintf("idem-key-%d", i))
		if rec.Code != http.StatusCreated {
			t.Fatalf("checkout %d failed: expected 201 Created, got %d (body: %s)", i, rec.Code, rec.Body.String())
		}
		// Capture order ID from response
		bodyStr := rec.Body.String()
		idx := strings.Index(bodyStr, `"id":"`)
		if idx != -1 {
			id := bodyStr[idx+6 : idx+6+strings.Index(bodyStr[idx+6:], `"`)]
			orderIDs = append(orderIDs, id)
		}
	}

	// 26th checkout from the same IP must be rejected with 429 Too Many Requests.
	rec26 := performCheckout(h, clientIP, "idem-key-26")
	if rec26.Code != http.StatusTooManyRequests {
		t.Fatalf("26th checkout expected 429 Too Many Requests, got %d (body: %s)", rec26.Code, rec26.Body.String())
	}
	if !strings.Contains(rec26.Body.String(), "too many") {
		t.Fatalf("expected rate limit response message, got: %s", rec26.Body.String())
	}

	// Another IP should NOT be blocked
	const otherIP = "198.51.100.2"
	recOther := performCheckout(h, otherIP, "idem-key-other-1")
	if recOther.Code != http.StatusCreated {
		t.Fatalf("checkout from different IP expected 201 Created, got %d", recOther.Code)
	}

	// Now pay for one of the 5 pending orders from clientIP
	if len(orderIDs) > 0 {
		ctx := context.Background()
		_, err := svc.ConfirmPayment(ctx, commerce.Payment{
			Provider:    "paytr",
			EventID:     orderIDs[0],
			OrderID:     orderIDs[0],
			AmountCents: 1000,
		})
		if err != nil {
			t.Fatalf("confirm payment: %v", err)
		}

		// Paying one order frees a slot for a new checkout.
		recAfterPay := performCheckout(h, clientIP, "idem-key-7")
		if recAfterPay.Code != http.StatusCreated {
			t.Fatalf("checkout after order paid expected 201 Created, got %d (body: %s)", recAfterPay.Code, recAfterPay.Body.String())
		}
	}
}

func TestCheckoutRateLimit10PerMinute(t *testing.T) {
	h, _, _ := setupTestCommerceHandler(t)
	// Explicitly wire 10 per minute attempt limiter
	h.CheckoutLimiter = security.NewAttemptLimiter(100, 10, time.Minute)
	const clientIP = "198.51.100.5"

	// 10 checkouts from clientIP: first 5 succeed with 201, next 5 hit pending order limit or rate limit (429)
	// But let's raise MaxPendingOrdersPerIP to 20 so we test the rate limiter in isolation!
	h.MaxPendingOrdersPerIP = 20

	for i := 1; i <= 10; i++ {
		rec := performCheckout(h, clientIP, fmt.Sprintf("rate-key-%d", i))
		if rec.Code != http.StatusCreated {
			t.Fatalf("checkout %d expected 201 Created, got %d", i, rec.Code)
		}
	}

	// 11th checkout must be rate limited with 429
	rec11 := performCheckout(h, clientIP, "rate-key-11")
	if rec11.Code != http.StatusTooManyRequests {
		t.Fatalf("11th checkout within minute expected 429 Too Many Requests, got %d", rec11.Code)
	}
}

func TestConcurrentCheckoutsFromOneIPNeverExceedPendingLimit(t *testing.T) {
	h, _, _ := setupTestCommerceHandler(t)
	var checkout http.Handler
	for _, route := range h.Routes() {
		if route.Pattern == "POST /api/v1/commerce/checkout" {
			checkout = route.Handler
		}
	}
	const attempts = 50
	ctx := context.Background()
	requests := make([]*http.Request, attempts)
	for i := range requests {
		key := fmt.Sprintf("race-%d", i)
		if _, _, err := h.Service.AddToCart(ctx, commerce.Shopper{AnonymousToken: "anon-" + key, Currency: "TRY"}, "var-item-1", 1); err != nil {
			t.Fatal(err)
		}
		body := fmt.Sprintf(`{"guest_email":"race@example.com","idempotency_key":%q}`, key)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.9:4000"
		req.AddCookie(&http.Cookie{Name: "hydro_commerce_cart", Value: "anon-" + key})
		requests[i] = req
	}
	var created atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, req := range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			rec := httptest.NewRecorder()
			checkout.ServeHTTP(rec, req)
			if rec.Code == http.StatusCreated {
				created.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if got := created.Load(); got != 25 {
		t.Fatalf("%d checkouts succeeded from one IP, want exactly 25", got)
	}
}
