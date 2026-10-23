package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

func newTestDB(t *testing.T) database.Executor {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "commerce.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	return db
}

// TestReserveStockIsAtomicUnderConcurrentLastUnit is the core proof this
// module exists for: two requests racing for the last unit must never both
// succeed, and stock must never go negative. Already proven once at the raw
// SQL level before any Go code existed; this proves the Go repository
// preserves that guarantee.
func TestReserveStockIsAtomicUnderConcurrentLastUnit(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}

	if err := store.EnsureStock(ctx, db, "variant-1", 1); err != nil {
		t.Fatal(err)
	}

	results := make(chan bool, 2)
	race := func() {
		ok, err := store.ReserveStock(ctx, db, "variant-1", 1)
		if err != nil {
			t.Error(err)
			results <- false
			return
		}
		results <- ok
	}
	go race()
	go race()

	first, second := <-results, <-results
	if first == second {
		t.Fatalf("exactly one reservation should succeed, got first=%v second=%v", first, second)
	}

	stock, err := store.GetStock(ctx, db, "variant-1")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Available != 0 || stock.Reserved != 1 {
		t.Fatalf("stock after race = %+v, want available=0 reserved=1", stock)
	}
}

func TestReserveStockFailsWhenNotEnoughAvailable(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}

	if err := store.EnsureStock(ctx, db, "variant-2", 2); err != nil {
		t.Fatal(err)
	}
	ok, err := store.ReserveStock(ctx, db, "variant-2", 5)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("reserving more than available should fail")
	}
	stock, err := store.GetStock(ctx, db, "variant-2")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Available != 2 || stock.Reserved != 0 {
		t.Fatalf("stock should be untouched by a failed reservation, got %+v", stock)
	}
}

func TestReleaseReservationReturnsStockToAvailable(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}

	if err := store.EnsureStock(ctx, db, "variant-3", 5); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.ReserveStock(ctx, db, "variant-3", 3); err != nil || !ok {
		t.Fatalf("reserve: ok=%v err=%v", ok, err)
	}
	if err := store.ReleaseReservation(ctx, db, "variant-3", 3); err != nil {
		t.Fatal(err)
	}
	stock, err := store.GetStock(ctx, db, "variant-3")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Available != 5 || stock.Reserved != 0 {
		t.Fatalf("stock after release = %+v, want available=5 reserved=0", stock)
	}
}

