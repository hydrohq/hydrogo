package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/modules/commerce"
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
	holdUntil := s.nullableTime(order.HoldUntil)
	query := s.insertIgnoreSQL(`commerce_customer_orders (id, order_number, member_id, guest_name, guest_email, guest_phone, guest_address, guest_city, guest_district, guest_country, guest_postal_code, idempotency_key, idempotency_hash, currency, subtotal_cents, discount_code, discount_cents, tax_cents, shipping_cents, total_cents, payment_method, payment_ref, hold_until, payment_status, fulfillment_status, created_at, updated_at, tenant_id) VALUES (` +
		s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `,` + s.p(9) + `,` + s.p(10) + `,` + s.p(11) + `,` + s.p(12) + `,` + s.p(13) + `,` + s.p(14) + `,` + s.p(15) + `,` + s.p(16) + `,` + s.p(17) + `,` + s.p(18) + `,` + s.p(19) + `,` + s.p(20) + `,` + s.p(21) + `,` + s.p(22) + `,` + s.p(23) + `,` + s.p(24) + `,` + s.p(25) + `,` + s.p(26) + `,` + s.p(27) + `,` + s.p(28) + `)`)
	res, err := db.ExecContext(ctx, query, order.ID, order.OrderNumber, memberID, order.GuestName, order.GuestEmail, order.GuestPhone, order.GuestAddress, order.GuestCity, order.GuestDistrict, order.GuestCountry, order.GuestPostalCode,
		order.IdempotencyKey, order.IdempotencyHash, order.Currency, order.SubtotalCents, order.DiscountCode, order.DiscountCents, order.TaxCents, order.ShippingCents, order.TotalCents, order.PaymentMethod, order.PaymentRef, holdUntil, string(order.PaymentStatus), string(order.FulfillmentStatus), s.dbTime(order.CreatedAt), s.dbTime(order.UpdatedAt), tenantID)
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
	if affected, affErr := res.RowsAffected(); affErr == nil && affected == 0 {
		existing, found, getErr := s.GetOrderByIdempotencyKey(ctx, db, order.IdempotencyKey)
		if getErr != nil {
			return commerce.Order{}, false, getErr
		}
		if found {
			return existing, false, nil
		}
		return commerce.Order{}, false, errors.New("order insert conflicted without a matching idempotency key")
	}
	for _, item := range items {
		itemQuery := `INSERT INTO commerce_customer_order_items (id, order_id, variant_id, sku, title, quantity, unit_price_cents, total_cents, created_at, tenant_id) VALUES (` +
			s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `,` + s.p(9) + `,` + s.p(10) + `)`
		if _, err := db.ExecContext(ctx, itemQuery, item.ID, order.ID, item.VariantID, item.SKU, item.Title, item.Quantity, item.UnitPriceCents, item.TotalCents, s.dbTime(order.CreatedAt), tenantID); err != nil {
			return commerce.Order{}, false, err
		}
	}
	if order.InvoiceStatus == "" {
		order.InvoiceStatus = commerce.InvoiceNone
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

// GetOrderByNumber returns the most recent order matching a human-readable order number.
func (s Store) GetOrderByNumber(ctx context.Context, db database.Executor, orderNumber string) (commerce.Order, error) {
	args := []any{strings.TrimSpace(orderNumber)}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return commerce.Order{}, err
	}
	row := db.QueryRowContext(ctx, `SELECT `+orderColumns+` FROM commerce_customer_orders WHERE UPPER(order_number) = UPPER(`+s.p(1)+`)`+filter+` ORDER BY created_at DESC LIMIT 1`, args...)
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
	rows, err := queryer.QueryContext(ctx, `SELECT id, order_id, variant_id, sku, title, quantity, unit_price_cents, total_cents FROM commerce_customer_order_items WHERE order_id = `+s.p(1)+filter+` ORDER BY variant_id ASC, id ASC`, args...)
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
	var paymentStatus, fulfillmentStatus, invoiceStatus string
	var holdUntil, shippedAt, deliveredAt, invoiceClaimedAt, invoicedAt, created, updated any
	if err := scanner.Scan(&order.ID, &order.OrderNumber, &order.MemberID, &order.GuestName, &order.GuestEmail, &order.GuestPhone, &order.GuestAddress, &order.GuestCity, &order.GuestDistrict, &order.GuestCountry, &order.GuestPostalCode,
		&order.IdempotencyKey, &order.IdempotencyHash, &order.Currency, &order.SubtotalCents, &order.DiscountCode, &order.DiscountCents, &order.TaxCents, &order.ShippingCents, &order.TotalCents,
		&order.PaymentMethod, &order.PaymentRef, &holdUntil,
		&paymentStatus, &fulfillmentStatus, &order.ShippingCarrier, &order.ShippingTrackingNumber, &order.ShippingTrackingURL, &order.ShippingShipmentID, &order.ShippingLabelURL, &shippedAt, &deliveredAt,
		&invoiceStatus, &order.InvoiceProvider, &order.InvoiceGUID, &order.InvoiceNo, &order.InvoiceURL, &order.InvoiceError, &invoiceClaimedAt, &invoicedAt,
		&created, &updated); err != nil {
		return commerce.Order{}, err
	}
	order.PaymentStatus = commerce.PaymentStatus(paymentStatus)
	order.FulfillmentStatus = commerce.FulfillmentStatus(fulfillmentStatus)
	if invoiceStatus == "" {
		order.InvoiceStatus = commerce.InvoiceNone
	} else {
		order.InvoiceStatus = commerce.InvoiceStatus(invoiceStatus)
	}
	var parseErr error
	if holdUntil != nil {
		if order.HoldUntil, parseErr = s.parseTime(holdUntil); parseErr != nil {
			return commerce.Order{}, parseErr
		}
	}
	if shippedAt != nil {
		if order.ShippedAt, parseErr = s.parseTime(shippedAt); parseErr != nil {
			return commerce.Order{}, parseErr
		}
	}
	if deliveredAt != nil {
		if order.DeliveredAt, parseErr = s.parseTime(deliveredAt); parseErr != nil {
			return commerce.Order{}, parseErr
		}
	}
	if invoiceClaimedAt != nil {
		if order.InvoiceClaimedAt, parseErr = s.parseTime(invoiceClaimedAt); parseErr != nil {
			return commerce.Order{}, parseErr
		}
	}
	if invoicedAt != nil {
		if order.InvoicedAt, parseErr = s.parseTime(invoicedAt); parseErr != nil {
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
	if order.InvoiceStatus == "" {
		order.InvoiceStatus = commerce.InvoiceNone
	}
	return order, true, nil
}

const orderColumns = `id, COALESCE(order_number,''), COALESCE(member_id,''), guest_name, guest_email, guest_phone, guest_address, guest_city, guest_district, guest_country, guest_postal_code, idempotency_key, idempotency_hash, currency, subtotal_cents, discount_code, discount_cents, tax_cents, shipping_cents, total_cents, payment_method, payment_ref, hold_until, payment_status, fulfillment_status, shipping_carrier, shipping_tracking_number, shipping_tracking_url, shipping_shipment_id, shipping_label_url, shipped_at, delivered_at, invoice_status, invoice_provider, invoice_guid, invoice_no, invoice_url, invoice_error, invoice_claimed_at, invoiced_at, created_at, updated_at`

func (s Store) nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return s.dbTime(t)
}

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

// ListExpiredPendingOrders returns pending orders whose payment hold has
// elapsed: card orders use cardCutoff, manual bank-transfer ("havale") orders
// use bankTransferCutoff, and orders with a hold_until deadline in the future
// (e.g. awaiting gateway fraud screening) are kept until that deadline passes.
func (s Store) ListExpiredPendingOrders(ctx context.Context, db database.Executor, now, cardCutoff, bankTransferCutoff time.Time, limit int) ([]commerce.Order, error) {
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
	args := []any{s.dbTime(now), s.dbTime(bankTransferCutoff), s.dbTime(cardCutoff)}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return nil, err
	}
	query := `SELECT ` + orderColumns + ` FROM commerce_customer_orders` +
		` WHERE payment_status = 'pending'` +
		` AND (hold_until IS NULL OR hold_until < ` + s.p(1) + `)` +
		` AND (((payment_method = 'havale' OR discount_code = 'method:havale') AND created_at < ` + s.p(2) + `)` +
		` OR (payment_method <> 'havale' AND discount_code <> 'method:havale' AND created_at < ` + s.p(3) + `))` +
		filter + ` ORDER BY created_at LIMIT ` + fmt.Sprintf("%d", limit)
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

// UpdateOrderPaymentAttempt persists the payment method and/or provider attempt
// reference chosen for a pending order.
func (s Store) UpdateOrderPaymentAttempt(ctx context.Context, db database.Executor, orderID, paymentMethod, paymentRef string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	setClauses := make([]string, 0, 3)
	args := make([]any, 0, 6)
	if paymentMethod != "" {
		args = append(args, paymentMethod)
		setClauses = append(setClauses, `payment_method = `+s.p(len(args)))
	}
	if paymentRef != "" {
		args = append(args, paymentRef)
		setClauses = append(setClauses, `payment_ref = `+s.p(len(args)))
	}
	if len(setClauses) == 0 {
		return nil
	}
	args = append(args, s.dbTime(s.now()))
	setClauses = append(setClauses, `updated_at = `+s.p(len(args)))
	args = append(args, orderID, string(commerce.PaymentPending))
	idPos, statusPos := len(args)-1, len(args)
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return err
	}
	query := `UPDATE commerce_customer_orders SET `
	for i, c := range setClauses {
		if i > 0 {
			query += `, `
		}
		query += c
	}
	query += ` WHERE id = ` + s.p(idPos) + ` AND payment_status = ` + s.p(statusPos) + filter
	_, err = db.ExecContext(ctx, query, args...)
	return err
}

