package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

// ListPublishedRoutes returns only the public projection used by discovery surfaces.
func (s Store) ListPublishedRoutes(ctx context.Context, db *sql.DB) ([]pages.Page, error) {
	if db == nil {
		return nil, database.ErrDatabaseRequired
	}
	query := `SELECT id, published_locale, published_slug, published_title, status, revision, publication_generation, updated_at FROM pages WHERE published_revision IS NOT NULL ORDER BY published_locale, published_slug, id`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]pages.Page, 0)
	for rows.Next() {
		var page pages.Page
		var status string
		var updated any
		if err := rows.Scan(&page.ID, &page.Locale, &page.Slug, &page.Title, &status, &page.Revision, &page.PublicationGeneration, &updated); err != nil {
			return nil, err
		}
		page.Status = pages.Status(status)
		page.UpdatedAt = parseStoredTime(updated)
		result = append(result, page)
	}
	return result, rows.Err()
}

// ListPublished returns bounded public page summaries for a locale. Draft
// fields and unpublished routes are intentionally absent from this query.
func (s Store) ListPublished(ctx context.Context, db database.Executor, locale string, limit int) ([]pages.Page, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("published page list requires a query-capable executor")
	}
	query := `SELECT id, published_locale, published_slug, published_title, status, revision, publication_generation, updated_at FROM pages WHERE published_locale = ` + s.placeholder(1) + ` AND published_revision IS NOT NULL ORDER BY updated_at DESC, id ASC LIMIT ` + s.placeholder(2)
	rows, err := queryer.QueryContext(ctx, query, locale, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]pages.Page, 0)
	for rows.Next() {
		var page pages.Page
		var status string
		var updated any
		if err := rows.Scan(&page.ID, &page.Locale, &page.Slug, &page.Title, &status, &page.Revision, &page.PublicationGeneration, &updated); err != nil {
			return nil, err
		}
		page.Status = pages.Status(status)
		page.UpdatedAt = parseStoredTime(updated)
		result = append(result, page)
	}
	return result, rows.Err()
}

func (s Store) GetContent(ctx context.Context, db database.Executor, pageID string) (pages.Page, error) {
	if err := database.RequireExecutor(db); err != nil {
		return pages.Page{}, err
	}
	var page pages.Page
	var status string
	var published any
	var createdAt, updatedAt, blocks any
	query := `SELECT id, locale, slug, title, body, blocks, status, version, revision, published_revision, publication_generation, created_at, updated_at FROM pages WHERE id = ` + s.placeholder(1)
	err := db.QueryRowContext(ctx, query, pageID).Scan(&page.ID, &page.Locale, &page.Slug, &page.Title, &page.Body, &blocks, &status, &page.Version, &page.Revision, &published, &page.PublicationGeneration, &createdAt, &updatedAt)
	if err != nil {
		return pages.Page{}, err
	}
	page.Status = pages.Status(status)
	page.Blocks, err = pages.DecodeBlocksForStore(blocks)
	if err != nil {
		return pages.Page{}, err
	}
	if published != nil {
		var value int
		switch v := published.(type) {
		case int64:
			value = int(v)
		case int:
			value = v
		}
		page.PublishedRevision = &value
	}
	page.CreatedAt = parseStoredTime(createdAt)
	page.UpdatedAt = parseStoredTime(updatedAt)
	return page, nil
}

func (s Store) GetPublishedByRoute(ctx context.Context, db database.Executor, locale, slug string) (pages.Page, error) {
	if err := database.RequireExecutor(db); err != nil {
		return pages.Page{}, err
	}
	var page pages.Page
	var status string
	var published any
	var createdAt, updatedAt, blocks any
	query := `SELECT id, published_locale, published_slug, published_title, published_body, published_blocks, status, version, revision, published_revision, publication_generation, created_at, updated_at FROM pages WHERE published_locale = ` + s.placeholder(1) + ` AND published_slug = ` + s.placeholder(2) + ` AND published_revision IS NOT NULL`
	err := db.QueryRowContext(ctx, query, locale, slug).Scan(&page.ID, &page.Locale, &page.Slug, &page.Title, &page.Body, &blocks, &status, &page.Version, &page.Revision, &published, &page.PublicationGeneration, &createdAt, &updatedAt)
	if err != nil {
		return pages.Page{}, err
	}
	page.Status = pages.Status(status)
	page.Blocks, err = pages.DecodeBlocksForStore(blocks)
	if err != nil {
		return pages.Page{}, err
	}
	if published != nil {
		var value int
		switch v := published.(type) {
		case int64:
			value = int(v)
		case int:
			value = v
		}
		page.PublishedRevision = &value
	}
	page.CreatedAt = parseStoredTime(createdAt)
	page.UpdatedAt = parseStoredTime(updatedAt)
	return page, nil
}

