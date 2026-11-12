package sqlstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/internal/tenant"
)

type transactionBeginner interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// bindTenant binds tenantID to a PostgreSQL transaction so row-level
// security sees the aggregate's tenant. An empty tenantID is the default
// single-tenant partition. Other dialects have no row-level security and are
// left untouched.
func (s Store) bindTenant(ctx context.Context, tx *sql.Tx, tenantID string) error {
	if s.Dialect != database.Postgres {
		return nil
	}
	if tenantID == "" {
		return tenant.BindPostgresSingleTenantTransaction(ctx, tx)
	}
	scope, err := tenant.NewScope(tenantID, tenantID)
	if err != nil {
		return err
	}
	return tenant.BindPostgresTransaction(ctx, tx, scope)
}

// withTenant runs fn against tenantID's rows. On PostgreSQL the work happens
// in a transaction bound to that tenant, because the analytics workers have no
// request scope and the aggregates carry their tenant explicitly.
func (s Store) withTenant(ctx context.Context, tenantID string, fn func(database.Executor) error) error {
	if s.Dialect != database.Postgres {
		return fn(s.Executor)
	}
	beginner, ok := s.Executor.(transactionBeginner)
	if !ok {
		return errors.New("analytics on PostgreSQL requires transaction support")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.bindTenant(ctx, tx, tenantID); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// retentionTenants lists the tenant partitions a retention purge must visit.
// Row-level security hides every tenant but the bound one, so a purge cannot
// simply delete across tenants on PostgreSQL. The default single-tenant
// partition and the request tenant, when present, are always included.
func (s Store) retentionTenants(ctx context.Context) ([]string, error) {
	tenants := []string{""}
	seen := map[string]bool{"": true}
	add := func(id string) {
		if !seen[id] {
			seen[id] = true
			tenants = append(tenants, id)
		}
	}
	if scope, err := tenant.FromContext(ctx); err == nil {
		add(scope.TenantID)
	}
	if s.TenantIDs != nil {
		ids, err := s.TenantIDs(ctx)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			add(id)
		}
	}
	return tenants, nil
}
