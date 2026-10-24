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

// --- Orders ----------------------------------------------------------------

// CreateOrder inserts an order and its items. If an order with the same
// (tenant, idempotency_key) already exists -- a retried checkout request --
// it returns the EXISTING order and isNew=false instead of erroring, so a
// client retry (or a double click) never creates a second order.
func (s Store) CreateOrder(ctx context.Context, db database.Executor, order commerce.Order, items []commerce.OrderItem) (commerce.Order, bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return commerce.Order{}, false, err
	}
	if err := commerce.ValidateOrder(order); err != nil {
		return commerce.Order{}, false, err
	}
	tenantID, err := s.tenantID(ctx)
	if err != nil {
		return commerce.Order{}, false, err
	}
	memberID := nullable(order.MemberID)
	query := `INSERT INTO commerce_customer_orders (id, member_id, guest_name, guest_email, guest_phone, guest_address, guest_city, guest_district, guest_country, guest_postal_code, idempotency_key, idempotency_hash, currency, subtotal_cents, discount_code, discount_cents, tax_cents, shipping_cents, total_cents, payment_status, fulfillment_status, created_at, updated_at, tenant_id) VALUES (` +
		s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `,` + s.p(9) + `,` + s.p(10) + `,` + s.p(11) + `,` + s.p(12) + `,` + s.p(13) + `,` + s.p(14) + `,` + s.p(15) + `,` + s.p(16) + `,` + s.p(17) + `,` + s.p(18) + `,` + s.p(19) + `,` + s.p(20) + `,` + s.p(21) + `,` + s.p(22) + `,` + s.p(23) + `,` + s.p(24) + `)`
	_, err = db.ExecContext(ctx, query, order.ID, memberID, order.GuestName, order.GuestEmail, order.GuestPhone, order.GuestAddress, order.GuestCity, order.GuestDistrict, order.GuestCountry, order.GuestPostalCode,
		order.IdempotencyKey, order.IdempotencyHash, order.Currency, order.SubtotalCents, order.DiscountCode, order.DiscountCents, order.TaxCents, order.ShippingCents, order.TotalCents, string(order.PaymentStatus), string(order.FulfillmentStatus), s.dbTime(order.CreatedAt), s.dbTime(order.UpdatedAt), tenantID)
	if err != nil {
		if database.IsUniqueViolation(err) {
			existing, found, getErr := s.GetOrderByIdempotencyKey(ctx, db, order.IdempotencyKey)
			if getErr != nil {
				return commerce.Order{}, false, getErr
			}
			if found {
				return existing, false, nil
			}
		}
		return commerce.Order{}, false, err
	}
	for _, item := range items {
		itemQuery := `INSERT INTO commerce_customer_order_items (id, order_id, variant_id, sku, title, quantity, unit_price_cents, total_cents, created_at, tenant_id) VALUES (` +
			s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `,` + s.p(9) + `,` + s.p(10) + `)`
		if _, err := db.ExecContext(ctx, itemQuery, item.ID, order.ID, item.VariantID, item.SKU, item.Title, item.Quantity, item.UnitPriceCents, item.TotalCents, s.dbTime(order.CreatedAt), tenantID); err != nil {
			return commerce.Order{}, false, err
		}
	}
	return order, true, nil
}

func (s Store) GetOrderByIdempotencyKey(ctx context.Context, db database.Executor, key string) (commerce.Order, bool, error) {
	args := []any{key}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return commerce.Order{}, false, err
	}
	row := db.QueryRowContext(ctx, `SELECT `+orderColumns+` FROM commerce_customer_orders WHERE idempotency_key = `+s.p(1)+filter, args...)
	return s.scanOrder(row)
}

// GetOrder returns one order by id.
func (s Store) GetOrder(ctx context.Context, db database.Executor, id string) (commerce.Order, error) {
	args := []any{id}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return commerce.Order{}, err
	}
	row := db.QueryRowContext(ctx, `SELECT `+orderColumns+` FROM commerce_customer_orders WHERE id = `+s.p(1)+filter, args...)
	order, found, err := s.scanOrder(row)
	if err != nil {
		return commerce.Order{}, err
	}
	if !found {
		return commerce.Order{}, commerce.ErrOrderNotFound
	}
	return order, nil
}

