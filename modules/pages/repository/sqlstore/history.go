package sqlstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/database"
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
	filter, tenantID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return nil, err
	}
	limitPosition := 2
	if filter != "" {
		limitPosition = 3
	}
	query := `SELECT id,page_id,revision,title,body,blocks,created_at FROM page_revisions WHERE page_id=` + s.placeholder(1) + filter + ` ORDER BY revision DESC LIMIT ` + s.placeholder(limitPosition)
	args := []any{pageID}
	if filter != "" {
		args = append(args, tenantID)
	}
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
		if err := rows.Scan(&item.ID, &item.PageID, &item.Revision, &item.Title, &item.Body, &rawBlocks, &created); err != nil {
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
