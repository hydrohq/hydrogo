package sqlstore

import (
	"context"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type Store struct {
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s Store) Record(ctx context.Context, tx database.Executor, id, action, targetID string) error {
	if _, scopeErr := tenant.FromContext(ctx); scopeErr == nil {
		return s.RecordScoped(ctx, tx, id, action, targetID)
	} else if s.RequireTenantScope {
		return scopeErr
	}
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	var requestID any
	if value := security.RequestIDFromContext(ctx); value != "" {
		requestID = value
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO audit_events (id, action, target_id, request_id) VALUES ("+s.placeholder(1)+", "+s.placeholder(2)+", "+s.placeholder(3)+", "+s.placeholder(4)+")", id, action, targetID, requestID)
	return err
}

// RecordScoped writes an audit event with an explicit tenant discriminator.
func (s Store) RecordScoped(ctx context.Context, tx database.Executor, id, action, targetID string) error {
	scope, err := tenant.Require(ctx)
	if err != nil {
		return err
	}
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	var requestID any
	if value := security.RequestIDFromContext(ctx); value != "" {
		requestID = value
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO audit_events (id, tenant_id, action, target_id, request_id) VALUES ("+s.placeholder(1)+", "+s.placeholder(2)+", "+s.placeholder(3)+", "+s.placeholder(4)+", "+s.placeholder(5)+")", id, scope.TenantID, action, targetID, requestID)
	return err
}

func (s Store) placeholder(position int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}
