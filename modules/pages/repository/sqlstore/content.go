package sqlstore

import (
	"context"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

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
	result, err := tx.ExecContext(ctx, `UPDATE pages SET title = `+s.placeholder(1)+`, slug = `+s.placeholder(2)+`, body = `+s.placeholder(3)+`, locale = `+s.placeholder(4)+`, revision = revision + 1, version = version + 1, updated_at = `+s.placeholder(5)+` WHERE id = `+s.placeholder(6)+` AND version = `+s.placeholder(7)+` AND status = 'draft'`, page.Title, page.Slug, page.Body, page.Locale, now, page.ID, expectedVersion)
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
	result, err := tx.ExecContext(ctx, `UPDATE pages SET status = 'published', published_revision = revision, publication_generation = publication_generation + 1, version = version + 1, updated_at = `+s.placeholder(1)+` WHERE id = `+s.placeholder(2)+` AND version = `+s.placeholder(3)+` AND status = 'draft'`, now, pageID, expectedVersion)
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
	return nil
}
