package sqlstore

import (
	"context"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type Store struct {
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s Store) tenantFilter(ctx context.Context, position int) (string, any, error) {
	if !s.RequireTenantScope {
		return "", nil, nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", nil, err
	}
	return " AND tenant_id = " + s.placeholder(position), scope.TenantID, nil
}

func (s Store) InsertDraft(ctx context.Context, tx database.Executor, id, title string) error {
	query := "INSERT INTO pages (id, title) VALUES (" + s.placeholder(1) + ", " + s.placeholder(2) + ")"
	args := []any{id, title}
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return scopeErr
		}
		query = "INSERT INTO pages (id, title, tenant_id) VALUES (" + s.placeholder(1) + ", " + s.placeholder(2) + ", " + s.placeholder(3) + ")"
		args = append(args, scope.TenantID)
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (s Store) UpdateTitle(ctx context.Context, tx database.Executor, id string, expectedVersion int, title string) (bool, error) {
	filter, scopeID, err := s.tenantFilter(ctx, 4)
	if err != nil {
		return false, err
	}
	args := []any{title, id, expectedVersion}
	if filter != "" {
		args = append(args, scopeID)
	}
	result, err := tx.ExecContext(ctx,
		"UPDATE pages SET title = "+s.placeholder(1)+", version = version + 1 WHERE id = "+s.placeholder(2)+" AND version = "+s.placeholder(3)+filter,
		args...,
	)
	if err != nil {
		return false, err
	}
	changed, err := result.RowsAffected()
	return changed == 1, err
}

func (s Store) placeholder(position int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}
