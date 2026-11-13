package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/modules/commerce"
)

// CreateDiscountCode inserts a promo code. Admin-authored, so no
// idempotency dance is needed here the way CreateOrder needs one -- a
// duplicate code is just a unique-constraint error the admin endpoint
// surfaces directly.
func (s Store) CreateDiscountCode(ctx context.Context, db database.Executor, code commerce.DiscountCode) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	tenantID, err := s.tenantID(ctx)
	if err != nil {
		return err
	}
	active := 0
	if code.Active {
		active = 1
	}
	now := s.now()
	query := `INSERT INTO commerce_discount_codes (id, code, kind, value, min_subtotal_cents, max_redemptions, redeemed_count, active, starts_at, ends_at, created_at, updated_at, tenant_id) VALUES (` +
		s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `,` + s.p(9) + `,` + s.p(10) + `,` + s.p(11) + `,` + s.p(12) + `,` + s.p(13) + `)`
	_, err = db.ExecContext(ctx, query, code.ID, code.Code, string(code.Kind), code.Value, code.MinSubtotalCents, code.MaxRedemptions, code.RedeemedCount, active,
		nullableTime(s, code.StartsAt), nullableTime(s, code.EndsAt), s.dbTime(now), s.dbTime(now), tenantID)
	return err
}

func nullableTime(s Store, t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return s.dbTime(t)
}

// GetDiscountCodeByCode looks up a promo code by its (case-sensitive,
// exact) code. Service.resolveDiscount normalizes casing before calling
// this, not the repository.
func (s Store) GetDiscountCodeByCode(ctx context.Context, db database.Executor, code string) (commerce.DiscountCode, error) {
	args := []any{code}
	filter, args, err := s.scope(ctx, args)
	if err != nil {
		return commerce.DiscountCode{}, err
	}
	row := db.QueryRowContext(ctx, `SELECT id, code, kind, value, min_subtotal_cents, max_redemptions, redeemed_count, active, starts_at, ends_at, created_at, updated_at FROM commerce_discount_codes WHERE UPPER(code) = UPPER(`+s.p(1)+`)`+filter, args...)
	discount, found, err := s.scanDiscountCode(row)
	if err != nil {
		return commerce.DiscountCode{}, err
	}
	if !found {
		return commerce.DiscountCode{}, commerce.ErrDiscountCodeNotFound
	}
	return discount, nil
}

func (s Store) scanDiscountCode(row interface{ Scan(...any) error }) (commerce.DiscountCode, bool, error) {
	var d commerce.DiscountCode
	var kind string
	var active int
	var startsAt, endsAt, created, updated any
	err := row.Scan(&d.ID, &d.Code, &kind, &d.Value, &d.MinSubtotalCents, &d.MaxRedemptions, &d.RedeemedCount, &active, &startsAt, &endsAt, &created, &updated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return commerce.DiscountCode{}, false, nil
		}
		return commerce.DiscountCode{}, false, err
	}
	d.Kind = commerce.DiscountKind(kind)
	d.Active = active != 0
	var parseErr error
	if startsAt != nil {
		if d.StartsAt, parseErr = s.parseTime(startsAt); parseErr != nil {
			return commerce.DiscountCode{}, false, parseErr
		}
	}
	if endsAt != nil {
		if d.EndsAt, parseErr = s.parseTime(endsAt); parseErr != nil {
			return commerce.DiscountCode{}, false, parseErr
		}
	}
	if d.CreatedAt, parseErr = s.parseTime(created); parseErr != nil {
		return commerce.DiscountCode{}, false, parseErr
	}
	if d.UpdatedAt, parseErr = s.parseTime(updated); parseErr != nil {
		return commerce.DiscountCode{}, false, parseErr
	}
	return d, true, nil
}

// RedeemDiscountCode increments redeemed_count by one conditional UPDATE --
// the same atomic-guard pattern ReserveStock uses for stock, and for the
// same reason: two checkouts racing the last redemption of a limited code
// must never both succeed. maxRedemptions is passed in (not re-read here)
// so the caller's own validated value is what the guard checks against;
// 0 means unlimited.
func (s Store) RedeemDiscountCode(ctx context.Context, db database.Executor, id string, maxRedemptions int) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	// maxRedemptions is bound twice (positions 3 and 4): SQLite's "?" has no
	// numbering, so one placeholder cannot be reused the way Postgres' $3 can.
	filter, args, err := s.scope(ctx, []any{s.dbTime(s.now()), id, maxRedemptions, maxRedemptions})
	if err != nil {
		return false, err
	}
	query := `UPDATE commerce_discount_codes SET redeemed_count = redeemed_count + 1, updated_at = ` + s.p(1) +
		` WHERE id = ` + s.p(2) + ` AND (` + s.p(3) + ` = 0 OR redeemed_count < ` + s.p(4) + `)` + filter
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

// ReleaseDiscountCode gives one redemption back (an order that used the code
// was abandoned before payment). Guarded so the counter never goes negative.
func (s Store) ReleaseDiscountCode(ctx context.Context, db database.Executor, code string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	filter, args, err := s.scope(ctx, []any{s.dbTime(s.now()), code})
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `UPDATE commerce_discount_codes SET redeemed_count = redeemed_count - 1, updated_at = `+s.p(1)+
		` WHERE UPPER(code) = UPPER(`+s.p(2)+`) AND redeemed_count > 0`+filter, args...)
	return err
}

const discountColumns = `id, code, kind, value, min_subtotal_cents, max_redemptions, redeemed_count, active, starts_at, ends_at, created_at, updated_at`

// ListDiscountCodes backs the admin coupon screen, newest first.
func (s Store) ListDiscountCodes(ctx context.Context, db database.Executor, limit, offset int) ([]commerce.DiscountCode, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("database executor does not support row queries")
	}
	filter, args, err := s.scope(ctx, nil)
	if err != nil {
		return nil, err
	}
	rows, err := queryer.QueryContext(ctx, `SELECT `+discountColumns+` FROM commerce_discount_codes WHERE 1=1`+filter+` ORDER BY created_at DESC LIMIT `+strconv.Itoa(limit)+` OFFSET `+strconv.Itoa(offset), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []commerce.DiscountCode{}
	for rows.Next() {
		d, _, err := s.scanDiscountCode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// SetDiscountCodeActive switches a code on or off; false when no such code exists.
func (s Store) SetDiscountCodeActive(ctx context.Context, db database.Executor, id string, active bool) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	flag := 0
	if active {
		flag = 1
	}
	filter, args, err := s.scope(ctx, []any{flag, s.dbTime(s.now()), id})
	if err != nil {
		return false, err
	}
	result, err := db.ExecContext(ctx, `UPDATE commerce_discount_codes SET active = `+s.p(1)+`, updated_at = `+s.p(2)+` WHERE id = `+s.p(3)+filter, args...)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}
