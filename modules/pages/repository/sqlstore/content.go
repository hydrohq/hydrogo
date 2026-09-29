package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

// ListPublishedRoutes returns only the public projection used by discovery surfaces.
// The hard cap protects sitemap generation from unbounded memory/response work;
// installations exceeding it should partition sitemap generation by locale.
const maxSitemapRoutes = 50000

var ErrInvalidListLimit = errors.New("page list limit must be between 1 and 100")

func (s Store) ListPublishedRoutes(ctx context.Context, db *sql.DB) ([]pages.Page, error) {
	if db == nil {
		return nil, database.ErrDatabaseRequired
	}
	filter, scopeID, err := s.tenantFilter(ctx, 1)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, published_locale, published_slug, published_title, status, revision, publication_generation, updated_at FROM pages WHERE published_revision IS NOT NULL` + filter + ` ORDER BY published_locale, published_slug, id LIMIT ` + fmt.Sprint(maxSitemapRoutes)
	args := []any{}
	if filter != "" {
		args = append(args, scopeID)
	}
	rows, err := db.QueryContext(ctx, query, args...)
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

// StreamPublishedRoutes emits the public sitemap projection row by row. It is
// bounded by the driver's cursor and does not materialize the route catalog.
func (s Store) StreamPublishedRoutes(ctx context.Context, db database.QueryExecutor, visit func(pages.Page) error) error {
	if db == nil {
		return database.ErrDatabaseRequired
	}
	if visit == nil {
		return errors.New("published route visitor is required")
	}
	filter, scopeID, err := s.tenantFilter(ctx, 1)
	if err != nil {
		return err
	}
	query := `SELECT id,published_locale,published_slug,published_title,status,revision,publication_generation,updated_at FROM pages WHERE published_revision IS NOT NULL` + filter + ` ORDER BY published_locale,published_slug,id`
	args := []any{}
	if filter != "" {
		args = append(args, scopeID)
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var page pages.Page
		var status string
		var updated any
		if err := rows.Scan(&page.ID, &page.Locale, &page.Slug, &page.Title, &status, &page.Revision, &page.PublicationGeneration, &updated); err != nil {
			return err
		}
		page.Status = pages.Status(status)
		page.UpdatedAt = parseStoredTime(updated)
		if err := visit(page); err != nil {
			return err
		}
	}
	return rows.Err()
}

// ListPublished returns bounded public page summaries for a locale. Draft
// fields and unpublished routes are intentionally absent from this query.
func (s Store) ListPublished(ctx context.Context, db database.Executor, locale string, limit int) ([]pages.Page, error) {
	if limit < 1 || limit > 100 {
		return nil, ErrInvalidListLimit
	}
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("published page list requires a query-capable executor")
	}
	filter, scopeID, err := s.tenantFilter(ctx, 3)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, published_locale, published_slug, published_title, status, revision, publication_generation, updated_at FROM pages WHERE published_locale = ` + s.placeholder(1) + ` AND published_revision IS NOT NULL` + filter + ` ORDER BY updated_at DESC, id ASC LIMIT ` + s.placeholder(2)
	args := []any{locale, limit}
	if filter != "" {
		if s.Dialect == database.Postgres {
			args = append(args, scopeID)
		} else {
			args = []any{locale, scopeID, limit}
		}
	}
	rows, err := queryer.QueryContext(ctx, query, args...)
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

// ListPublishedTranslations returns only live routes sharing a published slug.
// Draft translations are deliberately invisible to SEO alternate links.
func (s Store) ListPublishedTranslations(ctx context.Context, db database.Executor, slug string, limit int) ([]pages.Page, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" || len(slug) > 240 || limit < 1 || limit > 100 {
		return nil, ErrInvalidListLimit
	}
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("published translation list requires a query-capable executor")
	}
	filter, scopeID, err := s.tenantFilter(ctx, 3)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, published_locale, published_slug FROM pages WHERE published_slug = ` + s.placeholder(1) + ` AND published_revision IS NOT NULL` + filter + ` ORDER BY published_locale ASC LIMIT ` + s.placeholder(2)
	args := []any{slug, limit}
	if filter != "" {
		if s.Dialect == database.Postgres {
			args = append(args, scopeID)
		} else {
			args = []any{slug, scopeID, limit}
		}
	}
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]pages.Page, 0)
	for rows.Next() {
		var page pages.Page
		if err := rows.Scan(&page.ID, &page.Locale, &page.Slug); err != nil {
			return nil, err
		}
		result = append(result, page)
	}
	return result, rows.Err()
}

// ListRelatedPublished returns a bounded set of published pages ranked by
// shared taxonomy tags. The lookup joins the tenant/tag index and never reads
// draft relations.
func (s Store) ListRelatedPublished(ctx context.Context, db database.Executor, sourceID, locale string, limit int) ([]pages.Page, error) {
	if limit < 1 || limit > 10 {
		return nil, ErrInvalidListLimit
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("related page query requires row queries")
	}
	var query string
	var args []any
	if s.Dialect == database.Postgres {
		scopeID := ""
		if s.RequireTenantScope {
			scope, scopeErr := tenant.Require(ctx)
			if scopeErr != nil {
				return nil, scopeErr
			}
			scopeID = scope.TenantID
		}
		query = `SELECT p.id,p.published_locale,p.published_slug,p.published_title,p.status,p.revision,p.publication_generation,p.updated_at FROM page_published_tags source JOIN page_published_tags candidate ON candidate.tenant_id=source.tenant_id AND candidate.tag=source.tag JOIN pages p ON p.id=candidate.page_id AND p.tenant_id=candidate.tenant_id WHERE source.page_id=$1 AND source.tenant_id=$2 AND p.tenant_id=$2 AND p.published_locale=$3 AND p.status='published' AND p.published_revision IS NOT NULL AND p.id<>$1 GROUP BY p.id ORDER BY COUNT(source.tag) DESC,p.updated_at DESC,p.id ASC LIMIT $4`
		args = []any{sourceID, scopeID, locale, limit}
	} else {
		scopeID := ""
		if s.RequireTenantScope {
			scope, scopeErr := tenant.Require(ctx)
			if scopeErr != nil {
				return nil, scopeErr
			}
			scopeID = scope.TenantID
		}
		query = `SELECT p.id,p.published_locale,p.published_slug,p.published_title,p.status,p.revision,p.publication_generation,p.updated_at FROM page_published_tags source CROSS JOIN page_published_tags candidate JOIN pages p ON p.id=candidate.page_id AND p.tenant_id=candidate.tenant_id WHERE candidate.tenant_id=source.tenant_id AND candidate.tag=source.tag AND source.page_id=? AND source.tenant_id=? AND p.tenant_id=? AND p.published_locale=? AND p.status='published' AND p.published_revision IS NOT NULL AND p.id<>? GROUP BY p.id ORDER BY COUNT(source.tag) DESC,p.updated_at DESC,p.id ASC LIMIT ?`
		args = []any{sourceID, scopeID, scopeID, locale, sourceID, limit}
	}
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]pages.Page, 0, limit)
	for rows.Next() {
		var page pages.Page
		var status string
		var updated any
		if err := rows.Scan(&page.ID, &page.Locale, &page.Slug, &page.Title, &status, &page.Revision, &page.PublicationGeneration, &updated); err != nil {
			return nil, err
		}
		page.Status = pages.Status(status)
		page.UpdatedAt = parseStoredTime(updated)
		items = append(items, page)
	}
	return items, rows.Err()
}

// ListAdmin returns bounded protected Studio summaries, including unpublished
// records, without selecting body or block payloads.
func (s Store) ListAdmin(ctx context.Context, db database.Executor, limit int) ([]pages.Page, error) {
	if limit < 1 || limit > 100 {
		return nil, ErrInvalidListLimit
	}
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("page admin list requires a query-capable executor")
	}
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, locale, slug, title, status, version, revision, updated_at FROM pages WHERE 1=1` + filter + ` ORDER BY updated_at DESC, id ASC LIMIT ` + s.placeholder(1)
	args := []any{limit}
	if filter != "" {
		if s.Dialect == database.Postgres {
			args = append(args, scopeID)
		} else {
			args = []any{scopeID, limit}
		}
	}
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]pages.Page, 0)
	for rows.Next() {
		var page pages.Page
		var status string
		var updated any
		if err := rows.Scan(&page.ID, &page.Locale, &page.Slug, &page.Title, &status, &page.Version, &page.Revision, &updated); err != nil {
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
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return pages.Page{}, err
	}
	query := `SELECT id, locale, slug, title, body, blocks, status, version, revision, published_revision, publication_generation, created_at, updated_at FROM pages WHERE id = ` + s.placeholder(1) + filter
	args := []any{pageID}
	if filter != "" {
		args = append(args, scopeID)
	}
	err = db.QueryRowContext(ctx, query, args...).Scan(&page.ID, &page.Locale, &page.Slug, &page.Title, &page.Body, &blocks, &status, &page.Version, &page.Revision, &published, &page.PublicationGeneration, &createdAt, &updatedAt)
	if err != nil {
		return pages.Page{}, err
	}
	page.Status = pages.Status(status)
	page.Blocks, err = pages.DecodeBlocksForStore(blocks)
	if err != nil {
		return pages.Page{}, err
	}
	page.Tags, err = s.listPageTags(ctx, db, page.ID, "page_tags")
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
	filter, scopeID, err := s.tenantFilter(ctx, 3)
	if err != nil {
		return pages.Page{}, err
	}
	query := `SELECT id, published_locale, published_slug, published_title, published_body, published_blocks, status, version, revision, published_revision, publication_generation, created_at, updated_at FROM pages WHERE published_locale = ` + s.placeholder(1) + ` AND published_slug = ` + s.placeholder(2) + ` AND published_revision IS NOT NULL` + filter
	args := []any{locale, slug}
	if filter != "" {
		args = append(args, scopeID)
	}
	err = db.QueryRowContext(ctx, query, args...).Scan(&page.ID, &page.Locale, &page.Slug, &page.Title, &page.Body, &blocks, &status, &page.Version, &page.Revision, &published, &page.PublicationGeneration, &createdAt, &updatedAt)
	if err != nil {
		return pages.Page{}, err
	}
	page.Status = pages.Status(status)
	page.Blocks, err = pages.DecodeBlocksForStore(blocks)
	if err != nil {
		return pages.Page{}, err
	}
	page.Tags, err = s.listPageTags(ctx, db, page.ID, "page_published_tags")
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

func (s Store) storedTime(value time.Time) any {
	value = value.UTC()
	if s.Dialect == database.SQLite {
		return value.Format(time.RFC3339Nano)
	}
	return value
}

func (s Store) CreateDraftContent(ctx context.Context, tx database.Executor, page pages.Page) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := s.storedTime(time.Now().UTC())
	scopeID := ""
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return scopeErr
		}
		scopeID = scope.TenantID
	}
	blocks, err := pages.EncodeBlocksForStore(page.Blocks)
	if err != nil {
		return err
	}
	pageQuery := `INSERT INTO pages
		(id, title, version, locale, slug, body, blocks, status, revision, publication_generation, created_at, updated_at`
	pageValues := `VALUES (` + s.placeholder(1) + `, ` + s.placeholder(2) + `, 1, ` + s.placeholder(3) + `, ` + s.placeholder(4) + `, ` + s.placeholder(5) + `, ` + s.placeholder(6) + `, 'draft', 1, 0, ` + s.placeholder(7) + `, ` + s.placeholder(8)
	pageArgs := []any{page.ID, page.Title, page.Locale, page.Slug, page.Body, blocks, now, now}
	if s.RequireTenantScope {
		pageQuery += `, tenant_id`
		pageValues += `, ` + s.placeholder(9)
		pageArgs = append(pageArgs, scopeID)
	}
	pageQuery += `) ` + pageValues + `)`
	_, err = tx.ExecContext(ctx, pageQuery, pageArgs...)
	if err != nil {
		return err
	}
	if err := s.replacePageTags(ctx, tx, page.ID, page.Tags, "page_tags"); err != nil {
		return err
	}
	revisionQuery := `INSERT INTO page_revisions
		(id, page_id, revision, locale, slug, title, body, blocks, actor_id, created_at`
	revisionValues := `VALUES (` + s.placeholder(1) + `, ` + s.placeholder(2) + `, 1, ` + s.placeholder(3) + `, ` + s.placeholder(4) + `, ` + s.placeholder(5) + `, ` + s.placeholder(6) + `, ` + s.placeholder(7) + `, ` + s.placeholder(8) + `, ` + s.placeholder(9)
	revisionActor := ""
	if actor := auth.UserFromContext(ctx); actor != nil {
		revisionActor = actor.ID
	}
	revisionArgs := []any{page.ID + "-revision-1", page.ID, page.Locale, page.Slug, page.Title, page.Body, blocks, revisionActor, now}
	if s.RequireTenantScope {
		revisionQuery += `, tenant_id`
		revisionValues += `, ` + s.placeholder(10)
		revisionArgs = append(revisionArgs, scopeID)
	}
	revisionQuery += `) ` + revisionValues + `)`
	_, err = tx.ExecContext(ctx, revisionQuery, revisionArgs...)
	return err
}

