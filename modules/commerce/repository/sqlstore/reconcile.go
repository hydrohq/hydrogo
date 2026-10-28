package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

// ListNotifiedOrdersNotPaid returns orders that have a recorded payment
// notification for the given provider (commerce_payment_notifications,
// RecordPaymentNotification's own dedup table) but whose payment_status is
// not "paid" -- a confirmation that should have applied via
// UpdatePaymentStatus but didn't (for example: the order was already moved
// to "failed" by Worker.ExpireDue's own guarded transition before a
// delayed webhook arrived, so the pending->paid UPDATE found no row to
// apply to). For every provider this module ships today (paytr), the
// notification's provider_event_id IS the order id -- see
// modules/commerce/providers/paytr/transport.go's own doc comment -- so
// joining directly on id works without a separate order-id column on the
// notification itself.
func (s Store) ListNotifiedOrdersNotPaid(ctx context.Context, db database.Executor, provider string, limit int) ([]commerce.Order, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	if limit < 1 {
		limit = 1
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("database executor does not support row queries")
	}
	tenantID, err := s.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	qualifiedColumns := `o.id, COALESCE(o.member_id,''), o.guest_name, o.guest_email, o.guest_phone, o.guest_address, o.guest_city, o.guest_district, o.guest_country, o.guest_postal_code, o.idempotency_key, o.idempotency_hash, o.currency, o.subtotal_cents, o.discount_code, o.discount_cents, o.tax_cents, o.shipping_cents, o.total_cents, o.payment_status, o.fulfillment_status, o.shipping_carrier, o.shipping_tracking_number, o.shipping_tracking_url, o.shipped_at, o.created_at, o.updated_at`
	query := `SELECT ` + qualifiedColumns + ` FROM commerce_customer_orders o` +
		` JOIN commerce_payment_notifications n ON n.provider_event_id = o.id AND n.provider = ` + s.p(1) +
		` WHERE o.payment_status != 'paid'`
	args := []any{provider}
	if s.RequireTenantScope {
		// Two placeholders, two args: a ?-dialect cannot reuse one bound value.
		query += ` AND o.tenant_id = ` + s.p(2) + ` AND n.tenant_id = ` + s.p(3)
		args = append(args, tenantID, tenantID)
	}
	query += ` ORDER BY o.created_at LIMIT ` + fmt.Sprintf("%d", limit)
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []commerce.Order
	for rows.Next() {
		order, err := s.scanOrderRow(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

// StockReservationDrift compares commerce_stock.reserved against what
// pending orders actually still account for, per variant, and returns
// only the variants where they disagree -- the common, healthy case is an
// empty slice. Computed as two separate queries diffed in Go rather than
// one SQL join, since a correct "full outer join" comparison (a variant
// could have stock but no pending orders, or vice versa) needs syntax
// (FULL OUTER JOIN) SQLite does not support, and this module runs across
// SQLite, PostgreSQL and MySQL alike.
func (s Store) StockReservationDrift(ctx context.Context, db database.Executor, limit int) ([]commerce.ReservationDrift, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	if limit < 1 {
		limit = 1
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("database executor does not support row queries")
	}
	tenantID, err := s.tenantID(ctx)
	if err != nil {
		return nil, err
	}

	reserved := map[string]int{}
	{
		query := `SELECT variant_id, reserved FROM commerce_stock WHERE reserved > 0`
		var args []any
		if s.RequireTenantScope {
			query += ` AND tenant_id = ` + s.p(1)
			args = append(args, tenantID)
		}
		rows, err := queryer.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var variantID string
			var count int
			if err := rows.Scan(&variantID, &count); err != nil {
				_ = rows.Close()
				return nil, err
			}
			reserved[variantID] = count
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
	}

	expected := map[string]int{}
	{
		query := `SELECT oi.variant_id, SUM(oi.quantity) FROM commerce_customer_order_items oi` +
			` JOIN commerce_customer_orders o ON o.id = oi.order_id` +
			` WHERE o.payment_status = 'pending'`
		var args []any
		if s.RequireTenantScope {
			query += ` AND o.tenant_id = ` + s.p(1)
			args = append(args, tenantID)
		}
		query += ` GROUP BY oi.variant_id`
		rows, err := queryer.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var variantID string
			var count int
			if err := rows.Scan(&variantID, &count); err != nil {
				_ = rows.Close()
				return nil, err
			}
			expected[variantID] = count
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
	}

	variantIDs := make([]string, 0, len(reserved)+len(expected))
	seen := map[string]bool{}
	for variantID := range reserved {
		if !seen[variantID] {
			seen[variantID] = true
			variantIDs = append(variantIDs, variantID)
		}
	}
	for variantID := range expected {
		if !seen[variantID] {
			seen[variantID] = true
			variantIDs = append(variantIDs, variantID)
		}
	}
	sort.Strings(variantIDs)

	drift := make([]commerce.ReservationDrift, 0, len(variantIDs))
	for _, variantID := range variantIDs {
		if reserved[variantID] != expected[variantID] {
			drift = append(drift, commerce.ReservationDrift{VariantID: variantID, Reserved: reserved[variantID], Expected: expected[variantID]})
		}
		if len(drift) >= limit {
			break
		}
	}
	return drift, nil
}
