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

// scope returns the tenant condition bound to the placeholder right after
// args, and args with the tenant id appended. Callers must not pick the
// position by hand: a hardcoded number that collides with an earlier
// placeholder is silently wrong on Postgres ($n) and fine on SQLite (?).
func (s Store) scope(ctx context.Context, args []any) (string, []any, error) {
	filter, arg, err := s.tenantFilter(ctx, len(args)+1)
	if err != nil {
		return "", nil, err
	}
	if arg != nil {
		args = append(args, arg)
	}
	return filter, args, nil
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
		filter, args, filterErr := s.scope(ctx, []any{available, s.dbTime(s.now()), variantID})
		if filterErr != nil {
			return filterErr
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
	args := []any{variantID}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return commerce.Stock{}, err
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
	filter, args, err := s.scope(ctx, []any{quantity, quantity, s.dbTime(s.now()), variantID, quantity})
	if err != nil {
		return false, err
	}
	query := `UPDATE commerce_stock SET available = available - ` + s.p(1) + `, reserved = reserved + ` + s.p(2) + `, updated_at = ` + s.p(3) +
		` WHERE variant_id = ` + s.p(4) + ` AND available >= ` + s.p(5) + filter
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
	args := []any{quantity, quantity, s.dbTime(s.now()), variantID}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return err
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
	args := []any{quantity, s.dbTime(s.now()), variantID}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return err
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
	args := []any{value}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return commerce.Cart{}, false, err
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
	filter, args, err := s.scope(ctx, []any{itemID, cartID})
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `DELETE FROM commerce_shopping_cart_items WHERE id = `+s.p(1)+` AND cart_id = `+s.p(2)+filter, args...)
	return err
}

// ClearCart empties a cart once its contents became an order.
func (s Store) ClearCart(ctx context.Context, db database.Executor, cartID string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	filter, args, err := s.scope(ctx, []any{cartID})
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `DELETE FROM commerce_shopping_cart_items WHERE cart_id = `+s.p(1)+filter, args...)
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
	filter, args, err := s.scope(ctx, []any{cartID})
	if err != nil {
		return nil, err
	}
	rows, err := queryer.QueryContext(ctx, `SELECT id, cart_id, variant_id, quantity, unit_price_cents, created_at, updated_at FROM commerce_shopping_cart_items WHERE cart_id = `+s.p(1)+filter+` ORDER BY id`, args...)
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

// --- Payments ----------------------------------------------------------

// RecordPaymentNotification inserts a (provider, provider_event_id) dedup
// row. It returns false (not an error) when that exact event was already
// recorded, since a provider redelivering the same notification is normal
// documented behaviour (e.g. webhook retries), not a failure.
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
