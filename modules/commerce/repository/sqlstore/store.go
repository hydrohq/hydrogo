// Package sqlstore is modules/commerce's persistence layer: stock
// reservation, carts and orders. Dialect SQL stays here, never in service.go
// (AGENTS.md rule 4). Table names are deliberately distinct from the old,
// removed commerce module's (commerce_shopping_carts, not commerce_carts;
// commerce_customer_orders, not commerce_orders; commerce_payment_charges /
// commerce_payment_notifications, not commerce_payment_transactions /
// commerce_payment_webhook_events) -- the old module's migrations are still
// on disk and still baselined, so a fresh database enabling this module
// applies both; reusing its table names would silently collide. See
// migrations/202610231300_commerce_catalog_stock and siblings.
package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

type Store struct {
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s Store) p(n int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func (s Store) now() time.Time { return time.Now().UTC() }

// dbTime formats a time for storage the way the rest of the codebase does
// (internal/jobs/sqlstore.Store.dbTime): SQLite/Postgres store RFC3339Nano
// text, MySQL's driver accepts a time.Time directly for DATETIME columns.
func (s Store) dbTime(t time.Time) any {
	if s.Dialect == database.MySQL {
		return t.UTC()
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func (s Store) parseTime(value any) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v.UTC(), nil
	case string:
		return time.Parse(time.RFC3339Nano, v)
	case []byte:
		return time.Parse(time.RFC3339Nano, string(v))
	default:
		return time.Time{}, fmt.Errorf("unsupported stored time value type %T", value)
	}
}

func (s Store) tenantID(ctx context.Context) (string, error) {
	if !s.RequireTenantScope {
		return "", nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", err
	}
	return scope.TenantID, nil
}

func (s Store) tenantFilter(ctx context.Context, position int) (string, any, error) {
	id, err := s.tenantID(ctx)
	if err != nil {
		return "", nil, err
	}
	if !s.RequireTenantScope {
		return "", nil, nil
	}
	return " AND tenant_id = " + s.p(position), id, nil
}

func newID(prefix string) (string, error) {
	raw := make([]byte, 15)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(raw), nil
}

// --- Stock -------------------------------------------------------------

// EnsureStock sets a variant's available-to-sell count, creating the row if
// none exists yet. It is not meant for the hot path (it's admin
// seeding/restocking); ReserveStock is the only method that needs to be
// atomic under contention. It never touches reserved: an admin correcting
// the available count mid-sale must not silently wipe out a shopper's
// in-flight reservation.
func (s Store) EnsureStock(ctx context.Context, db database.Executor, variantID string, available int) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	tenantID, err := s.tenantID(ctx)
	if err != nil {
		return err
	}
	_, err = s.GetStock(ctx, db, variantID)
	if err == nil {
		filter, filterArg, filterErr := s.tenantFilter(ctx, 3)
		if filterErr != nil {
			return filterErr
		}
		args := []any{available, s.dbTime(s.now()), variantID}
		if filterArg != nil {
			args = append(args, filterArg)
		}
		_, err = db.ExecContext(ctx, `UPDATE commerce_stock SET available = `+s.p(1)+`, updated_at = `+s.p(2)+` WHERE variant_id = `+s.p(3)+filter, args...)
		return err
	}
	if !errors.Is(err, commerce.ErrItemNotFound) {
		return err
	}
	id, err := newID("stk_")
	if err != nil {
		return err
	}
	now := s.now()
	query := `INSERT INTO commerce_stock (id, variant_id, available, reserved, created_at, updated_at, tenant_id) VALUES (` +
		s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,0,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `)`
	_, err = db.ExecContext(ctx, query, id, variantID, available, s.dbTime(now), s.dbTime(now), tenantID)
	return err
}

// GetStock returns the current stock row for a variant.
func (s Store) GetStock(ctx context.Context, db database.Executor, variantID string) (commerce.Stock, error) {
	if err := database.RequireExecutor(db); err != nil {
		return commerce.Stock{}, err
	}
	filter, filterArg, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return commerce.Stock{}, err
	}
	args := []any{variantID}
	if filterArg != nil {
		args = append(args, filterArg)
	}
	row := db.QueryRowContext(ctx, `SELECT id, variant_id, available, reserved, created_at, updated_at FROM commerce_stock WHERE variant_id = `+s.p(1)+filter, args...)
	var stock commerce.Stock
	var created, updated any
	if err := row.Scan(&stock.ID, &stock.VariantID, &stock.Available, &stock.Reserved, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return commerce.Stock{}, commerce.ErrItemNotFound
		}
		return commerce.Stock{}, err
	}
	if stock.CreatedAt, err = s.parseTime(created); err != nil {
		return commerce.Stock{}, err
	}
	if stock.UpdatedAt, err = s.parseTime(updated); err != nil {
		return commerce.Stock{}, err
	}
	return stock, nil
}