func TestCartIsScopedToExactlyOneOwner(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}
	now := time.Now().UTC()

	memberCart := commerce.Cart{ID: "cart-member-1", MemberID: "member-1", Currency: "TRY", CreatedAt: now, UpdatedAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := store.CreateCart(ctx, db, memberCart); err != nil {
		t.Fatal(err)
	}
	anonCart := commerce.Cart{ID: "cart-anon-1", AnonymousToken: "anon-token-1", Currency: "TRY", CreatedAt: now, UpdatedAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := store.CreateCart(ctx, db, anonCart); err != nil {
		t.Fatal(err)
	}

	got, found, err := store.GetCartByMember(ctx, db, "member-1")
	if err != nil || !found || got.ID != "cart-member-1" {
		t.Fatalf("GetCartByMember = %+v/%v/%v", got, found, err)
	}
	got, found, err = store.GetCartByAnonymousToken(ctx, db, "anon-token-1")
	if err != nil || !found || got.ID != "cart-anon-1" {
		t.Fatalf("GetCartByAnonymousToken = %+v/%v/%v", got, found, err)
	}
	_, found, err = store.GetCartByMember(ctx, db, "no-such-member")
	if err != nil || found {
		t.Fatalf("unknown member should not be found: found=%v err=%v", found, err)
	}
}

func TestAddOrUpdateItemUpsertsByCartAndVariant(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}
	now := time.Now().UTC()

	cart := commerce.Cart{ID: "cart-1", AnonymousToken: "tok-1", Currency: "TRY", CreatedAt: now, UpdatedAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := store.CreateCart(ctx, db, cart); err != nil {
		t.Fatal(err)
	}

	item := commerce.CartItem{ID: "item-1", CartID: "cart-1", VariantID: "variant-1", Quantity: 1, UnitPriceCents: 1000, CreatedAt: now, UpdatedAt: now}
	if err := store.AddOrUpdateItem(ctx, db, item); err != nil {
		t.Fatal(err)
	}
	item.Quantity = 3
	if err := store.AddOrUpdateItem(ctx, db, item); err != nil {
		t.Fatal(err)
	}

	items, err := store.ListItems(ctx, db, "cart-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Quantity != 3 {
		t.Fatalf("expected one upserted item with quantity 3, got %+v", items)
	}
}

// TestCheckoutIsIdempotent is the second core proof: a retried checkout
// request (same tenant, same idempotency key -- a real client retry after a
// dropped response, or a double click) must produce exactly one order, not
// two, and must not double-charge.
func TestCheckoutIsIdempotent(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}
	now := time.Now().UTC()

	order := commerce.Order{
		ID: "order-1", GuestEmail: "guest@example.com", IdempotencyKey: "idem-1", IdempotencyHash: "hash-1",
		Currency: "TRY", TotalCents: 5000, PaymentStatus: commerce.PaymentPending, FulfillmentStatus: commerce.FulfillmentUnfulfilled,
		CreatedAt: now, UpdatedAt: now,
	}
	items := []commerce.OrderItem{{ID: "oi-1", OrderID: "order-1", VariantID: "variant-1", SKU: "SKU-1", Title: "Deri Kemer", Quantity: 1, UnitPriceCents: 5000, TotalCents: 5000}}

	created, isNew, err := store.CreateOrder(ctx, db, order, items)
	if err != nil || !isNew || created.ID != "order-1" {
		t.Fatalf("first checkout: created=%+v isNew=%v err=%v", created, isNew, err)
	}

	retry := order
	retry.ID = "order-2" // a naive client might even mint a new id on retry
	retryCreated, isNew, err := store.CreateOrder(ctx, db, retry, items)
	if err != nil {
		t.Fatal(err)
	}
	if isNew {
		t.Fatal("retried checkout with the same idempotency key must not create a second order")
	}
	if retryCreated.ID != "order-1" {
		t.Fatalf("retried checkout should return the original order, got %q", retryCreated.ID)
	}
}

func TestPaymentNotificationIsRecordedOnceOnly(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}

	first, err := store.RecordPaymentNotification(ctx, db, "paytr", "event-1")
	if err != nil || !first {
		t.Fatalf("first delivery: recorded=%v err=%v", first, err)
	}
	second, err := store.RecordPaymentNotification(ctx, db, "paytr", "event-1")
	if err != nil {
		t.Fatal(err)
	}
	if second {
		t.Fatal("a redelivered webhook event must not be recorded twice")
	}
}

func TestUpdatePaymentStatusOnlyAppliesFromThePendingState(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}
	now := time.Now().UTC()

	order := commerce.Order{
		ID: "order-pay-1", GuestEmail: "guest@example.com", IdempotencyKey: "idem-pay-1", IdempotencyHash: "hash-pay-1",
		Currency: "TRY", TotalCents: 1000, PaymentStatus: commerce.PaymentPending, FulfillmentStatus: commerce.FulfillmentUnfulfilled,
		CreatedAt: now, UpdatedAt: now,
	}
	if _, _, err := store.CreateOrder(ctx, db, order, nil); err != nil {
		t.Fatal(err)
	}

	applied, err := store.UpdatePaymentStatus(ctx, db, "order-pay-1", commerce.PaymentPending, commerce.PaymentPaid)
	if err != nil || !applied {
		t.Fatalf("pending->paid: applied=%v err=%v", applied, err)
	}
	// A duplicate webhook retrying the same transition must not error, but
	// must report it did nothing -- the order is no longer pending.
	appliedAgain, err := store.UpdatePaymentStatus(ctx, db, "order-pay-1", commerce.PaymentPending, commerce.PaymentPaid)
	if err != nil {
		t.Fatal(err)
	}
	if appliedAgain {
		t.Fatal("a second attempt to apply the same transition should be a no-op, not re-apply")
	}
}

