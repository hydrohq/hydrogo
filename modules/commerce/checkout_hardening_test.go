package commerce_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/migrations"
	"github.com/hydrohq/hydrogo/modules/commerce"
	"github.com/hydrohq/hydrogo/modules/commerce/repository/sqlstore"
)

type stockCall struct {
	VariantID string
	Quantity  int
}

type recordingStore struct {
	sqlstore.Store
	calls       []stockCall
	mockItems   []commerce.CartItem
	mockItemsOn bool
}

func (r *recordingStore) ReserveStock(ctx context.Context, db database.Executor, variantID string, quantity int) (bool, error) {
	r.calls = append(r.calls, stockCall{VariantID: variantID, Quantity: quantity})
	return r.Store.ReserveStock(ctx, db, variantID, quantity)
}

func (r *recordingStore) ListItems(ctx context.Context, db database.Executor, cartID string) ([]commerce.CartItem, error) {
	if r.mockItemsOn {
		return r.mockItems, nil
	}
	return r.Store.ListItems(ctx, db, cartID)
}

func (r *recordingStore) ClaimCart(ctx context.Context, db database.Executor, cartID string) (int64, error) {
	if r.mockItemsOn {
		return int64(len(r.mockItems)), nil
	}
	return r.Store.ClaimCart(ctx, db, cartID)
}

func TestCheckoutDeterministicReserveStockOrderAndMerging(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "commerce-stock-ordering.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	recStore := &recordingStore{Store: sqlstore.Store{Dialect: database.SQLite}}
	svc := commerce.Service{
		Transactions: database.NewTransactor(db),
		Repository:   recStore,
		Prices: commerce.StaticPrices{
			"variant-z": commerce.Price{Cents: 100, Currency: "TRY"},
			"variant-a": commerce.Price{Cents: 200, Currency: "TRY"},
			"variant-m": commerce.Price{Cents: 300, Currency: "TRY"},
		},
	}

	// Ensure stock
	for _, v := range []string{"variant-z", "variant-a", "variant-m"} {
		if err := recStore.Store.EnsureStock(ctx, db, v, 20); err != nil {
			t.Fatal(err)
		}
	}

	// Setup cart with items out of order: variant-z, variant-a, variant-m, and a duplicate variant-a
	recStore.mockItems = []commerce.CartItem{
		{ID: "ci-1", CartID: "cart-mock", VariantID: "variant-z", Quantity: 2},
		{ID: "ci-2", CartID: "cart-mock", VariantID: "variant-a", Quantity: 3},
		{ID: "ci-3", CartID: "cart-mock", VariantID: "variant-m", Quantity: 1},
		{ID: "ci-4", CartID: "cart-mock", VariantID: "variant-a", Quantity: 4}, // duplicate variant-a
	}
	recStore.mockItemsOn = true

	_, err = svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID:         "cart-mock",
		GuestEmail:     "shopper@example.com",
		IdempotencyKey: "idem-ordering-1",
	})
	if err != nil {
		t.Fatalf("checkout failed: %v", err)
	}

	// Expected calls:
	// 1. "variant-a" with merged quantity 3 + 4 = 7
	// 2. "variant-m" with quantity 1
	// 3. "variant-z" with quantity 2
	expectedCalls := []stockCall{
		{VariantID: "variant-a", Quantity: 7},
		{VariantID: "variant-m", Quantity: 1},
		{VariantID: "variant-z", Quantity: 2},
	}

	if !reflect.DeepEqual(recStore.calls, expectedCalls) {
		t.Fatalf("calls = %+v, want %+v", recStore.calls, expectedCalls)
	}
}

func TestCheckoutRejectsLineQuantityAboveMaximum(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "commerce-max-line.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	recStore := &recordingStore{Store: sqlstore.Store{Dialect: database.SQLite}}
	svc := commerce.Service{
		Transactions: database.NewTransactor(db),
		Repository:   recStore,
		Prices:       commerce.StaticPrices{"variant-a": commerce.Price{Cents: 100, Currency: "TRY"}},
	}
	if err := recStore.Store.EnsureStock(ctx, db, "variant-a", 500); err != nil {
		t.Fatal(err)
	}
	recStore.mockItemsOn = true

	cases := []struct {
		name     string
		items    []commerce.CartItem
		max      int
		wantFail bool
	}{
		{"default allows 100", []commerce.CartItem{{ID: "1", CartID: "c", VariantID: "variant-a", Quantity: 100}}, 0, false},
		{"default rejects 101", []commerce.CartItem{{ID: "1", CartID: "c", VariantID: "variant-a", Quantity: 101}}, 0, true},
		{"configured limit allows equal", []commerce.CartItem{{ID: "1", CartID: "c", VariantID: "variant-a", Quantity: 5}}, 5, false},
		{"configured limit rejects above", []commerce.CartItem{{ID: "1", CartID: "c", VariantID: "variant-a", Quantity: 6}}, 5, true},
		{"duplicate lines are summed", []commerce.CartItem{
			{ID: "1", CartID: "c", VariantID: "variant-a", Quantity: 3},
			{ID: "2", CartID: "c", VariantID: "variant-a", Quantity: 3},
		}, 5, true},
	}
	for i, tc := range cases {
		recStore.mockItems = tc.items
		_, err := svc.Checkout(ctx, commerce.CheckoutRequest{
			CartID: "c", GuestEmail: "shopper@example.com",
			IdempotencyKey: fmt.Sprintf("idem-max-line-%d", i), MaxLineQuantity: tc.max,
		})
		if tc.wantFail && !errors.Is(err, commerce.ErrLineQuantityExceeded) {
			t.Errorf("%s: err = %v, want ErrLineQuantityExceeded", tc.name, err)
		}
		if !tc.wantFail && err != nil {
			t.Errorf("%s: unexpected error %v", tc.name, err)
		}
	}
}