func parseStoredTime(value any) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v.UTC()
	case string:
		parsed, _ := time.Parse(time.RFC3339Nano, v)
		return parsed.UTC()
	case []byte:
		parsed, _ := time.Parse(time.RFC3339Nano, string(v))
		return parsed.UTC()
	default:
		return time.Time{}
	}
}

func (s Store) CreateDraftContent(ctx context.Context, tx database.Executor, page pages.Page) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	blocks, err := pages.EncodeBlocksForStore(page.Blocks)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO pages
		(id, title, version, locale, slug, body, blocks, status, revision, publication_generation, created_at, updated_at)
		VALUES (`+s.placeholder(1)+`, `+s.placeholder(2)+`, 1, `+s.placeholder(3)+`, `+s.placeholder(4)+`, `+s.placeholder(5)+`, `+s.placeholder(6)+`, 'draft', 1, 0, `+s.placeholder(7)+`, `+s.placeholder(8)+`)`,
		page.ID, page.Title, page.Locale, page.Slug, page.Body, blocks, now, now)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO page_revisions
		(id, page_id, revision, locale, slug, title, body, blocks, created_at)
		VALUES (`+s.placeholder(1)+`, `+s.placeholder(2)+`, 1, `+s.placeholder(3)+`, `+s.placeholder(4)+`, `+s.placeholder(5)+`, `+s.placeholder(6)+`, `+s.placeholder(7)+`, `+s.placeholder(8)+`)`,
		page.ID+"-revision-1", page.ID, page.Locale, page.Slug, page.Title, page.Body, blocks, now)
	return err
}

func (s Store) UpdateDraftContent(ctx context.Context, tx database.Executor, page pages.Page, expectedVersion int) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	blocks, err := pages.EncodeBlocksForStore(page.Blocks)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE pages SET title = `+s.placeholder(1)+`, slug = `+s.placeholder(2)+`, body = `+s.placeholder(3)+`, blocks = `+s.placeholder(4)+`, locale = `+s.placeholder(5)+`, revision = revision + 1, version = version + 1, status = 'draft', updated_at = `+s.placeholder(6)+` WHERE id = `+s.placeholder(7)+` AND version = `+s.placeholder(8), page.Title, page.Slug, page.Body, blocks, page.Locale, now, page.ID, expectedVersion)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return pages.ErrConflict
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO page_revisions
		(id, page_id, revision, locale, slug, title, body, blocks, created_at)
		VALUES (`+s.placeholder(1)+`, `+s.placeholder(2)+`, `+s.placeholder(3)+`, `+s.placeholder(4)+`, `+s.placeholder(5)+`, `+s.placeholder(6)+`, `+s.placeholder(7)+`, `+s.placeholder(8)+`, `+s.placeholder(9)+`)`,
		fmt.Sprintf("%s-revision-%d", page.ID, expectedVersion+1), page.ID, expectedVersion+1, page.Locale, page.Slug, page.Title, page.Body, blocks, now)
	return err
}

func (s Store) Publish(ctx context.Context, tx database.Executor, pageID string, expectedVersion int) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, `UPDATE pages SET status = 'published', published_revision = revision, published_locale = locale, published_slug = slug, published_title = title, published_body = body, published_blocks = blocks, publication_generation = publication_generation + 1, version = version + 1, updated_at = `+s.placeholder(1)+` WHERE id = `+s.placeholder(2)+` AND version = `+s.placeholder(3)+` AND status IN ('draft', 'published')`, now, pageID, expectedVersion)
	if err != nil {
		if message := strings.ToLower(err.Error()); strings.Contains(message, "unique") || strings.Contains(message, "duplicate") {
			return pages.ErrRouteConflict
		}
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return pages.ErrConflict
	}
	return nil
}
