package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

type Store struct{ Dialect database.Dialect }

func (s Store) GetOrder(ctx context.Context, db database.Executor, orderID string) (commerce.Order, error) {
	if err := database.RequireExecutor(db); err != nil {
		return commerce.Order{}, err
	}
	var order commerce.Order
	q := `SELECT id, user_id, currency, total_cents, payment_status, shipping_status, customer_name, customer_identity_number, customer_email, customer_phone, customer_address, customer_city, customer_country, customer_postal_code, customer_client_ip FROM commerce_orders WHERE id = ` + s.placeholder(1)
	if err := db.QueryRowContext(ctx, q, orderID).Scan(&order.ID, &order.UserID, &order.Currency, &order.TotalCents, &order.PaymentStatus, &order.ShippingStatus, &order.Customer.Name, &order.Customer.IdentityNumber, &order.Customer.Email, &order.Customer.Phone, &order.Customer.Address, &order.Customer.City, &order.Customer.Country, &order.Customer.PostalCode, &order.Customer.ClientIP); err != nil {
		return commerce.Order{}, err
	}
	if err := s.loadItems(ctx, db, &order); err != nil {
		return commerce.Order{}, err
	}
	return order, nil
}

func (s Store) CreateVariant(ctx context.Context, db database.Executor, variant commerce.ProductVariant) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	now := time.Now().UTC()
	q := `INSERT INTO commerce_product_variants (id, product_id, sku, title, price_cents, stock, status, created_at, updated_at) VALUES (` + s.placeholder(1) + `,` + s.placeholder(2) + `,` + s.placeholder(3) + `,` + s.placeholder(4) + `,` + s.placeholder(5) + `,` + s.placeholder(6) + `,` + s.placeholder(7) + `,` + s.placeholder(8) + `,` + s.placeholder(9) + `)`
	_, err := db.ExecContext(ctx, q, variant.ID, variant.ProductID, variant.SKU, variant.Title, variant.PriceCents, variant.Stock, variant.Status, now, now)
	return err
}

func (s Store) ReserveVariant(ctx context.Context, db database.Executor, id string, quantity int) (commerce.ProductVariant, error) {
	if err := database.RequireExecutor(db); err != nil {
		return commerce.ProductVariant{}, err
	}
	q := `UPDATE commerce_product_variants SET stock = stock - ` + s.placeholder(1) + `, status = CASE WHEN stock - ` + s.placeholder(1) + ` = 0 THEN 'out_of_stock' ELSE status END, updated_at = ` + s.placeholder(2) + ` WHERE id = ` + s.placeholder(3) + ` AND status IN ('published', 'out_of_stock') AND stock >= ` + s.placeholder(4)
	result, err := db.ExecContext(ctx, q, quantity, quantity, time.Now().UTC(), id, quantity)
	if err != nil {
		return commerce.ProductVariant{}, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return commerce.ProductVariant{}, err
	}
	if changed != 1 {
		return commerce.ProductVariant{}, commerce.ErrOutOfStock
	}
	var variant commerce.ProductVariant
	q = `SELECT v.id, v.product_id, v.sku, v.title, v.price_cents, p.currency, v.stock, v.status FROM commerce_product_variants v JOIN commerce_products p ON p.id = v.product_id WHERE v.id = ` + s.placeholder(1)
	err = db.QueryRowContext(ctx, q, id).Scan(&variant.ID, &variant.ProductID, &variant.SKU, &variant.Title, &variant.PriceCents, &variant.Currency, &variant.Stock, &variant.Status)
	return variant, err
}

