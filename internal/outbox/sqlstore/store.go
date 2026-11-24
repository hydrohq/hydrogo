package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/internal/eventcatalog"
	"github.com/hydrohq/hydrogo/internal/outbox"
	"github.com/hydrohq/hydrogo/internal/tenant"
)

type Store struct {
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s Store) Enqueue(ctx context.Context, tx database.Executor, id, topic, aggregateID string, payloadVersion int) error {
	if _, scopeErr := tenant.FromContext(ctx); scopeErr == nil {
		return s.EnqueueScoped(ctx, tx, id, topic, aggregateID, payloadVersion)
	} else if s.RequireTenantScope {
		return scopeErr
	}
	if err := eventcatalog.Validate(topic, payloadVersion); err != nil {
		return err
	}
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO outbox_events (id, topic, aggregate_id, payload_version, available_at, attempts) VALUES ("+s.placeholder(1)+", "+s.placeholder(2)+", "+s.placeholder(3)+", "+s.placeholder(4)+", "+s.placeholder(5)+", 0)", id, topic, aggregateID, payloadVersion, s.dbNow())
	return err
}

func (s Store) enqueueScoped(ctx context.Context, tx database.Executor, id, topic, aggregateID string, payloadVersion int) error {
	scope, err := tenant.Require(ctx)
	if err != nil {
		return err
	}
	if err := eventcatalog.Validate(topic, payloadVersion); err != nil {
		return err
	}
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO outbox_events (id, tenant_id, topic, aggregate_id, payload_version, available_at, attempts) VALUES ("+s.placeholder(1)+", "+s.placeholder(2)+", "+s.placeholder(3)+", "+s.placeholder(4)+", "+s.placeholder(5)+", "+s.placeholder(6)+", 0)", id, scope.TenantID, topic, aggregateID, payloadVersion, s.dbNow())
	return err
}

func (s Store) Claim(ctx context.Context, db database.Executor, worker string, now time.Time, lease time.Duration) (outbox.Event, string, bool, error) {
	if _, scopeErr := tenant.FromContext(ctx); scopeErr == nil {
		return s.ClaimScoped(ctx, db, worker, now, lease)
	} else if s.RequireTenantScope {
		return outbox.Event{}, "", false, scopeErr
	}
	if err := database.RequireExecutor(db); err != nil {
		return outbox.Event{}, "", false, err
	}
	if worker == "" || lease <= 0 {
		return outbox.Event{}, "", false, nil
	}
	var event outbox.Event
	var available any
	var leaseUntil, owner, token, lastError string
	q := `SELECT id, topic, aggregate_id, payload_version, attempts, available_at, COALESCE(lease_until, ''), COALESCE(lease_owner, ''), COALESCE(lease_token, ''), COALESCE(last_error, '') FROM outbox_events WHERE available_at <= ` + s.placeholder(1) + ` AND (lease_until IS NULL OR lease_until <= ` + s.placeholder(2) + `) ORDER BY available_at, id LIMIT 1`
	err := db.QueryRowContext(ctx, q, s.dbTimeValue(now), s.dbTimeValue(now)).Scan(&event.ID, &event.Topic, &event.AggregateID, &event.PayloadVersion, &event.Attempts, &available, &leaseUntil, &owner, &token, &lastError)
	if err != nil {
		if err == sql.ErrNoRows {
			return outbox.Event{}, "", false, nil
		}
		return outbox.Event{}, "", false, err
	}
	event.AvailableAt, err = parseOutboxTime(available)
	if err != nil {
		return outbox.Event{}, "", false, err
	}
	leaseToken := newToken(worker, event.ID, now)
	result, err := db.ExecContext(ctx, `UPDATE outbox_events SET lease_owner = `+s.placeholder(1)+`, lease_token = `+s.placeholder(2)+`, lease_until = `+s.placeholder(3)+`, attempts = attempts + 1 WHERE id = `+s.placeholder(4)+` AND (lease_until IS NULL OR lease_until <= `+s.placeholder(5)+`)`, worker, leaseToken, s.dbTimeValue(now.Add(lease)), event.ID, s.dbTimeValue(now))
	if err != nil {
		return outbox.Event{}, "", false, err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return outbox.Event{}, "", false, err
	}
	event.Attempts++
	event.LastError = lastError
	return event, leaseToken, true, nil
}