// ListOrderItems returns the line items of one order -- used to release
// their stock reservations when a payment fails or its hold expires
// (worker.go), since reservation and release both operate per variant.
func (s Store) ListOrderItems(ctx context.Context, db database.Executor, orderID string) ([]commerce.OrderItem, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("database executor does not support row queries")
	}
	filter, args, err := s.scope(ctx, []any{orderID})
	if err != nil {
		return nil, err
	}
	rows, err := queryer.QueryContext(ctx, `SELECT id, order_id, variant_id, sku, title, quantity, unit_price_cents, total_cents FROM commerce_customer_order_items WHERE order_id = `+s.p(1)+filter+` ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []commerce.OrderItem
	for rows.Next() {
		var item commerce.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.VariantID, &item.SKU, &item.Title, &item.Quantity, &item.UnitPriceCents, &item.TotalCents); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// orderScanner is satisfied by both *sql.Row and *sql.Rows, so the single
// order-column list below (id through updated_at) is written once and
// shared by every reader instead of duplicated per query.
type orderScanner interface {
	Scan(dest ...any) error
}

func (s Store) scanOrderRow(scanner orderScanner) (commerce.Order, error) {
	var order commerce.Order
	var paymentStatus, fulfillmentStatus string
	var shippedAt, created, updated any
	if err := scanner.Scan(&order.ID, &order.MemberID, &order.GuestName, &order.GuestEmail, &order.GuestPhone, &order.GuestAddress, &order.GuestCity, &order.GuestDistrict, &order.GuestCountry, &order.GuestPostalCode,
		&order.IdempotencyKey, &order.IdempotencyHash, &order.Currency, &order.SubtotalCents, &order.DiscountCode, &order.DiscountCents, &order.TaxCents, &order.ShippingCents, &order.TotalCents,
		&paymentStatus, &fulfillmentStatus, &order.ShippingCarrier, &order.ShippingTrackingNumber, &order.ShippingTrackingURL, &shippedAt, &created, &updated); err != nil {
		return commerce.Order{}, err
	}
	order.PaymentStatus = commerce.PaymentStatus(paymentStatus)
	order.FulfillmentStatus = commerce.FulfillmentStatus(fulfillmentStatus)
	var parseErr error
	if shippedAt != nil {
		if order.ShippedAt, parseErr = s.parseTime(shippedAt); parseErr != nil {
			return commerce.Order{}, parseErr
		}
	}
	if order.CreatedAt, parseErr = s.parseTime(created); parseErr != nil {
		return commerce.Order{}, parseErr
	}
	if order.UpdatedAt, parseErr = s.parseTime(updated); parseErr != nil {
		return commerce.Order{}, parseErr
	}
	return order, nil
}

func (s Store) scanOrder(row *sql.Row) (commerce.Order, bool, error) {
	order, err := s.scanOrderRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return commerce.Order{}, false, nil
		}
		return commerce.Order{}, false, err
	}
	return order, true, nil
}

const orderColumns = `id, COALESCE(member_id,''), guest_name, guest_email, guest_phone, guest_address, guest_city, guest_district, guest_country, guest_postal_code, idempotency_key, idempotency_hash, currency, subtotal_cents, discount_code, discount_cents, tax_cents, shipping_cents, total_cents, payment_status, fulfillment_status, shipping_carrier, shipping_tracking_number, shipping_tracking_url, shipped_at, created_at, updated_at`

// ListPendingOrdersOlderThan returns orders still awaiting payment whose
// hold should have expired by now -- worker.go's sweep calls this, then
// Service.ExpireOrder on each result.
func (s Store) ListPendingOrdersOlderThan(ctx context.Context, db database.Executor, cutoff time.Time, limit int) ([]commerce.Order, error) {
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
	args := []any{s.dbTime(cutoff)}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return nil, err
	}
	rows, err := queryer.QueryContext(ctx, `SELECT `+orderColumns+` FROM commerce_customer_orders WHERE payment_status = 'pending' AND created_at < `+s.p(1)+filter+` ORDER BY created_at LIMIT `+fmt.Sprintf("%d", limit), args...)
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

// ListOrdersByMemberID returns a member's own orders, newest first -- the
// read behind the member account surface's order history
// (internal/contracts/customerorders.Reader).
func (s Store) ListOrdersByMemberID(ctx context.Context, db database.Executor, memberID string, limit int) ([]commerce.Order, error) {
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
	args := []any{memberID}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return nil, err
	}
	rows, err := queryer.QueryContext(ctx, `SELECT `+orderColumns+` FROM commerce_customer_orders WHERE member_id = `+s.p(1)+filter+` ORDER BY created_at DESC LIMIT `+fmt.Sprintf("%d", limit), args...)
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

// ListOrders is the admin order-browsing query behind
// GET /api/v1/admin/commerce/orders -- optionally filtered by
// payment_status and/or fulfillment_status (either passed as "" skips
// that filter), newest first. No member/guest scoping: this is operator
// tooling, like ListPendingOrdersOlderThan's own unscoped-by-identity
// read, not a shopper-facing endpoint. limit+1 rows are fetched so the
// caller can tell whether another page exists without a separate COUNT
// query, the same over-fetch convention internal/contenttypes' admin list
// already uses.
func (s Store) ListOrders(ctx context.Context, db database.Executor, paymentStatus commerce.PaymentStatus, fulfillmentStatus commerce.FulfillmentStatus, limit, offset int) ([]commerce.Order, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	if limit < 1 {
		limit = 1
	}
	if offset < 0 {
		offset = 0
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("database executor does not support row queries")
	}

	query := `SELECT ` + orderColumns + ` FROM commerce_customer_orders WHERE 1=1`
	var args []any
	if paymentStatus != "" {
		args = append(args, string(paymentStatus))
		query += ` AND payment_status = ` + s.p(len(args))
	}
	if fulfillmentStatus != "" {
		args = append(args, string(fulfillmentStatus))
		query += ` AND fulfillment_status = ` + s.p(len(args))
	}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return nil, err
	}
	query += filter
	query += ` ORDER BY created_at DESC, id DESC LIMIT ` + fmt.Sprintf("%d", limit+1) + ` OFFSET ` + fmt.Sprintf("%d", offset)

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

// GetOrderByMemberID returns one order, scoped to the member who placed it --
// ErrOrderNotFound both when the order does not exist and when it belongs to
// someone else, so the member surface can't distinguish the two cases.
func (s Store) GetOrderByMemberID(ctx context.Context, db database.Executor, memberID, orderID string) (commerce.Order, error) {
	args := []any{orderID, memberID}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return commerce.Order{}, err
	}
	row := db.QueryRowContext(ctx, `SELECT `+orderColumns+` FROM commerce_customer_orders WHERE id = `+s.p(1)+` AND member_id = `+s.p(2)+filter, args...)
	order, found, err := s.scanOrder(row)
	if err != nil {
		return commerce.Order{}, err
	}
	if !found {
		return commerce.Order{}, commerce.ErrOrderNotFound
	}
	return order, nil
}

// UpdatePaymentStatus is a conditional UPDATE guarded by the current status,
// the same atomic-transition pattern as ReserveStock: it only applies when
// the order is still in the `from` state, so a redelivered payment webhook
// (expected and normal -- see modules/commerce/providers) can retry safely
// without double-processing the side effects (confirmation email, stock
// commit) that follow a successful transition.
func (s Store) UpdatePaymentStatus(ctx context.Context, db database.Executor, orderID string, from, to commerce.PaymentStatus) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	args := []any{string(to), s.dbTime(s.now()), orderID, string(from)}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return false, err
	}
	query := `UPDATE commerce_customer_orders SET payment_status = ` + s.p(1) + `, updated_at = ` + s.p(2) +
		` WHERE id = ` + s.p(3) + ` AND payment_status = ` + s.p(4) + filter
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

// ClaimShipment moves a paid, unfulfilled order to preparing so exactly one
// request may call the carrier. A preparing order whose claim is older than
// staleBefore (its request died mid call) can be claimed again.
func (s Store) ClaimShipment(ctx context.Context, db database.Executor, orderID string, staleBefore time.Time) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	filter, args, err := s.scope(ctx, []any{string(commerce.FulfillmentPreparing), s.dbTime(s.now()), orderID, string(commerce.PaymentPaid), string(commerce.FulfillmentUnfulfilled), string(commerce.FulfillmentPreparing), s.dbTime(staleBefore)})
	if err != nil {
		return false, err
	}
	query := `UPDATE commerce_customer_orders SET fulfillment_status = ` + s.p(1) + `, updated_at = ` + s.p(2) +
		` WHERE id = ` + s.p(3) + ` AND payment_status = ` + s.p(4) +
		` AND (fulfillment_status = ` + s.p(5) + ` OR (fulfillment_status = ` + s.p(6) + ` AND updated_at < ` + s.p(7) + `))` + filter
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

// ReleaseShipmentClaim gives a claim back after the carrier refused, so the
// operator can retry.
func (s Store) ReleaseShipmentClaim(ctx context.Context, db database.Executor, orderID string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	filter, args, err := s.scope(ctx, []any{string(commerce.FulfillmentUnfulfilled), s.dbTime(s.now()), orderID, string(commerce.FulfillmentPreparing)})
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `UPDATE commerce_customer_orders SET fulfillment_status = `+s.p(1)+`, updated_at = `+s.p(2)+
		` WHERE id = `+s.p(3)+` AND fulfillment_status = `+s.p(4)+filter, args...)
	return err
}

// RecordShipment stores the carrier's tracking details and moves a claimed
// (preparing) order to shipped. Guarded by fulfillment_status, so it applies
// at most once per claim.
func (s Store) RecordShipment(ctx context.Context, db database.Executor, orderID, carrier, trackingNumber, trackingURL string, shippedAt time.Time) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	args := []any{string(commerce.FulfillmentShipped), carrier, trackingNumber, trackingURL, s.dbTime(shippedAt), s.dbTime(s.now()), orderID}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return false, err
	}
	query := `UPDATE commerce_customer_orders SET fulfillment_status = ` + s.p(1) + `, shipping_carrier = ` + s.p(2) +
		`, shipping_tracking_number = ` + s.p(3) + `, shipping_tracking_url = ` + s.p(4) + `, shipped_at = ` + s.p(5) + `, updated_at = ` + s.p(6) +
		` WHERE id = ` + s.p(7) + ` AND fulfillment_status = '` + string(commerce.FulfillmentPreparing) + `'` + filter
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}