func (s Store) ListVariants(ctx context.Context, db database.Executor, productID string, publicOnly bool) ([]commerce.ProductVariant, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	q := `SELECT v.id, v.product_id, v.sku, v.title, v.price_cents, p.currency, v.stock, v.status FROM commerce_product_variants v JOIN commerce_products p ON p.id = v.product_id WHERE v.product_id = ` + s.placeholder(1)
	if publicOnly {
		q += ` AND v.status IN ('published', 'out_of_stock')`
	}
	q += ` ORDER BY v.title, v.id`
	rows, err := queryRows(ctx, db, q, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	variants := make([]commerce.ProductVariant, 0)
	for rows.Next() {
		var variant commerce.ProductVariant
		if err := rows.Scan(&variant.ID, &variant.ProductID, &variant.SKU, &variant.Title, &variant.PriceCents, &variant.Currency, &variant.Stock, &variant.Status); err != nil {
			return nil, err
		}
		variants = append(variants, variant)
	}
	return variants, rows.Err()
}

func (s Store) ListOrders(ctx context.Context, db database.Executor, limit int) ([]commerce.Order, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := `SELECT id, user_id, currency, total_cents, payment_status, shipping_status, customer_name, customer_identity_number, customer_email, customer_phone, customer_address, customer_city, customer_country, customer_postal_code, customer_client_ip FROM commerce_orders ORDER BY created_at DESC, id DESC LIMIT ` + s.placeholder(1)
	rows, err := queryRows(ctx, db, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := make([]commerce.Order, 0)
	for rows.Next() {
		var order commerce.Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Currency, &order.TotalCents, &order.PaymentStatus, &order.ShippingStatus, &order.Customer.Name, &order.Customer.IdentityNumber, &order.Customer.Email, &order.Customer.Phone, &order.Customer.Address, &order.Customer.City, &order.Customer.Country, &order.Customer.PostalCode, &order.Customer.ClientIP); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for index := range orders {
		if err := s.loadItems(ctx, db, &orders[index]); err != nil {
			return nil, err
		}
	}
	return orders, nil
}

func (s Store) CreateProduct(ctx context.Context, db database.Executor, product commerce.Product) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	q := `INSERT INTO commerce_products (id, sku, title, description, media_id, price_cents, currency, stock, status, created_at, updated_at) VALUES (` + s.placeholder(1) + `,` + s.placeholder(2) + `,` + s.placeholder(3) + `,` + s.placeholder(4) + `,` + s.placeholder(5) + `,` + s.placeholder(6) + `,` + s.placeholder(7) + `,` + s.placeholder(8) + `,` + s.placeholder(9) + `,` + s.placeholder(10) + `,` + s.placeholder(11) + `)`
	now := time.Now().UTC()
	_, err := db.ExecContext(ctx, q, product.ID, product.SKU, product.Title, product.Description, product.MediaID, product.PriceCents, product.Currency, product.Stock, product.Status, now, now)
	return err
}

func (s Store) UpdateProduct(ctx context.Context, db database.Executor, product commerce.Product) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	now := time.Now().UTC()
	q := `UPDATE commerce_products SET sku = ` + s.placeholder(1) + `, title = ` + s.placeholder(2) + `, description = ` + s.placeholder(3) + `, media_id = ` + s.placeholder(4) + `, price_cents = ` + s.placeholder(5) + `, currency = ` + s.placeholder(6) + `, stock = ` + s.placeholder(7) + `, status = ` + s.placeholder(8) + `, updated_at = ` + s.placeholder(9) + ` WHERE id = ` + s.placeholder(10)
	result, err := db.ExecContext(ctx, q, product.SKU, product.Title, product.Description, product.MediaID, product.PriceCents, product.Currency, product.Stock, product.Status, now, product.ID)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (s Store) GetProduct(ctx context.Context, db database.Executor, id string) (commerce.Product, error) {
	if err := database.RequireExecutor(db); err != nil {
		return commerce.Product{}, err
	}
	var product commerce.Product
	q := `SELECT id, sku, title, description, media_id, price_cents, currency, stock, status FROM commerce_products WHERE id = ` + s.placeholder(1)
	err := db.QueryRowContext(ctx, q, id).Scan(&product.ID, &product.SKU, &product.Title, &product.Description, &product.MediaID, &product.PriceCents, &product.Currency, &product.Stock, &product.Status)
	return product, err
}

func (s Store) ListProducts(ctx context.Context, db database.Executor, publicOnly bool) ([]commerce.Product, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	q := `SELECT id, sku, title, description, media_id, price_cents, currency, stock, status FROM commerce_products`
	if publicOnly {
		q += ` WHERE status IN ('published', 'out_of_stock')`
	}
	q += ` ORDER BY title, id`
	rows, err := queryRows(ctx, db, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	products := make([]commerce.Product, 0)
	for rows.Next() {
		var product commerce.Product
		if err := rows.Scan(&product.ID, &product.SKU, &product.Title, &product.Description, &product.MediaID, &product.PriceCents, &product.Currency, &product.Stock, &product.Status); err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, rows.Err()
}

func (s Store) GetOrCreateCart(ctx context.Context, db database.Executor, userID, currency string) (commerce.Cart, error) {
	cart, err := s.findCart(ctx, db, userID)
	if err != nil {
		return commerce.Cart{}, err
	}
	if cart.ID != "" {
		return cart, nil
	}
	id := fmt.Sprintf("cart-%d", time.Now().UnixNano())
	now := time.Now().UTC()
	q := `INSERT INTO commerce_carts (id, user_id, currency, created_at, updated_at) VALUES (` + s.placeholder(1) + `, ` + s.placeholder(2) + `, ` + s.placeholder(3) + `, ` + s.placeholder(4) + `, ` + s.placeholder(5) + `)`
	if _, err := db.ExecContext(ctx, q, id, userID, currency, now, now); err != nil {
		return commerce.Cart{}, err
	}
	return s.findCart(ctx, db, userID)
}

func (s Store) GetCart(ctx context.Context, db database.Executor, userID, cartID string) (commerce.Cart, error) {
	cart, err := s.findCart(ctx, db, userID)
	if err != nil || cart.ID != cartID {
		return commerce.Cart{}, sql.ErrNoRows
	}
	return cart, nil
}

func (s Store) findCart(ctx context.Context, db database.Executor, userID string) (commerce.Cart, error) {
	var cart commerce.Cart
	q := `SELECT id, user_id, currency FROM commerce_carts WHERE user_id = ` + s.placeholder(1)
	err := db.QueryRowContext(ctx, q, userID).Scan(&cart.ID, &cart.UserID, &cart.Currency)
	if errors.Is(err, sql.ErrNoRows) {
		return cart, nil
	}
	if err != nil {
		return cart, err
	}
	q = `SELECT i.id, i.cart_id, i.product_id, COALESCE(i.variant_id, ''), COALESCE(v.sku, p.sku), COALESCE(v.title, p.title), i.quantity, i.unit_cents FROM commerce_cart_items i JOIN commerce_products p ON p.id = i.product_id LEFT JOIN commerce_product_variants v ON v.id = i.variant_id WHERE i.cart_id = ` + s.placeholder(1) + ` ORDER BY i.id`
	rows, err := queryRows(ctx, db, q, cart.ID)
	if err != nil {
		return commerce.Cart{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var item commerce.CartItem
		if err := rows.Scan(&item.ID, &item.CartID, &item.ProductID, &item.VariantID, &item.SKU, &item.Title, &item.Quantity, &item.UnitCents); err != nil {
			return commerce.Cart{}, err
		}
		item.TotalCents = item.UnitCents * int64(item.Quantity)
		cart.Items = append(cart.Items, item)
		cart.TotalCents += item.TotalCents
	}
	return cart, rows.Err()
}

func (s Store) AddCartItem(ctx context.Context, db database.Executor, userID, productID, variantID string, quantity int) (commerce.Cart, error) {
	cart, err := s.GetOrCreateCart(ctx, db, userID, "TRY")
	if err != nil {
		return commerce.Cart{}, err
	}
	var price int64
	var status string
	var stock int
	var q string
	if variantID != "" {
		q := `SELECT price_cents, status, stock, product_id FROM commerce_product_variants WHERE id = ` + s.placeholder(1)
		if err := db.QueryRowContext(ctx, q, variantID).Scan(&price, &status, &stock, &productID); err != nil {
			return commerce.Cart{}, err
		}
	} else {
		q := `SELECT price_cents, status, stock FROM commerce_products WHERE id = ` + s.placeholder(1)
		if err := db.QueryRowContext(ctx, q, productID).Scan(&price, &status, &stock); err != nil {
			return commerce.Cart{}, err
		}
	}
	if status != string(commerce.ProductPublished) || stock < quantity {
		return commerce.Cart{}, commerce.ErrOutOfStock
	}
	var existing int
	q = `SELECT quantity FROM commerce_cart_items WHERE cart_id = ` + s.placeholder(1) + ` AND product_id = ` + s.placeholder(2) + ` AND COALESCE(variant_id, '') = ` + s.placeholder(3)
	err = db.QueryRowContext(ctx, q, cart.ID, productID, variantID).Scan(&existing)
	if errors.Is(err, sql.ErrNoRows) {
		q = `INSERT INTO commerce_cart_items (id, cart_id, product_id, variant_id, quantity, unit_cents) VALUES (` + s.placeholder(1) + `, ` + s.placeholder(2) + `, ` + s.placeholder(3) + `, ` + s.placeholder(4) + `, ` + s.placeholder(5) + `, ` + s.placeholder(6) + `)`
		var variantArg any
		if variantID != "" {
			variantArg = variantID
		}
		_, err = db.ExecContext(ctx, q, fmt.Sprintf("cart-item-%d", time.Now().UnixNano()), cart.ID, productID, variantArg, quantity, price)
	} else if err == nil {
		if existing+quantity > stock {
			return commerce.Cart{}, commerce.ErrOutOfStock
		}
		q = `UPDATE commerce_cart_items SET quantity = quantity + ` + s.placeholder(1) + `, unit_cents = ` + s.placeholder(2) + ` WHERE cart_id = ` + s.placeholder(3) + ` AND product_id = ` + s.placeholder(4) + ` AND COALESCE(variant_id, '') = ` + s.placeholder(5)
		_, err = db.ExecContext(ctx, q, quantity, price, cart.ID, productID, variantID)
	}
	if err != nil {
		return commerce.Cart{}, err
	}
	return s.findCart(ctx, db, userID)
}

func (s Store) RemoveCartItem(ctx context.Context, db database.Executor, userID, productID string) error {
	cart, err := s.findCart(ctx, db, userID)
	if err != nil {
		return err
	}
	if cart.ID == "" {
		return nil
	}
	q := `DELETE FROM commerce_cart_items WHERE cart_id = ` + s.placeholder(1) + ` AND product_id = ` + s.placeholder(2)
	_, err = db.ExecContext(ctx, q, cart.ID, productID)
	return err
}

func (s Store) ClearCart(ctx context.Context, db database.Executor, userID, cartID string) error {
	cart, err := s.findCart(ctx, db, userID)
	if err != nil {
		return err
	}
	if cart.ID != cartID {
		return sql.ErrNoRows
	}
	q := `DELETE FROM commerce_cart_items WHERE cart_id = ` + s.placeholder(1)
	_, err = db.ExecContext(ctx, q, cartID)
	return err
}

func (s Store) placeholder(n int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func (s Store) FindOrderByIdempotency(ctx context.Context, db database.Executor, userID, key string) (commerce.Order, bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return commerce.Order{}, false, err
	}
	var order commerce.Order
	var created string
	q := `SELECT id, user_id, currency, total_cents, payment_status, shipping_status, customer_name, customer_identity_number, customer_email, customer_phone, customer_address, customer_city, customer_country, customer_postal_code, customer_client_ip, created_at FROM commerce_orders WHERE user_id = ` + s.placeholder(1) + ` AND idempotency_key = ` + s.placeholder(2)
	err := db.QueryRowContext(ctx, q, userID, key).Scan(&order.ID, &order.UserID, &order.Currency, &order.TotalCents, &order.PaymentStatus, &order.ShippingStatus, &order.Customer.Name, &order.Customer.IdentityNumber, &order.Customer.Email, &order.Customer.Phone, &order.Customer.Address, &order.Customer.City, &order.Customer.Country, &order.Customer.PostalCode, &order.Customer.ClientIP, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return commerce.Order{}, false, nil
	}
	if err != nil {
		return commerce.Order{}, false, err
	}
	if err := s.loadItems(ctx, db, &order); err != nil {
		return commerce.Order{}, false, err
	}
	return order, true, nil
}

func (s Store) loadItems(ctx context.Context, db database.Executor, order *commerce.Order) error {
	q := `SELECT product_id, COALESCE(variant_id, ''), sku, title, quantity, unit_cents, total_cents FROM commerce_order_items WHERE order_id = ` + s.placeholder(1) + ` ORDER BY id`
	rows, err := queryRows(ctx, db, q, order.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item commerce.OrderItem
		if err := rows.Scan(&item.ProductID, &item.VariantID, &item.SKU, &item.Title, &item.Quantity, &item.UnitCents, &item.TotalCents); err != nil {
			return err
		}
		order.Items = append(order.Items, item)
	}
	return rows.Err()
}

func queryRows(ctx context.Context, db database.Executor, query string, args ...any) (*sql.Rows, error) {
	// The common executor intentionally exposes QueryRow only. SQL store
	// implementations receive *sql.DB/*sql.Tx in production and use this
	// narrow assertion at the adapter boundary.
	switch value := db.(type) {
	case interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	}:
		return value.QueryContext(ctx, query, args...)
	default:
		return nil, database.ErrDatabaseRequired
	}
}

func (s Store) ReserveProduct(ctx context.Context, db database.Executor, id string, quantity int) (commerce.Product, error) {
	if err := database.RequireExecutor(db); err != nil {
		return commerce.Product{}, err
	}
	q := `UPDATE commerce_products SET stock = stock - ` + s.placeholder(1) + `, status = CASE WHEN stock - ` + s.placeholder(1) + ` = 0 THEN 'out_of_stock' ELSE status END, updated_at = ` + s.placeholder(2) + ` WHERE id = ` + s.placeholder(3) + ` AND status IN ('published', 'out_of_stock') AND stock >= ` + s.placeholder(4)
	result, err := db.ExecContext(ctx, q, quantity, quantity, time.Now().UTC(), id, quantity)
	if err != nil {
		return commerce.Product{}, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return commerce.Product{}, err
	}
	if changed != 1 {
		return commerce.Product{}, commerce.ErrOutOfStock
	}
	var product commerce.Product
	q = `SELECT id, sku, title, price_cents, currency, stock, status FROM commerce_products WHERE id = ` + s.placeholder(1)
	err = db.QueryRowContext(ctx, q, id).Scan(&product.ID, &product.SKU, &product.Title, &product.PriceCents, &product.Currency, &product.Stock, &product.Status)
	return product, err
}

func (s Store) CreateOrder(ctx context.Context, db database.Executor, order commerce.Order, key string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	now := time.Now().UTC()
	q := `INSERT INTO commerce_orders (id, user_id, idempotency_key, currency, total_cents, payment_status, shipping_status, customer_name, customer_identity_number, customer_email, customer_phone, customer_address, customer_city, customer_country, customer_postal_code, customer_client_ip, created_at) VALUES (` + s.placeholder(1) + `, ` + s.placeholder(2) + `, ` + s.placeholder(3) + `, ` + s.placeholder(4) + `, ` + s.placeholder(5) + `, ` + s.placeholder(6) + `, ` + s.placeholder(7) + `, ` + s.placeholder(8) + `, ` + s.placeholder(9) + `, ` + s.placeholder(10) + `, ` + s.placeholder(11) + `, ` + s.placeholder(12) + `, ` + s.placeholder(13) + `, ` + s.placeholder(14) + `, ` + s.placeholder(15) + `, ` + s.placeholder(16) + `, ` + s.placeholder(17) + `)`
	if _, err := db.ExecContext(ctx, q, order.ID, order.UserID, key, order.Currency, order.TotalCents, order.PaymentStatus, order.ShippingStatus, order.Customer.Name, order.Customer.IdentityNumber, order.Customer.Email, order.Customer.Phone, order.Customer.Address, order.Customer.City, order.Customer.Country, order.Customer.PostalCode, order.Customer.ClientIP, now); err != nil {
		return err
	}
	for index, item := range order.Items {
		q = `INSERT INTO commerce_order_items (id, order_id, product_id, variant_id, sku, title, quantity, unit_cents, total_cents) VALUES (` + s.placeholder(1) + `, ` + s.placeholder(2) + `, ` + s.placeholder(3) + `, ` + s.placeholder(4) + `, ` + s.placeholder(5) + `, ` + s.placeholder(6) + `, ` + s.placeholder(7) + `, ` + s.placeholder(8) + `, ` + s.placeholder(9) + `)`
		var variantArg any
		if item.VariantID != "" {
			variantArg = item.VariantID
		}
		if _, err := db.ExecContext(ctx, q, fmt.Sprintf("%s-item-%d", order.ID, index), order.ID, item.ProductID, variantArg, item.SKU, item.Title, item.Quantity, item.UnitCents, item.TotalCents); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) ClaimWebhook(ctx context.Context, db database.Executor, provider, eventID string) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	q := `INSERT INTO commerce_payment_webhook_events (provider, provider_event_id, received_at) VALUES (` + s.placeholder(1) + `, ` + s.placeholder(2) + `, ` + s.placeholder(3) + `)`
	_, err := db.ExecContext(ctx, q, provider, eventID, time.Now().UTC())
	if err != nil {
		message := strings.ToLower(err.Error())
		if strings.Contains(message, "unique") || strings.Contains(message, "duplicate") || strings.Contains(message, "constraint") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s Store) ApplyPaymentEvent(ctx context.Context, db database.Executor, event commerce.WebhookEvent) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	var total int64
	var currency string
	q := `SELECT total_cents, currency FROM commerce_orders WHERE id = ` + s.placeholder(1)
	if err := db.QueryRowContext(ctx, q, event.OrderID).Scan(&total, &currency); err != nil {
		return err
	}
	if total != event.AmountCents || currency != event.Currency {
		return commerce.ErrPaymentMismatch
	}
	transactionID := event.Provider + "-" + event.ProviderEventID
	q = `INSERT INTO commerce_payment_transactions (id, order_id, provider, provider_payment_id, amount_cents, currency, status, created_at, updated_at) VALUES (` + s.placeholder(1) + `, ` + s.placeholder(2) + `, ` + s.placeholder(3) + `, ` + s.placeholder(4) + `, ` + s.placeholder(5) + `, ` + s.placeholder(6) + `, ` + s.placeholder(7) + `, ` + s.placeholder(8) + `, ` + s.placeholder(9) + `)`
	now := time.Now().UTC()
	if _, err := db.ExecContext(ctx, q, transactionID, event.OrderID, event.Provider, event.PaymentID, event.AmountCents, event.Currency, event.Status, now, now); err != nil {
		return err
	}
	q = `UPDATE commerce_orders SET payment_status = ` + s.placeholder(1) + ` WHERE id = ` + s.placeholder(2) + ` AND payment_status IN ('pending', 'failed')`
	result, err := db.ExecContext(ctx, q, event.Status, event.OrderID)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return commerce.ErrPaymentMismatch
	}
	return nil
}

func (s Store) GetPaymentTransaction(ctx context.Context, db database.Executor, orderID string) (commerce.PaymentTransaction, error) {
	if err := database.RequireExecutor(db); err != nil {
		return commerce.PaymentTransaction{}, err
	}
	var transaction commerce.PaymentTransaction
	q := `SELECT order_id, provider, provider_payment_id, amount_cents, currency, status FROM commerce_payment_transactions WHERE order_id = ` + s.placeholder(1) + ` ORDER BY created_at DESC LIMIT 1`
	err := db.QueryRowContext(ctx, q, orderID).Scan(&transaction.OrderID, &transaction.Provider, &transaction.ProviderPaymentID, &transaction.AmountCents, &transaction.Currency, &transaction.Status)
	return transaction, err
}

func (s Store) MarkPaymentRefunded(ctx context.Context, db database.Executor, orderID, paymentID string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	now := time.Now().UTC()
	q := `UPDATE commerce_payment_transactions SET status = ` + s.placeholder(1) + `, updated_at = ` + s.placeholder(2) + ` WHERE order_id = ` + s.placeholder(3) + ` AND provider_payment_id = ` + s.placeholder(4) + ` AND status = 'paid'`
	result, err := db.ExecContext(ctx, q, "refunded", now, orderID, paymentID)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return commerce.ErrPaymentMismatch
	}
	q = `UPDATE commerce_orders SET payment_status = ` + s.placeholder(1) + ` WHERE id = ` + s.placeholder(2) + ` AND payment_status = 'paid'`
	result, err = db.ExecContext(ctx, q, "refunded", orderID)
	if err != nil {
		return err
	}
	changed, err = result.RowsAffected()
	if err != nil || changed != 1 {
		return commerce.ErrPaymentMismatch
	}
	return nil
}
