package tenant

import (
	"context"
	"database/sql"
)

// ExecContext is the narrow database boundary needed to bind a tenant to a
// PostgreSQL transaction without exposing SQL details to domain services.
type ExecContext interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// BindPostgresTransaction sets the transaction-local RLS variable. Callers
// must invoke it on the transaction before tenant-scoped queries; PostgreSQL
// clears the setting automatically at transaction end.
func BindPostgresTransaction(ctx context.Context, exec ExecContext, scope Scope) error {
	if exec == nil || !validComponent(scope.TenantID) || !validComponent(scope.Slug) {
		return ErrMissingScope
	}
	if ctx == nil {
		ctx = context.Background()
	}
	_, err := exec.ExecContext(ctx, "SELECT set_config('app.tenant_rls_enforced', 'true', true), set_config('app.current_tenant_id', $1, true)", scope.TenantID)
	return err
}

// BindPostgresSingleTenantTransaction explicitly disables the optional RLS
// tenant predicate for a single-tenant deployment. The setting is
// transaction-local, so it cannot leak to another pooled connection or
// request. Shared deployments must use BindPostgresTransaction instead.
func BindPostgresSingleTenantTransaction(ctx context.Context, exec ExecContext) error {
	if exec == nil {
		return ErrMissingScope
	}
	if ctx == nil {
		ctx = context.Background()
	}
	_, err := exec.ExecContext(ctx, "SELECT set_config('app.tenant_rls_enforced', 'false', true)")
	return err
}