func (s Store) claimScoped(ctx context.Context, db database.Executor, worker string, now time.Time, lease time.Duration) (outbox.Event, string, bool, error) {
	scope, err := tenant.Require(ctx)
	if err != nil {
		return outbox.Event{}, "", false, err
	}
	if err := database.RequireExecutor(db); err != nil || worker == "" || lease <= 0 {
		return outbox.Event{}, "", false, err
	}
	var event outbox.Event
	var available any
	var leaseUntil, owner, token, lastError string
	q := `SELECT id, tenant_id, topic, aggregate_id, payload_version, attempts, available_at, COALESCE(lease_until, ''), COALESCE(lease_owner, ''), COALESCE(lease_token, ''), COALESCE(last_error, '') FROM outbox_events WHERE tenant_id = ` + s.placeholder(1) + ` AND available_at <= ` + s.placeholder(2) + ` AND (lease_until IS NULL OR lease_until <= ` + s.placeholder(3) + `) ORDER BY available_at, id LIMIT 1`
	err = db.QueryRowContext(ctx, q, scope.TenantID, s.dbTimeValue(now), s.dbTimeValue(now)).Scan(&event.ID, &event.TenantID, &event.Topic, &event.AggregateID, &event.PayloadVersion, &event.Attempts, &available, &leaseUntil, &owner, &token, &lastError)
	if err != nil {
		if err == sql.ErrNoRows {
			return outbox.Event{}, "", false, nil
		}
		return outbox.Event{}, "", false, err
	}
	event.AvailableAt, err = parseOutboxTime(available)
	if err != nil {
		return outbox.Event{}, "", false, err
	}
	leaseToken := newToken(worker, event.ID, now)
	result, err := db.ExecContext(ctx, `UPDATE outbox_events SET lease_owner = `+s.placeholder(1)+`, lease_token = `+s.placeholder(2)+`, lease_until = `+s.placeholder(3)+`, attempts = attempts + 1 WHERE id = `+s.placeholder(4)+` AND tenant_id = `+s.placeholder(5)+` AND (lease_until IS NULL OR lease_until <= `+s.placeholder(6)+`)`, worker, leaseToken, s.dbTimeValue(now.Add(lease)), event.ID, scope.TenantID, s.dbTimeValue(now))
	if err != nil {
		return outbox.Event{}, "", false, err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return outbox.Event{}, "", false, err
	}
	event.Attempts++
	event.LastError = lastError
	return event, leaseToken, true, nil
}

func (s Store) ackScoped(ctx context.Context, db database.Executor, id, worker, token string) error {
	scope, err := tenant.Require(ctx)
	if err != nil {
		return err
	}
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, `DELETE FROM outbox_events WHERE id = `+s.placeholder(1)+` AND tenant_id = `+s.placeholder(2)+` AND lease_owner = `+s.placeholder(3)+` AND lease_token = `+s.placeholder(4), id, scope.TenantID, worker, token)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return outbox.ErrLeaseInvalid
	}
	return nil
}

func (s Store) retryScoped(ctx context.Context, db database.Executor, id, worker, token string, availableAt time.Time, reason string) error {
	scope, err := tenant.Require(ctx)
	if err != nil {
		return err
	}
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, `UPDATE outbox_events SET lease_owner = NULL, lease_token = NULL, lease_until = NULL, available_at = `+s.placeholder(1)+`, last_error = `+s.placeholder(2)+` WHERE id = `+s.placeholder(3)+` AND tenant_id = `+s.placeholder(4)+` AND lease_owner = `+s.placeholder(5)+` AND lease_token = `+s.placeholder(6), s.dbTimeValue(availableAt), reason, id, scope.TenantID, worker, token)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return outbox.ErrLeaseInvalid
	}
	return nil
}

func (s Store) Ack(ctx context.Context, db database.Executor, id, worker, token string) error {
	if _, scopeErr := tenant.FromContext(ctx); scopeErr == nil {
		return s.AckScoped(ctx, db, id, worker, token)
	} else if s.RequireTenantScope {
		return scopeErr
	}
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, `DELETE FROM outbox_events WHERE id = `+s.placeholder(1)+` AND lease_owner = `+s.placeholder(2)+` AND lease_token = `+s.placeholder(3), id, worker, token)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return outbox.ErrLeaseInvalid
	}
	return nil
}

func (s Store) Retry(ctx context.Context, db database.Executor, id, worker, token string, availableAt time.Time, reason string) error {
	if _, scopeErr := tenant.FromContext(ctx); scopeErr == nil {
		return s.RetryScoped(ctx, db, id, worker, token, availableAt, reason)
	} else if s.RequireTenantScope {
		return scopeErr
	}
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, `UPDATE outbox_events SET lease_owner = NULL, lease_token = NULL, lease_until = NULL, available_at = `+s.placeholder(1)+`, last_error = `+s.placeholder(2)+` WHERE id = `+s.placeholder(3)+` AND lease_owner = `+s.placeholder(4)+` AND lease_token = `+s.placeholder(5), s.dbTimeValue(availableAt), reason, id, worker, token)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return outbox.ErrLeaseInvalid
	}
	return nil
}

// DeadLetter permanently removes a leased event from the delivery queue and
// records the bounded-delivery failure for operator inspection.
func (s Store) DeadLetter(ctx context.Context, db database.Executor, event outbox.Event, worker, token, reason string) error {
	if _, scopeErr := tenant.FromContext(ctx); scopeErr == nil {
		return s.DeadLetterScoped(ctx, db, event, worker, token, reason)
	} else if s.RequireTenantScope {
		return scopeErr
	}
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if err := insertDeadLetter(ctx, db, s, event, "", reason); err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, `DELETE FROM outbox_events WHERE id = `+s.placeholder(1)+` AND lease_owner = `+s.placeholder(2)+` AND lease_token = `+s.placeholder(3), event.ID, worker, token)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return outbox.ErrLeaseInvalid
	}
	return nil
}