// ReserveStock is the method this module exists for: a single conditional
// UPDATE that only succeeds if enough stock is available, so two requests
// racing for the last unit can never both succeed and available can never
// go negative. No optimistic-lock retry loop, no application-level mutex --
// the database's own row lock during the UPDATE is the only synchronization
// needed.
func (s Store) ReserveStock(ctx context.Context, db database.Executor, variantID string, quantity int) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	if quantity <= 0 {
		return false, errors.New("reservation quantity must be positive")
	}
	filter, filterArg, err := s.tenantFilter(ctx, 4)
	if err != nil {
		return false, err
	}
	args := []any{quantity, quantity, s.dbTime(s.now()), variantID, quantity}
	// updated_at placeholder sits before variant_id/quantity in the SET
	// clause; renumber positions explicitly per dialect below instead of
	// relying on args order matching s.p(n) call order.
	query := `UPDATE commerce_stock SET available = available - ` + s.p(1) + `, reserved = reserved + ` + s.p(2) + `, updated_at = ` + s.p(3) +
		` WHERE variant_id = ` + s.p(4) + ` AND available >= ` + s.p(5) + filter
	if filterArg != nil {
		args = append(args, filterArg)
	}
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

// ReleaseReservation returns reserved stock to available -- used when a
// cart's hold expires (worker.go) or an order is cancelled before payment.
// Unlike ReserveStock, this never needs a conditional guard: releasing is
// always valid once a reservation is known to exist.
func (s Store) ReleaseReservation(ctx context.Context, db database.Executor, variantID string, quantity int) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if quantity <= 0 {
		return errors.New("release quantity must be positive")
	}
	filter, filterArg, err := s.tenantFilter(ctx, 4)
	if err != nil {
		return err
	}
	args := []any{quantity, quantity, s.dbTime(s.now()), variantID}
	if filterArg != nil {
		args = append(args, filterArg)
	}
	query := `UPDATE commerce_stock SET available = available + ` + s.p(1) + `, reserved = reserved - ` + s.p(2) + `, updated_at = ` + s.p(3) +
		` WHERE variant_id = ` + s.p(4) + filter
	_, err = db.ExecContext(ctx, query, args...)
	return err
}

// CommitReservation finalizes a sale: available was already decremented at
// ReserveStock time, so a successful payment only needs to drop the
// reservation hold, not touch available again.
func (s Store) CommitReservation(ctx context.Context, db database.Executor, variantID string, quantity int) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if quantity <= 0 {
		return errors.New("commit quantity must be positive")
	}
	filter, filterArg, err := s.tenantFilter(ctx, 3)
	if err != nil {
		return err
	}
	args := []any{quantity, s.dbTime(s.now()), variantID}
	if filterArg != nil {
		args = append(args, filterArg)
	}
	query := `UPDATE commerce_stock SET reserved = reserved - ` + s.p(1) + `, updated_at = ` + s.p(2) + ` WHERE variant_id = ` + s.p(3) + filter
	_, err = db.ExecContext(ctx, query, args...)
	return err
}

// --- Carts ---------------------------------------------------------------

func (s Store) CreateCart(ctx context.Context, db database.Executor, cart commerce.Cart) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if err := commerce.ValidateCart(cart); err != nil {
		return err
	}
	tenantID, err := s.tenantID(ctx)
	if err != nil {
		return err
	}
	memberID, anonymousToken := nullable(cart.MemberID), nullable(cart.AnonymousToken)
	query := `INSERT INTO commerce_shopping_carts (id, member_id, anonymous_token, currency, created_at, updated_at, expires_at, tenant_id) VALUES (` +
		s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `)`
	_, err = db.ExecContext(ctx, query, cart.ID, memberID, anonymousToken, cart.Currency, s.dbTime(cart.CreatedAt), s.dbTime(cart.UpdatedAt), s.dbTime(cart.ExpiresAt), tenantID)
	return err
}

func (s Store) getCartBy(ctx context.Context, db database.Executor, column, value string) (commerce.Cart, bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return commerce.Cart{}, false, err
	}
	filter, filterArg, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return commerce.Cart{}, false, err
	}
	args := []any{value}
	if filterArg != nil {
		args = append(args, filterArg)
	}
	row := db.QueryRowContext(ctx, `SELECT id, COALESCE(member_id, ''), COALESCE(anonymous_token, ''), currency, created_at, updated_at, expires_at FROM commerce_shopping_carts WHERE `+column+` = `+s.p(1)+filter, args...)
	var cart commerce.Cart
	var created, updated, expires any
	if err := row.Scan(&cart.ID, &cart.MemberID, &cart.AnonymousToken, &cart.Currency, &created, &updated, &expires); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return commerce.Cart{}, false, nil
		}
		return commerce.Cart{}, false, err
	}
	var parseErr error
	if cart.CreatedAt, parseErr = s.parseTime(created); parseErr != nil {
		return commerce.Cart{}, false, parseErr
	}
	if cart.UpdatedAt, parseErr = s.parseTime(updated); parseErr != nil {
		return commerce.Cart{}, false, parseErr
	}
	if cart.ExpiresAt, parseErr = s.parseTime(expires); parseErr != nil {
		return commerce.Cart{}, false, parseErr
	}
	return cart, true, nil
}

