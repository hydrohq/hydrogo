package sqlstore

import (
	"context"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (s Store) ReplaceNavigation(ctx context.Context, tx database.Executor, locale string, items []site.NavigationItem) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return err
	}
	deleteQuery := `DELETE FROM navigation_items WHERE locale = $1` + filter
	if s.Dialect != database.Postgres {
		deleteQuery = `DELETE FROM navigation_items WHERE locale = ?` + filter
	}
	deleteArgs := []any{locale}
	if filter != "" {
		deleteArgs = append(deleteArgs, scopeID)
	}
	if _, err := tx.ExecContext(ctx, deleteQuery, deleteArgs...); err != nil {
		return err
	}
	for _, item := range items {
		if item.Target == "" {
			item.Target = "_self"
		}
		query := `INSERT INTO navigation_items (tenant_id,id,locale,label,href,parent_id,target,position,is_visible,updated_at) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `,` + s.p(9) + `,` + s.p(10) + `)`
		updatedAt := item.UpdatedAt.UTC()
		if updatedAt.IsZero() {
			updatedAt = time.Now().UTC()
		}
		args := []any{scopeID, item.ID, locale, item.Label, item.Href, item.ParentID, item.Target, item.Position, item.Visible, updatedAt}
		if s.Dialect != database.Postgres {
			query = `INSERT INTO navigation_items (tenant_id,id,locale,label,href,parent_id,target,position,is_visible,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`
			args[8] = boolInt(item.Visible)
		}
		if s.Dialect == database.SQLite {
			args[9] = updatedAt.Format(time.RFC3339Nano)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) ListNavigation(ctx context.Context, db database.Executor, locale string) ([]site.NavigationItem, error) {
	return s.listNavigation(ctx, db, locale, true)
}

func (s Store) ListAdminNavigation(ctx context.Context, db database.Executor, locale string) ([]site.NavigationItem, error) {
	return s.listNavigation(ctx, db, locale, false)
}

func (s Store) listNavigation(ctx context.Context, db database.Executor, locale string, visibleOnly bool) ([]site.NavigationItem, error) {
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return nil, err
	}
	query := `SELECT id,locale,label,href,parent_id,target,position,is_visible,updated_at FROM navigation_items WHERE locale = $1`
	query += filter + ` ORDER BY position,id LIMIT 500`
	if s.Dialect != database.Postgres {
		query = `SELECT id,locale,label,href,parent_id,target,position,is_visible,updated_at FROM navigation_items WHERE locale = ?`
		query += filter + ` ORDER BY position,id LIMIT 500`
	}
	args := []any{locale}
	if filter != "" {
		args = append(args, scopeID)
	}
	rows, err := queryContext(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]site.NavigationItem, 0)
	for rows.Next() {
		var item site.NavigationItem
		var visible any
		var updated any
		if err := rows.Scan(&item.ID, &item.Locale, &item.Label, &item.Href, &item.ParentID, &item.Target, &item.Position, &visible, &updated); err != nil {
			return nil, err
		}
		item.Visible = parseBool(visible)
		item.UpdatedAt = parseTime(updated)
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if !visibleOnly {
		return result, nil
	}
	byID := make(map[string]site.NavigationItem, len(result))
	for _, item := range result {
		byID[item.ID] = item
	}
	public := make([]site.NavigationItem, 0, len(result))
	for _, item := range result {
		if !item.Visible {
			continue
		}
		parentID := item.ParentID
		seen := map[string]struct{}{item.ID: {}}
		visibleTree := true
		for parentID != "" {
			parent, exists := byID[parentID]
			if !exists || !parent.Visible {
				visibleTree = false
				break
			}
			if _, cycle := seen[parentID]; cycle {
				visibleTree = false
				break
			}
			seen[parentID] = struct{}{}
			parentID = parent.ParentID
		}
		if visibleTree {
			public = append(public, item)
		}
	}
	return public, nil
}
