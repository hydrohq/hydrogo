package sqlstore

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

var (
	ErrSpoolFull   = errors.New("mail spool capacity reached")
	ErrSpoolLease  = errors.New("mail spool lease is no longer current")
	ErrSpoolRecord = errors.New("mail spool record conflict")
)

type Spool struct {
	DB                 database.Executor
	Tx                 database.Transactor
	Dialect            database.Dialect
	RequireTenantScope bool
	MaxPending         int
}

func (s Spool) p(n int) string {
	if s.Dialect == database.SQLite {
		return "?"
	}
	return fmt.Sprintf("$%d", n)
}

func (s Spool) tenant(ctx context.Context) (string, error) {
	if !s.RequireTenantScope {
		return "", nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", err
	}
	return scope.TenantID, nil
}

// Enqueue participates in the form submission transaction. event_key makes
// retries safe when a caller repeats the same transaction intent.
func (s Spool) Enqueue(ctx context.Context, tx database.Executor, notification site.EmailNotification) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	if strings.TrimSpace(notification.EventKey) == "" || len(notification.EventKey) > 256 || len(notification.Subject) == 0 || len(notification.Subject) > 998 || strings.ContainsAny(notification.Subject, "\r\n") || len(notification.Body) > 1<<20 || len(notification.HTMLBody) > 1<<20 {
		return mailer.ErrInvalidMessage
	}
	recipient := strings.TrimSpace(notification.To)
	parsed, err := mail.ParseAddress(recipient)
	if err != nil || parsed.Address != recipient || len(recipient) > 320 {
		return mailer.ErrInvalidMessage
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	id := spoolID(tenantID, notification.EventKey)
	var existingRecipient, existingSubject, existingBody, existingHTML string
	lookup := `SELECT recipient,subject,body,html_body FROM mail_spool WHERE tenant_id = ` + s.p(1) + ` AND event_key = ` + s.p(2)
	err = tx.QueryRowContext(ctx, lookup, tenantID, notification.EventKey).Scan(&existingRecipient, &existingSubject, &existingBody, &existingHTML)
	if err == nil {
		if existingRecipient != recipient || existingSubject != notification.Subject || existingBody != notification.Body || existingHTML != notification.HTMLBody {
			return ErrSpoolRecord
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	maxPending := s.MaxPending
	if maxPending < 1 {
		maxPending = 10000
	}
	var active int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM mail_spool WHERE tenant_id = `+s.p(1)+` AND status IN ('pending','sending')`, tenantID).Scan(&active); err != nil {
		return err
	}
	if active >= maxPending {
		return ErrSpoolFull
	}
	now := time.Now().UTC()
	query := `INSERT INTO mail_spool (id,tenant_id,event_key,recipient,subject,body,html_body,status,attempts,next_attempt_at,last_error,created_at,updated_at) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `,'pending',0,` + s.p(8) + `,'',` + s.p(9) + `,` + s.p(10) + `) ON CONFLICT (tenant_id,event_key) DO NOTHING`
	result, err := tx.ExecContext(ctx, query, id, tenantID, notification.EventKey, recipient, notification.Subject, notification.Body, notification.HTMLBody, spoolTime(s.Dialect, now), spoolTime(s.Dialect, now), spoolTime(s.Dialect, now))
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil || inserted == 1 {
		return err
	}
	if err := tx.QueryRowContext(ctx, lookup, tenantID, notification.EventKey).Scan(&existingRecipient, &existingSubject, &existingBody, &existingHTML); err != nil {
		return err
	}
	if existingRecipient != recipient || existingSubject != notification.Subject || existingBody != notification.Body || existingHTML != notification.HTMLBody {
		return ErrSpoolRecord
	}
	return nil
}

func (s Spool) Claim(ctx context.Context, worker string, now time.Time, lease time.Duration) (mailer.SpoolEntry, string, bool, error) {
	if err := database.RequireExecutor(s.DB); err != nil {
		return mailer.SpoolEntry{}, "", false, err
	}
	if strings.TrimSpace(worker) == "" || len(worker) > 128 || lease <= 0 {
		return mailer.SpoolEntry{}, "", false, nil
	}
	var result mailer.SpoolEntry
	var token string
	claimed := false
	transactor := s.Tx
	if transactor == (database.Transactor{}) {
		if db, ok := s.DB.(*sql.DB); ok {
			transactor = database.NewTenantTransactor(db, s.Dialect, s.RequireTenantScope)
		} else {
			return result, "", false, errors.New("mail spool transaction is required")
		}
	}
	err := transactor.Within(ctx, func(tx database.Executor) error {
		tenantID, err := s.tenant(ctx)
		if err != nil {
			return err
		}
		var query string
		args := []any{tenantID, spoolTime(s.Dialect, now), spoolTime(s.Dialect, now)}
		if s.Dialect == database.Postgres {
			query = `SELECT id,event_key,recipient,subject,body,html_body,status,attempts,next_attempt_at,created_at,updated_at,COALESCE(last_error,'') FROM mail_spool WHERE tenant_id=$1 AND ((status='pending' AND next_attempt_at <= $2) OR (status='sending' AND lease_until <= $3)) ORDER BY next_attempt_at,id LIMIT 1 FOR UPDATE SKIP LOCKED`
		} else {
			query = `SELECT id,event_key,recipient,subject,body,html_body,status,attempts,next_attempt_at,created_at,updated_at,COALESCE(last_error,'') FROM mail_spool WHERE tenant_id=? AND ((status='pending' AND next_attempt_at <= ?) OR (status='sending' AND lease_until <= ?)) ORDER BY next_attempt_at,id LIMIT 1`
		}
		var next, created, updated any
		err = tx.QueryRowContext(ctx, query, args...).Scan(&result.ID, &result.EventKey, &result.Message.To, &result.Message.Subject, &result.Message.Body, &result.Message.HTMLBody, &result.Status, &result.Attempts, &next, &created, &updated, &result.LastError)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		result.TenantID = tenantID
		result.NextAttemptAt = scanTime(next)
		result.CreatedAt = scanTime(created)
		result.UpdatedAt = scanTime(updated)
		newToken, err := newSpoolToken()
		if err != nil {
			return err
		}
		token = newToken
		update := `UPDATE mail_spool SET status='sending',attempts=attempts+1,lease_owner=` + s.p(1) + `,lease_token=` + s.p(2) + `,lease_until=` + s.p(3) + `,updated_at=` + s.p(4) + ` WHERE id=` + s.p(5) + ` AND tenant_id=` + s.p(6) + ` AND ((status='pending' AND next_attempt_at <= ` + s.p(7) + `) OR (status='sending' AND lease_until <= ` + s.p(8) + `))`
		changed, err := tx.ExecContext(ctx, update, worker, token, spoolTime(s.Dialect, now.Add(lease)), spoolTime(s.Dialect, now), result.ID, tenantID, spoolTime(s.Dialect, now), spoolTime(s.Dialect, now))
		if err != nil {
			return err
		}
		count, err := changed.RowsAffected()
		if err != nil || count != 1 {
			return err
		}
		result.Attempts++
		result.Status = mailer.SpoolSending
		result.UpdatedAt = now
		claimed = true
		return nil
	})
	return result, token, claimed, err
}

func (s Spool) Delivered(ctx context.Context, id, worker, token string, now time.Time) error {
	return s.finish(ctx, id, worker, token, `status='delivered',delivered_at=`+s.p(1)+`,updated_at=`+s.p(2)+`,lease_owner=NULL,lease_token=NULL,lease_until=NULL`, spoolTime(s.Dialect, now), spoolTime(s.Dialect, now))
}

func (s Spool) Retry(ctx context.Context, id, worker, token string, retryAt time.Time, reason string) error {
	return s.finish(ctx, id, worker, token, `status='pending',next_attempt_at=`+s.p(1)+`,last_error=`+s.p(2)+`,updated_at=`+s.p(3)+`,lease_owner=NULL,lease_token=NULL,lease_until=NULL`, spoolTime(s.Dialect, retryAt), boundedSpoolError(reason), spoolTime(s.Dialect, time.Now().UTC()))
}

func (s Spool) FailPermanent(ctx context.Context, id, worker, token, reason string) error {
	return s.finish(ctx, id, worker, token, `status='failed_permanent',last_error=`+s.p(1)+`,updated_at=`+s.p(2)+`,lease_owner=NULL,lease_token=NULL,lease_until=NULL`, boundedSpoolError(reason), spoolTime(s.Dialect, time.Now().UTC()))
}

func (s Spool) RetryNow(ctx context.Context, id string) error {
	return s.withTx(ctx, func(tx database.Executor) error {
		tenantID, err := s.tenant(ctx)
		if err != nil {
			return err
		}
		now := spoolTime(s.Dialect, time.Now().UTC())
		query := `UPDATE mail_spool SET status='pending',attempts=0,next_attempt_at=` + s.p(1) + `,lease_owner=NULL,lease_token=NULL,lease_until=NULL,updated_at=` + s.p(2) + ` WHERE id=` + s.p(3) + ` AND tenant_id=` + s.p(4) + ` AND status='failed_permanent'`
		result, err := tx.ExecContext(ctx, query, now, now, id, tenantID)
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
	})
}

func (s Spool) List(ctx context.Context, status string, limit int) ([]mailer.SpoolEntry, error) {
	if status != "" && status != mailer.SpoolPending && status != mailer.SpoolSending && status != mailer.SpoolDelivered && status != mailer.SpoolFailedPermanent {
		return nil, errors.New("invalid mail spool status")
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}
	var result []mailer.SpoolEntry
	err := s.withQuery(ctx, func(db database.QueryExecutor) error {
		tenantID, err := s.tenant(ctx)
		if err != nil {
			return err
		}
		query := `SELECT id,tenant_id,event_key,recipient,subject,body,html_body,status,attempts,next_attempt_at,created_at,updated_at,delivered_at,COALESCE(last_error,'') FROM mail_spool WHERE tenant_id=` + s.p(1)
		args := []any{tenantID}
		if status != "" {
			query += ` AND status=` + s.p(2)
			args = append(args, status)
		}
		query += ` ORDER BY created_at DESC,id LIMIT ` + s.p(len(args)+1)
		args = append(args, limit)
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		result = make([]mailer.SpoolEntry, 0, limit)
		for rows.Next() {
			var item mailer.SpoolEntry
			var next, created, updated, delivered any
			if err := rows.Scan(&item.ID, &item.TenantID, &item.EventKey, &item.Message.To, &item.Message.Subject, &item.Message.Body, &item.Message.HTMLBody, &item.Status, &item.Attempts, &next, &created, &updated, &delivered, &item.LastError); err != nil {
				return err
			}
			item.NextAttemptAt = scanTime(next)
			item.CreatedAt = scanTime(created)
			item.UpdatedAt = scanTime(updated)
			item.DeliveredAt = scanTime(delivered)
			result = append(result, item)
		}
		return rows.Err()
	})
	return result, err
}

func (s Spool) PruneDelivered(ctx context.Context, before time.Time, limit int) (int, error) {
	if before.IsZero() {
		return 0, errors.New("mail spool cutoff is required")
	}
	if limit < 1 {
		return 0, nil
	}
	if limit > 100 {
		limit = 100
	}
	removed := 0
	err := s.withTx(ctx, func(tx database.Executor) error {
		tenantID, err := s.tenant(ctx)
		if err != nil {
			return err
		}
		rows, err := tx.(interface {
			QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		}).QueryContext(ctx, `SELECT id FROM mail_spool WHERE tenant_id=`+s.p(1)+` AND status='delivered' AND delivered_at < `+s.p(2)+` ORDER BY delivered_at,id LIMIT `+s.p(3), tenantID, spoolTime(s.Dialect, before), limit)
		if err != nil {
			return err
		}
		ids := make([]string, 0, limit)
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				_ = rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, id := range ids {
			result, err := tx.ExecContext(ctx, `DELETE FROM mail_spool WHERE id=`+s.p(1)+` AND tenant_id=`+s.p(2)+` AND status='delivered'`, id, tenantID)
			if err != nil {
				return err
			}
			count, err := result.RowsAffected()
			if err != nil {
				return err
			}
			removed += int(count)
		}
		return nil
	})
	return removed, err
}

func (s Spool) finish(ctx context.Context, id, worker, token, set string, values ...any) error {
	return s.withTx(ctx, func(tx database.Executor) error {
		tenantID, err := s.tenant(ctx)
		if err != nil {
			return err
		}
		args := append(append([]any(nil), values...), id, tenantID, worker, token)
		query := `UPDATE mail_spool SET ` + set + ` WHERE id=` + s.p(len(values)+1) + ` AND tenant_id=` + s.p(len(values)+2) + ` AND status='sending' AND lease_owner=` + s.p(len(values)+3) + ` AND lease_token=` + s.p(len(values)+4)
		result, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return err
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if changed != 1 {
			return mailer.ErrSpoolLeaseInvalid
		}
		return nil
	})
}

func (s Spool) withTx(ctx context.Context, fn func(database.Executor) error) error {
	transactor := s.Tx
	if transactor == (database.Transactor{}) {
		if db, ok := s.DB.(*sql.DB); ok {
			transactor = database.NewTenantTransactor(db, s.Dialect, s.RequireTenantScope)
		} else {
			return errors.New("mail spool transaction is required")
		}
	}
	return transactor.Within(ctx, fn)
}

func (s Spool) withQuery(ctx context.Context, fn func(database.QueryExecutor) error) error {
	transactor := s.Tx
	if transactor == (database.Transactor{}) {
		if db, ok := s.DB.(*sql.DB); ok {
			transactor = database.NewTenantTransactor(db, s.Dialect, s.RequireTenantScope)
		} else {
			return errors.New("mail spool query transaction is required")
		}
	}
	return transactor.WithinQuery(ctx, fn)
}

func spoolID(tenantID, eventKey string) string {
	hash := sha256.Sum256([]byte(tenantID + "\x00" + eventKey))
	return "mail-" + hex.EncodeToString(hash[:16])
}

func newSpoolToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func spoolTime(dialect database.Dialect, value time.Time) any {
	value = value.UTC()
	if dialect == database.SQLite {
		return value.Format(time.RFC3339Nano)
	}
	return value
}

func scanTime(value any) time.Time {
	switch value := value.(type) {
	case time.Time:
		return value.UTC()
	case string:
		parsed, _ := time.Parse(time.RFC3339Nano, value)
		return parsed.UTC()
	case []byte:
		parsed, _ := time.Parse(time.RFC3339Nano, string(value))
		return parsed.UTC()
	default:
		return time.Time{}
	}
}

func boundedSpoolError(reason string) string {
	return mailer.BoundSpoolError(reason)
}