func (s Store) GetCartByMember(ctx context.Context, db database.Executor, memberID string) (commerce.Cart, bool, error) {
	return s.getCartBy(ctx, db, "member_id", memberID)
}

func (s Store) GetCartByAnonymousToken(ctx context.Context, db database.Executor, token string) (commerce.Cart, bool, error) {
	return s.getCartBy(ctx, db, "anonymous_token", token)
}

// AddOrUpdateItem upserts a cart line item by (cart_id, variant_id): adding
// an item already in the cart updates its quantity rather than duplicating
// the row, matching the migration's UNIQUE(cart_id, variant_id).
func (s Store) AddOrUpdateItem(ctx context.Context, db database.Executor, item commerce.CartItem) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if err := commerce.ValidateItem(item); err != nil {
		return err
	}
	tenantID, err := s.tenantID(ctx)
	if err != nil {
		return err
	}
	switch s.Dialect {
	case database.MySQL:
		upsert := `INSERT INTO commerce_shopping_cart_items (id, cart_id, variant_id, quantity, unit_price_cents, created_at, updated_at, tenant_id) VALUES (?,?,?,?,?,?,?,?) ` +
			`ON DUPLICATE KEY UPDATE quantity = VALUES(quantity), unit_price_cents = VALUES(unit_price_cents), updated_at = VALUES(updated_at)`
		_, err = db.ExecContext(ctx, upsert, item.ID, item.CartID, item.VariantID, item.Quantity, item.UnitPriceCents, s.dbTime(item.CreatedAt), s.dbTime(item.UpdatedAt), tenantID)
	case database.Postgres:
		upsert := `INSERT INTO commerce_shopping_cart_items (id, cart_id, variant_id, quantity, unit_price_cents, created_at, updated_at, tenant_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ` +
			`ON CONFLICT (cart_id, variant_id) DO UPDATE SET quantity = EXCLUDED.quantity, unit_price_cents = EXCLUDED.unit_price_cents, updated_at = EXCLUDED.updated_at`
		_, err = db.ExecContext(ctx, upsert, item.ID, item.CartID, item.VariantID, item.Quantity, item.UnitPriceCents, s.dbTime(item.CreatedAt), s.dbTime(item.UpdatedAt), tenantID)
	default: // SQLite
		upsert := `INSERT INTO commerce_shopping_cart_items (id, cart_id, variant_id, quantity, unit_price_cents, created_at, updated_at, tenant_id) VALUES (?,?,?,?,?,?,?,?) ` +
			`ON CONFLICT (cart_id, variant_id) DO UPDATE SET quantity = excluded.quantity, unit_price_cents = excluded.unit_price_cents, updated_at = excluded.updated_at`
		_, err = db.ExecContext(ctx, upsert, item.ID, item.CartID, item.VariantID, item.Quantity, item.UnitPriceCents, s.dbTime(item.CreatedAt), s.dbTime(item.UpdatedAt), tenantID)
	}
	return err
}

func (s Store) RemoveItem(ctx context.Context, db database.Executor, cartID, itemID string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, `DELETE FROM commerce_shopping_cart_items WHERE id = `+s.p(1)+` AND cart_id = `+s.p(2), itemID, cartID)
	return err
}