// ExtendOrderHold pushes back hold_until on a pending order unless a later hold
// deadline is already stored.
func (s Store) ExtendOrderHold(ctx context.Context, db database.Executor, orderID string, holdUntil time.Time) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if holdUntil.IsZero() {
		return nil
	}
	holdVal := s.dbTime(holdUntil)
	args := []any{holdVal, s.dbTime(s.now()), orderID, string(commerce.PaymentPending), holdVal}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return err
	}
	query := `UPDATE commerce_customer_orders SET hold_until = ` + s.p(1) + `, updated_at = ` + s.p(2) +
		` WHERE id = ` + s.p(3) + ` AND payment_status = ` + s.p(4) +
		` AND (hold_until IS NULL OR hold_until < ` + s.p(5) + `)` + filter
	_, err = db.ExecContext(ctx, query, args...)
	return err
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
func (s Store) RecordShipment(ctx context.Context, db database.Executor, orderID string, shipment commerce.ShipmentResult, shippedAt time.Time) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	args := []any{string(commerce.FulfillmentShipped), shipment.Carrier, shipment.TrackingNumber, shipment.TrackingURL, shipment.ShipmentID, shipment.LabelURL, s.dbTime(shippedAt), s.dbTime(s.now()), orderID}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return false, err
	}
	query := `UPDATE commerce_customer_orders SET fulfillment_status = ` + s.p(1) + `, shipping_carrier = ` + s.p(2) +
		`, shipping_tracking_number = ` + s.p(3) + `, shipping_tracking_url = ` + s.p(4) +
		`, shipping_shipment_id = ` + s.p(5) + `, shipping_label_url = ` + s.p(6) +
		`, shipped_at = ` + s.p(7) + `, delivered_at = NULL, updated_at = ` + s.p(8) +
		` WHERE id = ` + s.p(9) + ` AND fulfillment_status = '` + string(commerce.FulfillmentPreparing) + `'` + filter
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

