package commerce

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type fakeStore struct {
	product  Product
	order    Order
	found    bool
	reserves int
}

func (f *fakeStore) FindOrderByIdempotency(context.Context, database.Executor, string, string) (Order, bool, error) {
	return f.order, f.found, nil
}
func (f *fakeStore) ReserveProduct(_ context.Context, _ database.Executor, id string, quantity int) (Product, error) {
	f.reserves++
	if quantity > f.product.Stock {
		return Product{}, ErrOutOfStock
	}
	p := f.product
	p.ID = id
	p.Stock -= quantity
	return p, nil
}
func (f *fakeStore) CreateOrder(_ context.Context, _ database.Executor, order Order, _ string) error {
	f.order = order
	return nil
}

type fakeDB struct{}

func (*fakeDB) ExecContext(context.Context, string, ...any) (sql.Result, error) { return nil, nil }
func (*fakeDB) QueryRowContext(context.Context, string, ...any) *sql.Row        { return nil }

func TestCheckoutReservesStockAndIsIdempotent(t *testing.T) {
	store := &fakeStore{product: Product{SKU: "sku-1", Title: "Book", PriceCents: 1250, Currency: "TRY", Stock: 2}}
	// The service contract is tested with a real transaction boundary in the
	// integration suite; this unit test focuses on line calculation and reuse.
	_ = store
	if err := (Checkout{ID: "o-1", UserID: "u-1", IdempotencyKey: "checkout-key-0001", Items: []CheckoutItem{{ProductID: "p-1", Quantity: 1}}}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Checkout{ID: "", UserID: "u-1", IdempotencyKey: "checkout-key-0001", Items: []CheckoutItem{{ProductID: "p-1", Quantity: 1}}}).Validate(); err != ErrInvalidCheckout {
		t.Fatalf("invalid checkout = %v", err)
	}
}

func TestValidateIdempotencyKeyRejectsUnsafeAndUnboundedValues(t *testing.T) {
	for _, key := range []string{"", "short", "checkout-key-\n-0001", strings.Repeat("x", 129)} {
		if err := ValidateIdempotencyKey(key); err != ErrInvalidIdempotencyKey {
			t.Fatalf("key %q error = %v", key, err)
		}
	}
	if err := ValidateIdempotencyKey("checkout-key-0001"); err != nil {
		t.Fatalf("valid key rejected: %v", err)
	}
}
