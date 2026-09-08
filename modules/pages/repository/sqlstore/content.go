package sqlstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func (s Store) GetContent(ctx context.Context, db database.Executor, pageID string) (pages.Page, error) {
	var page pages.Page
	var status string
	var published any
	var createdAt, updatedAt any
	query := `SELECT id, locale, slug, title, body, status, version, revision, published_revision, publication_generation, created_at, updated_at FROM pages WHERE id = ` + s.placeholder(1)
	err := db.QueryRowContext(ctx, query, pageID).Scan(&page.ID, &page.Locale, &page.Slug, &page.Title, &page.Body, &status, &page.Version, &page.Revision, &published, &page.PublicationGeneration, &createdAt, &updatedAt)
	if err != nil {
		return pages.Page{}, err
	}
	page.Status = pages.Status(status)
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
	var page pages.Page
	var status string
	var published any
	var createdAt, updatedAt any
	query := `SELECT id, published_locale, published_slug, published_title, published_body, status, version, revision, published_revision, publication_generation, created_at, updated_at FROM pages WHERE published_locale = ` + s.placeholder(1) + ` AND published_slug = ` + s.placeholder(2) + ` AND published_revision IS NOT NULL`
	err := db.QueryRowContext(ctx, query, locale, slug).Scan(&page.ID, &page.Locale, &page.Slug, &page.Title, &page.Body, &status, &page.Version, &page.Revision, &published, &page.PublicationGeneration, &createdAt, &updatedAt)
	if err != nil {
		return pages.Page{}, err
	}
	page.Status = pages.Status(status)
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
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := tx.ExecContext(ctx, `INSERT INTO pages
        (id, title, version, locale, slug, body, status, revision, publication_generation, created_at, updated_at)
        VALUES (`+s.placeholder(1)+`, `+s.placeholder(2)+`, 1, `+s.placeholder(3)+`, `+s.placeholder(4)+`, `+s.placeholder(5)+`, 'draft', 1, 0, `+s.placeholder(6)+`, `+s.placeholder(7)+`)`,
		page.ID, page.Title, page.Locale, page.Slug, page.Body, now, now)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO page_revisions
        (id, page_id, revision, locale, slug, title, body, created_at)
        VALUES (`+s.placeholder(1)+`, `+s.placeholder(2)+`, 1, `+s.placeholder(3)+`, `+s.placeholder(4)+`, `+s.placeholder(5)+`, `+s.placeholder(6)+`, `+s.placeholder(7)+`)`,
		page.ID+"-revision-1", page.ID, page.Locale, page.Slug, page.Title, page.Body, now)
	return err
}

func (s Store) UpdateDraftContent(ctx context.Context, tx database.Executor, page pages.Page, expectedVersion int) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, `UPDATE pages SET title = `+s.placeholder(1)+`, slug = `+s.placeholder(2)+`, body = `+s.placeholder(3)+`, locale = `+s.placeholder(4)+`, revision = revision + 1, version = version + 1, status = 'draft', updated_at = `+s.placeholder(5)+` WHERE id = `+s.placeholder(6)+` AND version = `+s.placeholder(7), page.Title, page.Slug, page.Body, page.Locale, now, page.ID, expectedVersion)
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
        (id, page_id, revision, locale, slug, title, body, created_at)
        VALUES (`+s.placeholder(1)+`, `+s.placeholder(2)+`, `+s.placeholder(3)+`, `+s.placeholder(4)+`, `+s.placeholder(5)+`, `+s.placeholder(6)+`, `+s.placeholder(7)+`, `+s.placeholder(8)+`)`,
		fmt.Sprintf("%s-revision-%d", page.ID, expectedVersion+1), page.ID, expectedVersion+1, page.Locale, page.Slug, page.Title, page.Body, now)
	return err
}

func (s Store) Publish(ctx context.Context, tx database.Executor, pageID string, expectedVersion int) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, `UPDATE pages SET status = 'published', published_revision = revision, published_locale = locale, published_slug = slug, published_title = title, published_body = body, publication_generation = publication_generation + 1, version = version + 1, updated_at = `+s.placeholder(1)+` WHERE id = `+s.placeholder(2)+` AND version = `+s.placeholder(3)+` AND status IN ('draft', 'published')`, now, pageID, expectedVersion)
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