func (s Store) deadLetterScoped(ctx context.Context, db database.Executor, event outbox.Event, worker, token, reason string) error {
	scope, err := tenant.Require(ctx)
	if err != nil {
		return err
	}
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if err := insertDeadLetter(ctx, db, s, event, scope.TenantID, reason); err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, `DELETE FROM outbox_events WHERE id = `+s.placeholder(1)+` AND tenant_id = `+s.placeholder(2)+` AND lease_owner = `+s.placeholder(3)+` AND lease_token = `+s.placeholder(4), event.ID, scope.TenantID, worker, token)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return outbox.ErrLeaseInvalid
	}
	return nil
}

// bound runs fn on db bound to the context tenant for PostgreSQL row-level
// security. Workers pass a raw *sql.DB; without the transaction-local
// app.current_tenant_id the fail-closed policy hides every named tenant's
// rows. An open transaction (not a database.DB) is used as is: its caller
// already bound it.
func (s Store) bound(ctx context.Context, db database.Executor, fn func(database.Executor) error) error {
	if _, err := tenant.Require(ctx); err != nil {
		return err
	}
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	handle, ok := db.(database.DB)
	if s.Dialect != database.Postgres || !ok {
		return fn(db)
	}
	return database.NewTenantTransactor(handle, s.Dialect, true).Within(ctx, fn)
}

// EnqueueScoped requires an explicit tenant scope. It is the shared-database
// outbox boundary; legacy Enqueue remains for single-customer deployments.
func (s Store) EnqueueScoped(ctx context.Context, db database.Executor, id, topic, aggregateID string, payloadVersion int) error {
	return s.bound(ctx, db, func(exec database.Executor) error {
		return s.enqueueScoped(ctx, exec, id, topic, aggregateID, payloadVersion)
	})
}

// ClaimScoped claims only events belonging to the context tenant. The tenant
// predicate is part of both the read and lease update to prevent cross-tenant
// worker claims in a shared database.
func (s Store) ClaimScoped(ctx context.Context, db database.Executor, worker string, now time.Time, lease time.Duration) (event outbox.Event, token string, claimed bool, err error) {
	err = s.bound(ctx, db, func(exec database.Executor) error {
		var claimErr error
		event, token, claimed, claimErr = s.claimScoped(ctx, exec, worker, now, lease)
		return claimErr
	})
	if err != nil {
		return outbox.Event{}, "", false, err
	}
	return event, token, claimed, nil
}

func (s Store) AckScoped(ctx context.Context, db database.Executor, id, worker, token string) error {
	return s.bound(ctx, db, func(exec database.Executor) error { return s.ackScoped(ctx, exec, id, worker, token) })
}

func (s Store) RetryScoped(ctx context.Context, db database.Executor, id, worker, token string, availableAt time.Time, reason string) error {
	return s.bound(ctx, db, func(exec database.Executor) error {
		return s.retryScoped(ctx, exec, id, worker, token, availableAt, reason)
	})
}

func (s Store) DeadLetterScoped(ctx context.Context, db database.Executor, event outbox.Event, worker, token, reason string) error {
	return s.bound(ctx, db, func(exec database.Executor) error {
		return s.deadLetterScoped(ctx, exec, event, worker, token, reason)
	})
}

func insertDeadLetter(ctx context.Context, db database.Executor, s Store, event outbox.Event, tenantID, reason string) error {
	_, err := db.ExecContext(ctx, `INSERT INTO outbox_dead_letters (id, tenant_id, topic, aggregate_id, payload_version, attempts, last_error, failed_at) VALUES (`+s.placeholder(1)+`, `+s.placeholder(2)+`, `+s.placeholder(3)+`, `+s.placeholder(4)+`, `+s.placeholder(5)+`, `+s.placeholder(6)+`, `+s.placeholder(7)+`, `+s.placeholder(8)+`)`, event.ID, tenantID, event.Topic, event.AggregateID, event.PayloadVersion, event.Attempts, reason, s.dbNow())
	return err
}

func newToken(worker, id string, now time.Time) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s-%s-%d", worker, id, now.UnixNano())
	}
	return fmt.Sprintf("%x", b)
}

func (s Store) placeholder(position int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}

func (s Store) dbTimeValue(t time.Time) any {
	t = t.UTC()
	switch s.Dialect {
	case database.SQLite, database.Postgres:
		// Stored as TEXT: RFC3339Nano trims trailing zeros, which breaks
		// lexicographical string comparison in SQLite and PostgreSQL when
		// comparing timestamps (e.g. 'Z' > '0'..'9').
		// The fixed width keeps text order equal to time order.
		return t.Format("2006-01-02T15:04:05.000000000Z")
	}
	return t
}

func (s Store) dbNow() any { return s.dbTimeValue(time.Now()) }

func parseOutboxTime(value any) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v.UTC(), nil
	case string:
		return time.Parse(time.RFC3339Nano, v)
	case []byte:
		return time.Parse(time.RFC3339Nano, string(v))
	default:
		return time.Time{}, fmt.Errorf("unsupported timestamp value %T", value)
	}
}