func TestCommitReservationOnlyReducesReservedNotAvailable(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}

	if err := store.EnsureStock(ctx, db, "variant-4", 5); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.ReserveStock(ctx, db, "variant-4", 2); err != nil || !ok {
		t.Fatalf("reserve: ok=%v err=%v", ok, err)
	}
	if err := store.CommitReservation(ctx, db, "variant-4", 2); err != nil {
		t.Fatal(err)
	}
	stock, err := store.GetStock(ctx, db, "variant-4")
	if err != nil {
		t.Fatal(err)
	}
	// available stays at 3 (already decremented at reserve time); reserved
	// drops to 0 because the sale is now final, not just held.
	if stock.Available != 3 || stock.Reserved != 0 {
		t.Fatalf("stock after commit = %+v, want available=3 reserved=0", stock)
	}
}

func TestListOrderItemsReturnsWhatWasCreated(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}
	now := time.Now().UTC()

	order := commerce.Order{
		ID: "order-items-1", GuestEmail: "guest@example.com", IdempotencyKey: "idem-items-1", IdempotencyHash: "h",
		Currency: "TRY", TotalCents: 2000, PaymentStatus: commerce.PaymentPending, FulfillmentStatus: commerce.FulfillmentUnfulfilled,
		CreatedAt: now, UpdatedAt: now,
	}
	items := []commerce.OrderItem{
		{ID: "oi-a", OrderID: "order-items-1", VariantID: "variant-a", SKU: "SKU-A", Title: "A", Quantity: 2, UnitPriceCents: 500, TotalCents: 1000},
		{ID: "oi-b", OrderID: "order-items-1", VariantID: "variant-b", SKU: "SKU-B", Title: "B", Quantity: 1, UnitPriceCents: 1000, TotalCents: 1000},
	}
	if _, _, err := store.CreateOrder(ctx, db, order, items); err != nil {
		t.Fatal(err)
	}

	got, err := store.ListOrderItems(ctx, db, "order-items-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 order items, got %d", len(got))
	}
	byVariant := map[string]commerce.OrderItem{}
	for _, item := range got {
		byVariant[item.VariantID] = item
	}
	if byVariant["variant-a"].Quantity != 2 || byVariant["variant-b"].Quantity != 1 {
		t.Fatalf("unexpected quantities: %+v", byVariant)
	}
}

func TestEnsureStockUpdatesAvailableWhenRowAlreadyExists(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}

	if err := store.EnsureStock(ctx, db, "variant-restock", 3); err != nil {
		t.Fatal(err)
	}
	// Re-running with a different count (an admin correcting a stock take)
	// must update available, not silently keep the old value.
	if err := store.EnsureStock(ctx, db, "variant-restock", 10); err != nil {
		t.Fatal(err)
	}
	stock, err := store.GetStock(ctx, db, "variant-restock")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Available != 10 {
		t.Fatalf("available = %d, want 10 after restock", stock.Available)
	}
}

func TestEnsureStockNeverOverwritesAnInFlightReservation(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}

	if err := store.EnsureStock(ctx, db, "variant-restock-2", 5); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.ReserveStock(ctx, db, "variant-restock-2", 2); err != nil || !ok {
		t.Fatalf("reserve: ok=%v err=%v", ok, err)
	}
	// An admin resetting available to 10 must not wipe out the 2 units
	// already reserved by a shopper mid-checkout.
	if err := store.EnsureStock(ctx, db, "variant-restock-2", 10); err != nil {
		t.Fatal(err)
	}
	stock, err := store.GetStock(ctx, db, "variant-restock-2")
	if err != nil {
		t.Fatal(err)
	}
	if stock.Available != 10 || stock.Reserved != 2 {
		t.Fatalf("stock = %+v, want available=10 reserved=2 (reservation preserved)", stock)
	}
}

