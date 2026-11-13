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

const defaultInvoiceStaleWindow = 5 * time.Minute

// ClaimInvoice atomically transitions a paid, uninvoiced order into
// invoice_status='pending' so concurrent admin clicks or background workers
// never double-issue an external invoice. Stale pending claims older than
// staleBefore can be reclaimed.
func (s Store) ClaimInvoice(ctx context.Context, db database.Executor, orderID, provider string, claimedAt, staleBefore time.Time) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	if claimedAt.IsZero() {
		claimedAt = s.now()
	}
	if staleBefore.IsZero() {
		staleBefore = claimedAt.Add(-defaultInvoiceStaleWindow)
	}
	args := []any{
		string(commerce.InvoicePending),
		strings.TrimSpace(provider),
		s.dbTime(claimedAt),
		s.dbTime(claimedAt),
		orderID,
		string(commerce.PaymentPaid),
		string(commerce.InvoiceNone),
		string(commerce.InvoiceFailed),
		string(commerce.InvoicePending),
		s.dbTime(staleBefore),
	}
	filter, scopedArgs, err := s.scope(ctx, args)
	if err != nil {
		return false, err
	}
	query := `UPDATE commerce_customer_orders
		SET invoice_status = ` + s.p(1) + `,
		    invoice_provider = ` + s.p(2) + `,
		    invoice_claimed_at = ` + s.p(3) + `,
		    invoice_error = '',
		    updated_at = ` + s.p(4) + `
		WHERE id = ` + s.p(5) + `
		  AND payment_status = ` + s.p(6) + `
		  AND total_cents > 0
		  AND (
		    invoice_status = ''
		    OR invoice_status IN (` + s.p(7) + `, ` + s.p(8) + `)
		    OR (invoice_status = ` + s.p(9) + ` AND (invoice_claimed_at IS NULL OR invoice_claimed_at < ` + s.p(10) + `))
		  )` + filter
	res, err := db.ExecContext(ctx, query, scopedArgs...)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// RecordInvoiceIssued saves the external invoice metadata once issued and marks
// invoice_status='issued'. Only a currently pending claim may transition to
// issued so late stale attempts cannot clobber a finalized invoice.
func (s Store) RecordInvoiceIssued(ctx context.Context, db database.Executor, orderID string, result commerce.InvoiceResult, invoicedAt time.Time) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	if invoicedAt.IsZero() {
		invoicedAt = s.now()
	}
	args := []any{
		string(commerce.InvoiceIssued),
		strings.TrimSpace(result.Provider),
		strings.TrimSpace(result.GUID),
		strings.TrimSpace(result.InvoiceNo),
		strings.TrimSpace(result.URL),
		s.dbTime(invoicedAt),
		s.dbTime(invoicedAt),
		orderID,
		string(commerce.InvoicePending),
	}
	filter, scopedArgs, err := s.scope(ctx, args)
	if err != nil {
		return false, err
	}
	query := `UPDATE commerce_customer_orders
		SET invoice_status = ` + s.p(1) + `,
		    invoice_provider = ` + s.p(2) + `,
		    invoice_guid = ` + s.p(3) + `,
		    invoice_no = ` + s.p(4) + `,
		    invoice_url = ` + s.p(5) + `,
		    invoice_error = '',
		    invoiced_at = ` + s.p(6) + `,
		    updated_at = ` + s.p(7) + `
		WHERE id = ` + s.p(8) + `
		  AND invoice_status = ` + s.p(9) + filter
	res, err := db.ExecContext(ctx, query, scopedArgs...)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// RecordInvoiceFailed marks a pending invoice claim as failed with a bounded
// diagnostic message so operators can inspect the failure and retry manually.
func (s Store) RecordInvoiceFailed(ctx context.Context, db database.Executor, orderID, errMsg string, failedAt time.Time) (bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, err
	}
	if failedAt.IsZero() {
		failedAt = s.now()
	}
	msg := trimRunes(strings.TrimSpace(errMsg), 500)
	args := []any{
		string(commerce.InvoiceFailed),
		msg,
		s.dbTime(failedAt),
		orderID,
		string(commerce.InvoicePending),
	}
	filter, scopedArgs, err := s.scope(ctx, args)
	if err != nil {
		return false, err
	}
	query := `UPDATE commerce_customer_orders
		SET invoice_status = ` + s.p(1) + `,
		    invoice_error = ` + s.p(2) + `,
		    invoice_claimed_at = NULL,
		    updated_at = ` + s.p(3) + `
		WHERE id = ` + s.p(4) + `
		  AND invoice_status = ` + s.p(5) + filter
	res, err := db.ExecContext(ctx, query, scopedArgs...)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ListOrdersDueForAutoInvoice returns paid orders with positive totals whose
// invoice_status is still uninitialized ('none' or ”) and whose updated_at is
// at or before paidBefore (allowing post-checkout tax details to settle).
func (s Store) ListOrdersDueForAutoInvoice(ctx context.Context, db database.Executor, paidBefore time.Time, limit int) ([]commerce.Order, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	if paidBefore.IsZero() {
		paidBefore = s.now()
	}
	if limit <= 0 {
		limit = 25
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("database executor does not support querying rows")
	}
	args := []any{
		string(commerce.PaymentPaid),
		string(commerce.InvoiceNone),
		s.dbTime(paidBefore),
	}
	filter, scopedArgs, err := s.scope(ctx, args)
	if err != nil {
		return nil, err
	}
	query := `SELECT ` + orderColumns + `
		FROM commerce_customer_orders
		WHERE payment_status = ` + s.p(1) + `
		  AND total_cents > 0
		  AND (invoice_status = ` + s.p(2) + ` OR invoice_status = '')
		  AND updated_at <= ` + s.p(3) + filter + `
		ORDER BY created_at ASC
		LIMIT ` + fmt.Sprintf("%d", limit)
	rows, err := queryer.QueryContext(ctx, query, scopedArgs...)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

func trimRunes(s string, max int) string {
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}
