package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

type Store struct{ Dialect database.Dialect }

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
	q := `SELECT id, user_id, currency, total_cents, payment_status, shipping_status, created_at FROM commerce_orders WHERE user_id = ` + s.placeholder(1) + ` AND idempotency_key = ` + s.placeholder(2)
	err := db.QueryRowContext(ctx, q, userID, key).Scan(&order.ID, &order.UserID, &order.Currency, &order.TotalCents, &order.PaymentStatus, &order.ShippingStatus, &created)
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
	q := `SELECT product_id, sku, title, quantity, unit_cents, total_cents FROM commerce_order_items WHERE order_id = ` + s.placeholder(1) + ` ORDER BY id`
	rows, err := queryRows(ctx, db, q, order.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item commerce.OrderItem
		if err := rows.Scan(&item.ProductID, &item.SKU, &item.Title, &item.Quantity, &item.UnitCents, &item.TotalCents); err != nil {
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
	result, err := db.ExecContext(ctx, q, quantity, time.Now().UTC(), id, quantity)
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
	q := `INSERT INTO commerce_orders (id, user_id, idempotency_key, currency, total_cents, payment_status, shipping_status, created_at) VALUES (` + s.placeholder(1) + `, ` + s.placeholder(2) + `, ` + s.placeholder(3) + `, ` + s.placeholder(4) + `, ` + s.placeholder(5) + `, ` + s.placeholder(6) + `, ` + s.placeholder(7) + `, ` + s.placeholder(8) + `)`
	if _, err := db.ExecContext(ctx, q, order.ID, order.UserID, key, order.Currency, order.TotalCents, order.PaymentStatus, order.ShippingStatus, now); err != nil {
		return err
	}
	for index, item := range order.Items {
		q = `INSERT INTO commerce_order_items (id, order_id, product_id, sku, title, quantity, unit_cents, total_cents) VALUES (` + s.placeholder(1) + `, ` + s.placeholder(2) + `, ` + s.placeholder(3) + `, ` + s.placeholder(4) + `, ` + s.placeholder(5) + `, ` + s.placeholder(6) + `, ` + s.placeholder(7) + `, ` + s.placeholder(8) + `)`
		if _, err := db.ExecContext(ctx, q, fmt.Sprintf("%s-item-%d", order.ID, index), order.ID, item.ProductID, item.SKU, item.Title, item.Quantity, item.UnitCents, item.TotalCents); err != nil {
			return err
		}
	}
	return nil
}