func (s Store) UpdateDraftContent(ctx context.Context, tx database.Executor, page pages.Page, expectedVersion int) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := s.storedTime(time.Now().UTC())
	scopeID := ""
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return scopeErr
		}
		scopeID = scope.TenantID
	}
	blocks, err := pages.EncodeBlocksForStore(page.Blocks)
	if err != nil {
		return err
	}
	filter := ""
	args := []any{page.Title, page.Slug, page.Body, blocks, page.Locale, now, page.ID, expectedVersion}
	if s.RequireTenantScope {
		filter = ` AND tenant_id = ` + s.placeholder(9)
		args = append(args, scopeID)
	}
	result, err := tx.ExecContext(ctx, `UPDATE pages SET title = `+s.placeholder(1)+`, slug = `+s.placeholder(2)+`, body = `+s.placeholder(3)+`, blocks = `+s.placeholder(4)+`, locale = `+s.placeholder(5)+`, revision = revision + 1, version = version + 1, status = 'draft', updated_at = `+s.placeholder(6)+` WHERE id = `+s.placeholder(7)+` AND version = `+s.placeholder(8)+filter, args...)
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
	if err := s.replacePageTags(ctx, tx, page.ID, page.Tags, "page_tags"); err != nil {
		return err
	}
	revisionQuery := `INSERT INTO page_revisions
		(id, page_id, revision, locale, slug, title, body, blocks, actor_id, created_at`
	revisionValues := `VALUES (` + s.placeholder(1) + `, ` + s.placeholder(2) + `, ` + s.placeholder(3) + `, ` + s.placeholder(4) + `, ` + s.placeholder(5) + `, ` + s.placeholder(6) + `, ` + s.placeholder(7) + `, ` + s.placeholder(8) + `, ` + s.placeholder(9) + `, ` + s.placeholder(10)
	revisionActor := ""
	if actor := auth.UserFromContext(ctx); actor != nil {
		revisionActor = actor.ID
	}
	revisionArgs := []any{fmt.Sprintf("%s-revision-%d", page.ID, expectedVersion+1), page.ID, expectedVersion + 1, page.Locale, page.Slug, page.Title, page.Body, blocks, revisionActor, now}
	if s.RequireTenantScope {
		revisionQuery += `, tenant_id`
		revisionValues += `, ` + s.placeholder(11)
		revisionArgs = append(revisionArgs, scopeID)
	}
	revisionQuery += `) ` + revisionValues + `)`
	_, err = tx.ExecContext(ctx, revisionQuery, revisionArgs...)
	return err
}

