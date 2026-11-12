package contract_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/migrations"
	"github.com/hydrohq/hydrogo/modules/commerce"
	commercestore "github.com/hydrohq/hydrogo/modules/commerce/repository/sqlstore"
)

func TestCommerceLastItem_SingleVariantConcurrency(t *testing.T) {
	if os.Getenv("HYDRO_TEST_POSTGRES_URL") == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := openIsolatedPostgres(t, ctx)
	if err := database.ApplyMigrations(ctx, db, database.Postgres, migrations.FS); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	const variantID = "variant-last-item"
	store := commercestore.Store{Dialect: database.Postgres}
	svc := commerce.Service{
		Transactions: database.NewTransactor(db),
		Repository:   store,
		Prices: commerce.StaticPrices{
			variantID: commerce.Price{Cents: 1500, Currency: "TRY"},
		},
	}

	if err := store.EnsureStock(ctx, db, variantID, 1); err != nil {
		t.Fatalf("ensure stock: %v", err)
	}

	const numGoroutines = 20
	carts := make([]commerce.Cart, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		shopper := commerce.Shopper{
			AnonymousToken: fmt.Sprintf("shopper-%d", i),
			Currency:       "TRY",
		}
		cart, _, err := svc.AddToCart(ctx, shopper, variantID, 1)
		if err != nil {
			t.Fatalf("add to cart %d: %v", i, err)
		}
		carts[i] = cart
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make([]error, numGoroutines)
	orders := make([]commerce.Order, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			order, err := svc.Checkout(ctx, commerce.CheckoutRequest{
				CartID:         carts[idx].ID,
				GuestEmail:     fmt.Sprintf("guest%d@example.com", idx),
				IdempotencyKey: fmt.Sprintf("idem-single-%d", idx),
				Currency:       "TRY",
			})
			results[idx] = err
			orders[idx] = order
		}(i)
	}

	close(start)
	wg.Wait()

	var successCount, outOfStockCount int
	for i, err := range results {
		if err == nil {
			successCount++
			if orders[i].ID == "" {
				t.Fatalf("expected order ID on successful checkout")
			}
		} else if errors.Is(err, commerce.ErrOutOfStock) {
			outOfStockCount++
		} else {
			t.Fatalf("goroutine %d unexpected error: %v", i, err)
		}
	}

	if successCount != 1 {
		t.Fatalf("successful checkouts = %d, want 1", successCount)
	}
	if outOfStockCount != 19 {
		t.Fatalf("out of stock errors = %d, want 19", outOfStockCount)
	}

	stock, err := store.GetStock(ctx, db, variantID)
	if err != nil {
		t.Fatalf("get stock: %v", err)
	}
	if stock.Available < 0 {
		t.Fatalf("available stock dropped below zero: %d", stock.Available)
	}
	if stock.Available != 0 {
		t.Fatalf("available stock = %d, want 0", stock.Available)
	}
	if stock.Reserved != 1 {
		t.Fatalf("reserved stock = %d, want 1", stock.Reserved)
	}
	if stock.Available+stock.Reserved != 1 {
		t.Fatalf("inconsistent stock sum: %d + %d != 1", stock.Available, stock.Reserved)
	}

	var orderCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM commerce_customer_orders").Scan(&orderCount); err != nil {
		t.Fatalf("query order count: %v", err)
	}
	if orderCount != 1 {
		t.Fatalf("persisted order count = %d, want 1", orderCount)
	}
}

func TestCommerceLastItem_TwoVariantsReverseOrderNoDeadlock(t *testing.T) {
	if os.Getenv("HYDRO_TEST_POSTGRES_URL") == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := openIsolatedPostgres(t, ctx)
	if err := database.ApplyMigrations(ctx, db, database.Postgres, migrations.FS); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	const (
		variantA = "variant-alpha"
		variantB = "variant-beta"
	)
	store := commercestore.Store{Dialect: database.Postgres}
	svc := commerce.Service{
		Transactions: database.NewTransactor(db),
		Repository:   store,
		Prices: commerce.StaticPrices{
			variantA: commerce.Price{Cents: 1000, Currency: "TRY"},
			variantB: commerce.Price{Cents: 2000, Currency: "TRY"},
		},
	}

	const initialStock = 10
	if err := store.EnsureStock(ctx, db, variantA, initialStock); err != nil {
		t.Fatalf("ensure stock A: %v", err)
	}
	if err := store.EnsureStock(ctx, db, variantB, initialStock); err != nil {
		t.Fatalf("ensure stock B: %v", err)
	}

	const numGoroutines = 20
	carts := make([]commerce.Cart, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		shopper := commerce.Shopper{
			AnonymousToken: fmt.Sprintf("shopper-reverse-%d", i),
			Currency:       "TRY",
		}
		if i%2 == 0 {
			// Even carts: add A then B
			if _, _, err := svc.AddToCart(ctx, shopper, variantA, 1); err != nil {
				t.Fatalf("add item A to cart %d: %v", i, err)
			}
			cart, _, err := svc.AddToCart(ctx, shopper, variantB, 1)
			if err != nil {
				t.Fatalf("add item B to cart %d: %v", i, err)
			}
			carts[i] = cart
		} else {
			// Odd carts: add B then A (reverse order)
			if _, _, err := svc.AddToCart(ctx, shopper, variantB, 1); err != nil {
				t.Fatalf("add item B to cart %d: %v", i, err)
			}
			cart, _, err := svc.AddToCart(ctx, shopper, variantA, 1)
			if err != nil {
				t.Fatalf("add item A to cart %d: %v", i, err)
			}
			carts[i] = cart
		}
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			_, err := svc.Checkout(ctx, commerce.CheckoutRequest{
				CartID:         carts[idx].ID,
				GuestEmail:     fmt.Sprintf("guest-reverse%d@example.com", idx),
				IdempotencyKey: fmt.Sprintf("idem-reverse-%d", idx),
				Currency:       "TRY",
			})
			results[idx] = err
		}(i)
	}

	close(start)
	wg.Wait()

	var successCount, outOfStockCount int
	for i, err := range results {
		if err == nil {
			successCount++
		} else if errors.Is(err, commerce.ErrOutOfStock) {
			outOfStockCount++
		} else {
			t.Fatalf("goroutine %d failed with unexpected error (possible deadlock): %v", i, err)
		}
	}

	if successCount+outOfStockCount != numGoroutines {
		t.Fatalf("all checkouts must finish either successfully or with stock error, got success=%d outOfStock=%d", successCount, outOfStockCount)
	}

	stockA, err := store.GetStock(ctx, db, variantA)
	if err != nil {
		t.Fatalf("get stock A: %v", err)
	}
	stockB, err := store.GetStock(ctx, db, variantB)
	if err != nil {
		t.Fatalf("get stock B: %v", err)
	}

	if stockA.Available < 0 || stockB.Available < 0 {
		t.Fatalf("available stock dropped below zero: A=%d, B=%d", stockA.Available, stockB.Available)
	}
	if stockA.Available+stockA.Reserved != initialStock {
		t.Fatalf("stock A sum inconsistent: %d + %d != %d", stockA.Available, stockA.Reserved, initialStock)
	}
	if stockB.Available+stockB.Reserved != initialStock {
		t.Fatalf("stock B sum inconsistent: %d + %d != %d", stockB.Available, stockB.Reserved, initialStock)
	}
	if stockA.Reserved != successCount || stockB.Reserved != successCount {
		t.Fatalf("reserved stock does not match successful order count %d: A=%d, B=%d", successCount, stockA.Reserved, stockB.Reserved)
	}
}