func TestListPendingOrdersOlderThanFindsOnlyStalePendingOrders(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}
	old := time.Now().UTC().Add(-time.Hour)
	recent := time.Now().UTC()

	stale := commerce.Order{ID: "order-stale", GuestEmail: "a@example.com", IdempotencyKey: "idem-stale", IdempotencyHash: "h", Currency: "TRY", PaymentStatus: commerce.PaymentPending, FulfillmentStatus: commerce.FulfillmentUnfulfilled, CreatedAt: old, UpdatedAt: old}
	if _, _, err := store.CreateOrder(ctx, db, stale, nil); err != nil {
		t.Fatal(err)
	}
	fresh := commerce.Order{ID: "order-fresh", GuestEmail: "b@example.com", IdempotencyKey: "idem-fresh", IdempotencyHash: "h", Currency: "TRY", PaymentStatus: commerce.PaymentPending, FulfillmentStatus: commerce.FulfillmentUnfulfilled, CreatedAt: recent, UpdatedAt: recent}
	if _, _, err := store.CreateOrder(ctx, db, fresh, nil); err != nil {
		t.Fatal(err)
	}
	paidOld := commerce.Order{ID: "order-paid-old", GuestEmail: "c@example.com", IdempotencyKey: "idem-paid-old", IdempotencyHash: "h", Currency: "TRY", PaymentStatus: commerce.PaymentPaid, FulfillmentStatus: commerce.FulfillmentUnfulfilled, CreatedAt: old, UpdatedAt: old}
	if _, _, err := store.CreateOrder(ctx, db, paidOld, nil); err != nil {
		t.Fatal(err)
	}

	cutoff := time.Now().UTC().Add(-30 * time.Minute)
	stalePending, err := store.ListPendingOrdersOlderThan(ctx, db, cutoff, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(stalePending) != 1 || stalePending[0].ID != "order-stale" {
		t.Fatalf("expected only order-stale, got %+v", stalePending)
	}
}

func TestListOrdersByMemberIDReturnsOnlyThatMembersOrdersNewestFirst(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}
	older := time.Now().UTC().Add(-time.Hour)
	newer := time.Now().UTC()

	first := commerce.Order{ID: "order-member-1", MemberID: "member-a", IdempotencyKey: "idem-member-1", IdempotencyHash: "h", Currency: "TRY", PaymentStatus: commerce.PaymentPending, FulfillmentStatus: commerce.FulfillmentUnfulfilled, CreatedAt: older, UpdatedAt: older}
	if _, _, err := store.CreateOrder(ctx, db, first, nil); err != nil {
		t.Fatal(err)
	}
	second := commerce.Order{ID: "order-member-2", MemberID: "member-a", IdempotencyKey: "idem-member-2", IdempotencyHash: "h", Currency: "TRY", PaymentStatus: commerce.PaymentPaid, FulfillmentStatus: commerce.FulfillmentUnfulfilled, CreatedAt: newer, UpdatedAt: newer}
	if _, _, err := store.CreateOrder(ctx, db, second, nil); err != nil {
		t.Fatal(err)
	}
	other := commerce.Order{ID: "order-member-other", MemberID: "member-b", IdempotencyKey: "idem-member-other", IdempotencyHash: "h", Currency: "TRY", PaymentStatus: commerce.PaymentPending, FulfillmentStatus: commerce.FulfillmentUnfulfilled, CreatedAt: newer, UpdatedAt: newer}
	if _, _, err := store.CreateOrder(ctx, db, other, nil); err != nil {
		t.Fatal(err)
	}

	orders, err := store.ListOrdersByMemberID(ctx, db, "member-a", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 2 || orders[0].ID != "order-member-2" || orders[1].ID != "order-member-1" {
		t.Fatalf("expected [order-member-2, order-member-1] newest first, got %+v", orders)
	}
}

func TestGetOrderByMemberIDRejectsAnotherMembersOrder(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	store := Store{Dialect: database.SQLite}

	owned := commerce.Order{ID: "order-owned", MemberID: "member-a", IdempotencyKey: "idem-owned", IdempotencyHash: "h", Currency: "TRY", PaymentStatus: commerce.PaymentPending, FulfillmentStatus: commerce.FulfillmentUnfulfilled, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if _, _, err := store.CreateOrder(ctx, db, owned, nil); err != nil {
		t.Fatal(err)
	}

	if _, err := store.GetOrderByMemberID(ctx, db, "member-a", "order-owned"); err != nil {
		t.Fatalf("expected the owning member to read their own order, got %v", err)
	}
	if _, err := store.GetOrderByMemberID(ctx, db, "member-b", "order-owned"); !errors.Is(err, commerce.ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound for a non-owning member, got %v", err)
	}
}