func (s Store) Publish(ctx context.Context, tx database.Executor, pageID string, expectedVersion int) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := s.storedTime(time.Now().UTC())
	filter, scopeID, err := s.tenantFilter(ctx, 4)
	if err != nil {
		return err
	}
	args := []any{now, pageID, expectedVersion}
	if filter != "" {
		args = append(args, scopeID)
	}
	result, err := tx.ExecContext(ctx, `UPDATE pages SET status = 'published', published_revision = revision, published_locale = locale, published_slug = slug, published_title = title, published_body = body, published_blocks = blocks, publication_generation = publication_generation + 1, version = version + 1, updated_at = `+s.placeholder(1)+` WHERE id = `+s.placeholder(2)+` AND version = `+s.placeholder(3)+` AND status IN ('draft', 'published')`+filter, args...)
	if err != nil {
		if database.IsUniqueViolation(err) {
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
	tagFilter, tagScope, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return err
	}
	deleteArgs := []any{pageID}
	if tagFilter != "" {
		deleteArgs = append(deleteArgs, tagScope)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM page_published_tags WHERE page_id = `+s.placeholder(1)+tagFilter, deleteArgs...); err != nil {
		return err
	}
	copyQuery := `INSERT INTO page_published_tags (tenant_id,page_id,tag) SELECT tenant_id,page_id,tag FROM page_tags WHERE page_id=` + s.placeholder(1)
	copyArgs := []any{pageID}
	if filter != "" {
		copyQuery += ` AND tenant_id=` + s.placeholder(2)
		copyArgs = append(copyArgs, scopeID)
	}
	if _, err := tx.ExecContext(ctx, copyQuery, copyArgs...); err != nil {
		return err
	}
	return nil
}

func (s Store) replacePageTags(ctx context.Context, tx database.Executor, pageID string, tags []string, relation string) error {
	if relation != "page_tags" && relation != "page_published_tags" {
		return errors.New("invalid page tag relation")
	}
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return err
	}
	args := []any{pageID}
	query := `DELETE FROM ` + relation + ` WHERE page_id=` + s.placeholder(1) + filter
	if filter != "" {
		args = append(args, scopeID)
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || len(tag) > 64 {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		insert := `INSERT INTO ` + relation + ` (tenant_id,page_id,tag) VALUES (` + s.placeholder(1) + `,` + s.placeholder(2) + `,` + s.placeholder(3) + `)`
		_, err := tx.ExecContext(ctx, insert, scopeID, pageID, tag)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s Store) listPageTags(ctx context.Context, db database.Executor, pageID, relation string) ([]string, error) {
	if relation != "page_tags" && relation != "page_published_tags" {
		return nil, errors.New("invalid page tag relation")
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("page tag lookup requires row queries")
	}
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return nil, err
	}
	query := `SELECT tag FROM ` + relation + ` WHERE page_id=` + s.placeholder(1) + filter + ` ORDER BY tag`
	args := []any{pageID}
	if filter != "" {
		args = append(args, scopeID)
	}
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := make([]string, 0, 8)
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}