// GetOrderByShipmentID returns one order matching an external provider
// shipment identifier within the current tenant scope.
func (s Store) GetOrderByShipmentID(ctx context.Context, db database.Executor, shipmentID string) (commerce.Order, error) {
	if err := database.RequireExecutor(db); err != nil {
		return commerce.Order{}, err
	}
	id := strings.TrimSpace(shipmentID)
	if id == "" {
		return commerce.Order{}, commerce.ErrOrderNotFound
	}
	args := []any{id}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return commerce.Order{}, err
	}
	row := db.QueryRowContext(ctx, `SELECT `+orderColumns+` FROM commerce_customer_orders WHERE shipping_shipment_id = `+s.p(1)+filter, args...)
	order, found, err := s.scanOrder(row)
	if err != nil {
		return commerce.Order{}, err
	}
	if !found {
		return commerce.Order{}, commerce.ErrOrderNotFound
	}
	return order, nil
}

// UpdateShipmentTracking fills or refreshes carrier tracking fields on an order
// when non-empty values are supplied.
func (s Store) UpdateShipmentTracking(ctx context.Context, db database.Executor, orderID, trackingNumber, trackingURL, labelURL string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	setClauses := make([]string, 0, 4)
	args := make([]any, 0, 6)
	if strings.TrimSpace(trackingNumber) != "" {
		args = append(args, trackingNumber)
		setClauses = append(setClauses, `shipping_tracking_number = `+s.p(len(args)))
	}
	if strings.TrimSpace(trackingURL) != "" {
		args = append(args, trackingURL)
		setClauses = append(setClauses, `shipping_tracking_url = `+s.p(len(args)))
	}
	if strings.TrimSpace(labelURL) != "" {
		args = append(args, labelURL)
		setClauses = append(setClauses, `shipping_label_url = `+s.p(len(args)))
	}
	if len(setClauses) == 0 {
		return nil
	}
	args = append(args, s.dbTime(s.now()))
	setClauses = append(setClauses, `updated_at = `+s.p(len(args)))
	args = append(args, orderID)
	idPos := len(args)
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return err
	}
	query := `UPDATE commerce_customer_orders SET ` + strings.Join(setClauses, `, `) + ` WHERE id = ` + s.p(idPos) + filter
	_, err = db.ExecContext(ctx, query, args...)
	return err
}