func (s Store) ListItems(ctx context.Context, db database.Executor, cartID string) ([]commerce.CartItem, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("database executor does not support row queries")
	}
	rows, err := queryer.QueryContext(ctx, `SELECT id, cart_id, variant_id, quantity, unit_price_cents, created_at, updated_at FROM commerce_shopping_cart_items WHERE cart_id = `+s.p(1)+` ORDER BY id`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []commerce.CartItem
	for rows.Next() {
		var item commerce.CartItem
		var created, updated any
		if err := rows.Scan(&item.ID, &item.CartID, &item.VariantID, &item.Quantity, &item.UnitPriceCents, &created, &updated); err != nil {
			return nil, err
		}
		if item.CreatedAt, err = s.parseTime(created); err != nil {
			return nil, err
		}
		if item.UpdatedAt, err = s.parseTime(updated); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

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
	query := `INSERT INTO commerce_customer_orders (id, member_id, guest_name, guest_email, guest_phone, guest_address, guest_city, guest_country, guest_postal_code, idempotency_key, idempotency_hash, currency, subtotal_cents, discount_code, discount_cents, tax_cents, shipping_cents, total_cents, payment_status, fulfillment_status, created_at, updated_at, tenant_id) VALUES (` +
		s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `,` + s.p(9) + `,` + s.p(10) + `,` + s.p(11) + `,` + s.p(12) + `,` + s.p(13) + `,` + s.p(14) + `,` + s.p(15) + `,` + s.p(16) + `,` + s.p(17) + `,` + s.p(18) + `,` + s.p(19) + `,` + s.p(20) + `,` + s.p(21) + `,` + s.p(22) + `,` + s.p(23) + `)`
	_, err = db.ExecContext(ctx, query, order.ID, memberID, order.GuestName, order.GuestEmail, order.GuestPhone, order.GuestAddress, order.GuestCity, order.GuestCountry, order.GuestPostalCode,
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
	filter, filterArg, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return commerce.Order{}, false, err
	}
	args := []any{key}
	if filterArg != nil {
		args = append(args, filterArg)
	}
	row := db.QueryRowContext(ctx, `SELECT `+orderColumns+` FROM commerce_customer_orders WHERE idempotency_key = `+s.p(1)+filter, args...)
	return s.scanOrder(row)
}

// GetOrder returns one order by id.
func (s Store) GetOrder(ctx context.Context, db database.Executor, id string) (commerce.Order, error) {
	filter, filterArg, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return commerce.Order{}, err
	}
	args := []any{id}
	if filterArg != nil {
		args = append(args, filterArg)
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
	rows, err := queryer.QueryContext(ctx, `SELECT id, order_id, variant_id, sku, title, quantity, unit_price_cents, total_cents FROM commerce_customer_order_items WHERE order_id = `+s.p(1)+` ORDER BY id`, orderID)
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
	var created, updated any
	if err := scanner.Scan(&order.ID, &order.MemberID, &order.GuestName, &order.GuestEmail, &order.GuestPhone, &order.GuestAddress, &order.GuestCity, &order.GuestCountry, &order.GuestPostalCode,
		&order.IdempotencyKey, &order.IdempotencyHash, &order.Currency, &order.SubtotalCents, &order.DiscountCode, &order.DiscountCents, &order.TaxCents, &order.ShippingCents, &order.TotalCents,
		&paymentStatus, &fulfillmentStatus, &created, &updated); err != nil {
		return commerce.Order{}, err
	}
	order.PaymentStatus = commerce.PaymentStatus(paymentStatus)
	order.FulfillmentStatus = commerce.FulfillmentStatus(fulfillmentStatus)
	var parseErr error
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

const orderColumns = `id, COALESCE(member_id,''), guest_name, guest_email, guest_phone, guest_address, guest_city, guest_country, guest_postal_code, idempotency_key, idempotency_hash, currency, subtotal_cents, discount_code, discount_cents, tax_cents, shipping_cents, total_cents, payment_status, fulfillment_status, created_at, updated_at`

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
	filter, filterArg, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return nil, err
	}
	args := []any{s.dbTime(cutoff)}
	if filterArg != nil {
		args = append(args, filterArg)
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
	filter, filterArg, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return nil, err
	}
	args := []any{memberID}
	if filterArg != nil {
		args = append(args, filterArg)
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

// GetOrderByMemberID returns one order, scoped to the member who placed it --
// ErrOrderNotFound both when the order does not exist and when it belongs to
// someone else, so the member surface can't distinguish the two cases.
func (s Store) GetOrderByMemberID(ctx context.Context, db database.Executor, memberID, orderID string) (commerce.Order, error) {
	filter, filterArg, err := s.tenantFilter(ctx, 3)
	if err != nil {
		return commerce.Order{}, err
	}
	args := []any{orderID, memberID}
	if filterArg != nil {
		args = append(args, filterArg)
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
	filter, filterArg, err := s.tenantFilter(ctx, 5)
	if err != nil {
		return false, err
	}
	args := []any{string(to), s.dbTime(s.now()), orderID, string(from)}
	if filterArg != nil {
		args = append(args, filterArg)
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

// --- Payments ----------------------------------------------------------

// RecordPaymentNotification inserts a (provider, provider_event_id) dedup
// row. It returns false (not an error) when that exact event was already
// recorded, since a provider redelivering the same notification is normal,
// documented behaviour (see PayTR's own docs), not a failure.
func (s Store) RecordPaymentNotification(ctx context.Context, db database.Executor, provider, eventID string) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	tenantID, err := s.tenantID(ctx)
	if err != nil {
		return false, err
	}
	query := `INSERT INTO commerce_payment_notifications (provider, provider_event_id, received_at, tenant_id) VALUES (` +
		s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `)`
	_, err = db.ExecContext(ctx, query, provider, eventID, s.dbTime(s.now()), tenantID)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}
