package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

func TestSQLiteVariantReservationAndOrderListing(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, "file:commerce-contract?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE commerce_products (id TEXT PRIMARY KEY, sku TEXT NOT NULL UNIQUE, title TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', media_id TEXT NOT NULL DEFAULT '', price_cents INTEGER NOT NULL, currency TEXT NOT NULL, stock INTEGER NOT NULL, status TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE commerce_orders (id TEXT PRIMARY KEY, user_id TEXT NOT NULL, idempotency_key TEXT NOT NULL, currency TEXT NOT NULL, total_cents INTEGER NOT NULL, payment_status TEXT NOT NULL, shipping_status TEXT NOT NULL, customer_name TEXT NOT NULL DEFAULT '', customer_identity_number TEXT NOT NULL DEFAULT '', customer_email TEXT NOT NULL DEFAULT '', customer_phone TEXT NOT NULL DEFAULT '', customer_address TEXT NOT NULL DEFAULT '', customer_city TEXT NOT NULL DEFAULT '', customer_country TEXT NOT NULL DEFAULT '', customer_postal_code TEXT NOT NULL DEFAULT '', customer_client_ip TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, UNIQUE(user_id, idempotency_key))`,
		`CREATE TABLE commerce_payment_transactions (id TEXT PRIMARY KEY, order_id TEXT NOT NULL, provider TEXT NOT NULL, provider_payment_id TEXT NOT NULL UNIQUE, amount_cents INTEGER NOT NULL, currency TEXT NOT NULL, status TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE commerce_order_items (id TEXT PRIMARY KEY, order_id TEXT NOT NULL, product_id TEXT NOT NULL, sku TEXT NOT NULL, title TEXT NOT NULL, quantity INTEGER NOT NULL, unit_cents INTEGER NOT NULL, total_cents INTEGER NOT NULL, variant_id TEXT)`,
		`CREATE TABLE commerce_product_variants (id TEXT PRIMARY KEY, product_id TEXT NOT NULL, sku TEXT NOT NULL UNIQUE, title TEXT NOT NULL, price_cents INTEGER NOT NULL, stock INTEGER NOT NULL, status TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
	} {
		if _, err := db.ExecContext(context.Background(), statement); err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.Exec(`INSERT INTO commerce_products (id, sku, title, description, media_id, price_cents, currency, stock, status, created_at, updated_at) VALUES ('p1','P1','Bottle','','',1000,'TRY',10,'published','now','now')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO commerce_product_variants (id, product_id, sku, title, price_cents, stock, status, created_at, updated_at) VALUES ('v1','p1','V1','Blue',1200,2,'published','now','now')`)
	if err != nil {
		t.Fatal(err)
	}

	store := Store{Dialect: database.SQLite}
	variant, err := store.ReserveVariant(context.Background(), db, "v1", 1)
	if err != nil || variant.Currency != "TRY" || variant.Stock != 1 {
		t.Fatalf("reserve variant = %#v, %v", variant, err)
	}
	variants, err := store.ListVariants(context.Background(), db, "p1", true)
	if err != nil || len(variants) != 1 || variants[0].Currency != "TRY" {
		t.Fatalf("list variants = %#v, %v", variants, err)
	}

	_, err = db.Exec(`INSERT INTO commerce_orders (id,user_id,idempotency_key,currency,total_cents,payment_status,shipping_status,created_at) VALUES ('o1','u1','k1','TRY',1200,'pending','pending','now')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO commerce_order_items (id,order_id,product_id,variant_id,sku,title,quantity,unit_cents,total_cents) VALUES ('oi1','o1','p1','v1','V1','Blue',1,1200,1200)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyPaymentEvent(context.Background(), db, commerce.WebhookEvent{ID: "evt-1", Provider: "stripe", ProviderEventID: "evt-1", OrderID: "o1", PaymentID: "pi-1", Status: "paid", AmountCents: 1200, Currency: "TRY"}); err != nil {
		t.Fatal(err)
	}
	var paymentCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM commerce_payment_transactions WHERE provider_payment_id = 'pi-1'`).Scan(&paymentCount); err != nil || paymentCount != 1 {
		t.Fatalf("payment transaction count = %d, %v", paymentCount, err)
	}
	transaction, err := store.GetPaymentTransaction(context.Background(), db, "o1")
	if err != nil || transaction.ProviderPaymentID != "pi-1" || transaction.Status != "paid" {
		t.Fatalf("payment transaction = %+v, %v", transaction, err)
	}
	if err := store.ClaimPaymentRefund(context.Background(), db, "o1", "pi-1"); err != nil {
		t.Fatal(err)
	}
	if err := store.ClaimPaymentRefund(context.Background(), db, "o1", "pi-1"); !errors.Is(err, commerce.ErrRefundPending) {
		t.Fatalf("second refund claim = %v", err)
	}
	if err := store.MarkPaymentRefunded(context.Background(), db, "o1", "pi-1"); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT payment_status FROM commerce_orders WHERE id = 'o1'`).Scan(&transaction.Status); err != nil || transaction.Status != "refunded" {
		t.Fatalf("order refund status = %q, %v", transaction.Status, err)
	}
	orders, err := store.ListOrders(context.Background(), db, 10)
	if err != nil || len(orders) != 1 || len(orders[0].Items) != 1 || orders[0].Items[0].VariantID != "v1" {
		t.Fatalf("list orders = %#v, %v", orders, err)
	}
	orders, err = store.ListOrdersByUserID(context.Background(), db, "u1", 10)
	if err != nil || len(orders) != 1 || orders[0].UserID != "u1" {
		t.Fatalf("list customer orders = %#v, %v", orders, err)
	}
	_, err = db.Exec(`INSERT INTO commerce_products (id, sku, title, description, media_id, price_cents, currency, stock, status, created_at, updated_at) VALUES ('p2','P2','Reserved','','',1000,'TRY',0,'published','now','now')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO commerce_orders (id,user_id,idempotency_key,currency,total_cents,payment_status,shipping_status,created_at) VALUES ('o2','u2','k2','TRY',2000,'pending','pending','2000-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO commerce_order_items (id,order_id,product_id,sku,title,quantity,unit_cents,total_cents) VALUES ('oi2','o2','p2','P2','Reserved',2,1000,2000)`)
	if err != nil {
		t.Fatal(err)
	}
	expired, err := store.ExpirePendingOrders(context.Background(), db, time.Now().UTC())
	if err != nil || expired != 1 {
		t.Fatalf("expire pending = %d, %v", expired, err)
	}
	var paymentStatus string
	var stock int
	if err := db.QueryRow(`SELECT payment_status FROM commerce_orders WHERE id = 'o2'`).Scan(&paymentStatus); err != nil || paymentStatus != "expired" {
		t.Fatalf("expired payment status = %q, %v", paymentStatus, err)
	}
	if err := db.QueryRow(`SELECT stock FROM commerce_products WHERE id = 'p2'`).Scan(&stock); err != nil || stock != 2 {
		t.Fatalf("released stock = %d, %v", stock, err)
	}

	prod, err := store.GetProduct(context.Background(), db, "p1")
	if err != nil || prod.Title != "Bottle" {
		t.Fatalf("get product = %#v, %v", prod, err)
	}
	prod.Title = "Updated Bottle"
	prod.PriceCents = 1500
	if err := store.UpdateProduct(context.Background(), db, prod); err != nil {
		t.Fatalf("update product = %v", err)
	}
	prod2, err := store.GetProduct(context.Background(), db, "p1")
	if err != nil || prod2.Title != "Updated Bottle" || prod2.PriceCents != 1500 {
		t.Fatalf("get updated product = %#v, %v", prod2, err)
	}
}

var _ database.Executor = (*sql.DB)(nil)
var _ commerce.OrderReader = Store{}
