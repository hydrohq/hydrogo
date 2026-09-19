package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/outbox"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
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
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO outbox_events (id, topic, aggregate_id, payload_version, available_at, attempts) VALUES ("+s.placeholder(1)+", "+s.placeholder(2)+", "+s.placeholder(3)+", "+s.placeholder(4)+", "+s.placeholder(5)+", 0)", id, topic, aggregateID, payloadVersion, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// EnqueueScoped requires an explicit tenant scope. It is the shared-database
// outbox boundary; legacy Enqueue remains for single-customer deployments.
func (s Store) EnqueueScoped(ctx context.Context, tx database.Executor, id, topic, aggregateID string, payloadVersion int) error {
	scope, err := tenant.Require(ctx)
	if err != nil {
		return err
	}
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO outbox_events (id, tenant_id, topic, aggregate_id, payload_version, available_at, attempts) VALUES ("+s.placeholder(1)+", "+s.placeholder(2)+", "+s.placeholder(3)+", "+s.placeholder(4)+", "+s.placeholder(5)+", "+s.placeholder(6)+", 0)", id, scope.TenantID, topic, aggregateID, payloadVersion, time.Now().UTC().Format(time.RFC3339Nano))
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
	var available, leaseUntil, owner, token, lastError string
	q := `SELECT id, topic, aggregate_id, payload_version, attempts, available_at, COALESCE(lease_until, ''), COALESCE(lease_owner, ''), COALESCE(lease_token, ''), COALESCE(last_error, '') FROM outbox_events WHERE available_at <= ` + s.placeholder(1) + ` AND (lease_until IS NULL OR lease_until <= ` + s.placeholder(2) + `) ORDER BY available_at, id LIMIT 1`
	err := db.QueryRowContext(ctx, q, now.UTC().Format(time.RFC3339Nano), now.UTC().Format(time.RFC3339Nano)).Scan(&event.ID, &event.Topic, &event.AggregateID, &event.PayloadVersion, &event.Attempts, &available, &leaseUntil, &owner, &token, &lastError)
	if err != nil {
		if err == sql.ErrNoRows {
			return outbox.Event{}, "", false, nil
		}
		return outbox.Event{}, "", false, err
	}
	event.AvailableAt, err = time.Parse(time.RFC3339Nano, available)
	if err != nil {
		return outbox.Event{}, "", false, err
	}
	leaseToken := newToken(worker, event.ID, now)
	result, err := db.ExecContext(ctx, `UPDATE outbox_events SET lease_owner = `+s.placeholder(1)+`, lease_token = `+s.placeholder(2)+`, lease_until = `+s.placeholder(3)+`, attempts = attempts + 1 WHERE id = `+s.placeholder(4)+` AND (lease_until IS NULL OR lease_until <= `+s.placeholder(5)+`)`, worker, leaseToken, now.Add(lease).UTC().Format(time.RFC3339Nano), event.ID, now.UTC().Format(time.RFC3339Nano))
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

// ClaimScoped claims only events belonging to the context tenant. The tenant
// predicate is part of both the read and lease update to prevent cross-tenant
// worker claims in a shared database.
func (s Store) ClaimScoped(ctx context.Context, db database.Executor, worker string, now time.Time, lease time.Duration) (outbox.Event, string, bool, error) {
	scope, err := tenant.Require(ctx)
	if err != nil {
		return outbox.Event{}, "", false, err
	}
	if err := database.RequireExecutor(db); err != nil || worker == "" || lease <= 0 {
		return outbox.Event{}, "", false, err
	}
	var event outbox.Event
	var available, leaseUntil, owner, token, lastError string
	q := `SELECT id, tenant_id, topic, aggregate_id, payload_version, attempts, available_at, COALESCE(lease_until, ''), COALESCE(lease_owner, ''), COALESCE(lease_token, ''), COALESCE(last_error, '') FROM outbox_events WHERE tenant_id = ` + s.placeholder(1) + ` AND available_at <= ` + s.placeholder(2) + ` AND (lease_until IS NULL OR lease_until <= ` + s.placeholder(3) + `) ORDER BY available_at, id LIMIT 1`
	err = db.QueryRowContext(ctx, q, scope.TenantID, now.UTC().Format(time.RFC3339Nano), now.UTC().Format(time.RFC3339Nano)).Scan(&event.ID, &event.TenantID, &event.Topic, &event.AggregateID, &event.PayloadVersion, &event.Attempts, &available, &leaseUntil, &owner, &token, &lastError)
	if err != nil {
		if err == sql.ErrNoRows {
			return outbox.Event{}, "", false, nil
		}
		return outbox.Event{}, "", false, err
	}
	event.AvailableAt, err = time.Parse(time.RFC3339Nano, available)
	if err != nil {
		return outbox.Event{}, "", false, err
	}
	leaseToken := newToken(worker, event.ID, now)
	result, err := db.ExecContext(ctx, `UPDATE outbox_events SET lease_owner = `+s.placeholder(1)+`, lease_token = `+s.placeholder(2)+`, lease_until = `+s.placeholder(3)+`, attempts = attempts + 1 WHERE id = `+s.placeholder(4)+` AND tenant_id = `+s.placeholder(5)+` AND (lease_until IS NULL OR lease_until <= `+s.placeholder(6)+`)`, worker, leaseToken, now.Add(lease).UTC().Format(time.RFC3339Nano), event.ID, scope.TenantID, now.UTC().Format(time.RFC3339Nano))
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

func (s Store) AckScoped(ctx context.Context, db database.Executor, id, worker, token string) error {
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

func (s Store) RetryScoped(ctx context.Context, db database.Executor, id, worker, token string, availableAt time.Time, reason string) error {
	scope, err := tenant.Require(ctx)
	if err != nil {
		return err
	}
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, `UPDATE outbox_events SET lease_owner = NULL, lease_token = NULL, lease_until = NULL, available_at = `+s.placeholder(1)+`, last_error = `+s.placeholder(2)+` WHERE id = `+s.placeholder(3)+` AND tenant_id = `+s.placeholder(4)+` AND lease_owner = `+s.placeholder(5)+` AND lease_token = `+s.placeholder(6), availableAt.UTC().Format(time.RFC3339Nano), reason, id, scope.TenantID, worker, token)
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
	result, err := db.ExecContext(ctx, `UPDATE outbox_events SET lease_owner = NULL, lease_token = NULL, lease_until = NULL, available_at = `+s.placeholder(1)+`, last_error = `+s.placeholder(2)+` WHERE id = `+s.placeholder(3)+` AND lease_owner = `+s.placeholder(4)+` AND lease_token = `+s.placeholder(5), availableAt.UTC().Format(time.RFC3339Nano), reason, id, worker, token)
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