// CancelShipmentRecord clears carrier shipment details and returns a shipped
// order to unfulfilled so an operator can create a replacement shipment.
func (s Store) CancelShipmentRecord(ctx context.Context, db database.Executor, orderID string) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	args := []any{string(commerce.FulfillmentUnfulfilled), s.dbTime(s.now()), orderID, string(commerce.FulfillmentShipped)}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return false, err
	}
	query := `UPDATE commerce_customer_orders SET fulfillment_status = ` + s.p(1) +
		`, shipping_carrier = '', shipping_tracking_number = '', shipping_tracking_url = '', shipping_shipment_id = '', shipping_label_url = '', shipped_at = NULL, delivered_at = NULL, updated_at = ` + s.p(2) +
		` WHERE id = ` + s.p(3) + ` AND fulfillment_status = ` + s.p(4) + filter
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

// MarkOrderDelivered atomically transitions a shipped order to delivered.
func (s Store) MarkOrderDelivered(ctx context.Context, db database.Executor, orderID string, deliveredAt time.Time) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	args := []any{string(commerce.FulfillmentDelivered), s.dbTime(deliveredAt), s.dbTime(s.now()), orderID, string(commerce.FulfillmentShipped)}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return false, err
	}
	query := `UPDATE commerce_customer_orders SET fulfillment_status = ` + s.p(1) +
		`, delivered_at = ` + s.p(2) + `, updated_at = ` + s.p(3) +
		` WHERE id = ` + s.p(4) + ` AND fulfillment_status = ` + s.p(5) + filter
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
