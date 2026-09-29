package sqlstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func (s Store) ListRevisions(ctx context.Context, db database.Executor, pageID string, limit int) ([]pages.RevisionSnapshot, error) {
	if pageID == "" || limit < 1 || limit > 50 {
		return nil, errors.New("invalid revision query")
	}
	queryer, ok := db.(database.QueryExecutor)
	if !ok {
		return nil, database.ErrDatabaseRequired
	}
	query := `SELECT pr.id,pr.page_id,pr.revision,pr.title,pr.body,pr.blocks,pr.created_at,pr.actor_id,COALESCE(u.email,'') FROM page_revisions pr LEFT JOIN users u ON u.id=pr.actor_id AND u.tenant_id=pr.tenant_id WHERE pr.page_id=` + s.placeholder(1)
	args := []any{pageID}
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return nil, scopeErr
		}
		query += ` AND pr.tenant_id=` + s.placeholder(2)
		args = append(args, scope.TenantID)
	}
	limitPosition := len(args) + 1
	query += ` ORDER BY pr.revision DESC LIMIT ` + s.placeholder(limitPosition)
	args = append(args, limit)
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]pages.RevisionSnapshot, 0)
	for rows.Next() {
		var item pages.RevisionSnapshot
		var rawBlocks, created any
		if err := rows.Scan(&item.ID, &item.PageID, &item.Revision, &item.Title, &item.Body, &rawBlocks, &created, &item.ActorID, &item.ActorEmail); err != nil {
			return nil, err
		}
		item.Blocks, err = pages.DecodeBlocksForStore(rawBlocks)
		if err != nil {
			return nil, fmt.Errorf("decode revision blocks: %w", err)
		}
		item.CreatedAt = parseStoredTime(created)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s Store) GetRevisionSlug(ctx context.Context, db database.Executor, pageID string, revision int) (string, string, error) {
	if err := database.RequireExecutor(db); err != nil {
		return "", "", err
	}
	if pageID == "" || revision < 1 {
		return "", "", errors.New("invalid published revision")
	}
	query := `SELECT locale,slug FROM page_revisions WHERE page_id=` + s.placeholder(1) + ` AND revision=` + s.placeholder(2)
	args := []any{pageID, revision}
	if s.RequireTenantScope {
		scope, err := tenant.Require(ctx)
		if err != nil {
			return "", "", err
		}
		query += ` AND tenant_id=` + s.placeholder(3)
		args = append(args, scope.TenantID)
	}
	var locale, slug string
	err := db.QueryRowContext(ctx, query, args...).Scan(&locale, &slug)
	return locale, slug, err
}
