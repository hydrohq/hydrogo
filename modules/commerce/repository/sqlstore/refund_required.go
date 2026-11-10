package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

// RecordRefundRequired saves a manual refund requirement row in the database.
// The unique constraint on (order_id, provider) ensures repeated callbacks do not create duplicate records.
func (s Store) RecordRefundRequired(ctx context.Context, db database.Executor, item commerce.RefundRequired) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	tenantID, err := s.tenantID(ctx)
	if err != nil {
		return err
	}
	createdAt := s.dbTime(item.CreatedAt)
	if item.CreatedAt.IsZero() {
		createdAt = s.dbTime(s.now())
	}
	query := `INSERT INTO refund_required (id, order_id, provider, amount_cents, currency, reason, created_at, resolved_note, tenant_id) VALUES (` +
		s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `,` + s.p(9) + `)`
	_, err = db.ExecContext(ctx, query, item.ID, item.OrderID, item.Provider, item.AmountCents, item.Currency, item.Reason, createdAt, item.ResolvedNote, tenantID)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil
		}
		return err
	}
	return nil
}

// ListRefundRequired returns recorded refund requirements, optionally filtering for unresolved items.
func (s Store) ListRefundRequired(ctx context.Context, db database.Executor, unresolvedOnly bool, limit, offset int) ([]commerce.RefundRequired, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	tenantID, err := s.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("database executor does not support querying rows")
	}

	query := `SELECT id, order_id, provider, amount_cents, currency, reason, created_at, resolved_at, resolved_note FROM refund_required WHERE 1=1`
	var args []any
	argIdx := 1
	if s.RequireTenantScope {
		query += ` AND tenant_id = ` + s.p(argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if unresolvedOnly {
		query += ` AND resolved_at IS NULL`
	}
	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		query += ` LIMIT ` + s.p(argIdx)
		args = append(args, limit)
		argIdx++
		if offset > 0 {
			query += ` OFFSET ` + s.p(argIdx)
			args = append(args, offset)
			argIdx++
		}
	}

	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []commerce.RefundRequired
	for rows.Next() {
		var (
			r             commerce.RefundRequired
			createdAtVal  any
			resolvedAtVal any
		)
		if err := rows.Scan(&r.ID, &r.OrderID, &r.Provider, &r.AmountCents, &r.Currency, &r.Reason, &createdAtVal, &resolvedAtVal, &r.ResolvedNote); err != nil {
			return nil, err
		}
		if createdAtVal != nil {
			if t, err := s.parseTime(createdAtVal); err == nil {
				r.CreatedAt = t
			}
		}
		if resolvedAtVal != nil {
			if t, err := s.parseTime(resolvedAtVal); err == nil {
				r.ResolvedAt = &t
			}
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// ResolveRefundRequired marks a refund requirement as resolved.
func (s Store) ResolveRefundRequired(ctx context.Context, db database.Executor, id, note string, resolvedAt time.Time) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	tenantID, err := s.tenantID(ctx)
	if err != nil {
		return false, err
	}
	resTime := s.dbTime(resolvedAt)
	args := []any{resTime, note, id}
	query := `UPDATE refund_required SET resolved_at = ` + s.p(1) + `, resolved_note = ` + s.p(2) + ` WHERE id = ` + s.p(3) + ` AND resolved_at IS NULL`
	if s.RequireTenantScope {
		query += ` AND tenant_id = ` + s.p(4)
		args = append(args, tenantID)
	}
	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// RecordSMSDedup inserts a customer SMS deduplication row. Returns false if already recorded.
func (s Store) RecordSMSDedup(ctx context.Context, db database.Executor, dedupKey, orderID, phone, event string, sentAt time.Time) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	tenantID, err := s.tenantID(ctx)
	if err != nil {
		return false, err
	}
	now := s.dbTime(sentAt)
	id := "sms_" + dedupKey
	query := `INSERT INTO customer_sms_dedup (id, order_id, phone, event, dedup_key, sent_at, tenant_id) VALUES (` +
		s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `)`
	_, err = db.ExecContext(ctx, query, id, orderID, phone, event, dedupKey, now, tenantID)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
