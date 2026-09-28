package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

type Store struct {
	Dialect            database.Dialect
	RequireTenantScope bool
}

var ErrInvalidListLimit = errors.New("news list limit must be between 1 and 100")

func (s Store) p(n int) string {
	if s.Dialect == database.SQLite {
		return "?"
	}
	return fmt.Sprintf("$%d", n)
}

func (s Store) tenantFilter(ctx context.Context, position int) (string, any, error) {
	if !s.RequireTenantScope {
		return "", nil, nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", nil, err
	}
	return " AND tenant_id = " + s.p(position), scope.TenantID, nil
}

func (s Store) CreateDraft(ctx context.Context, tx database.Executor, item news.Item) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := time.Now().UTC()
	scopeID := ""
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return scopeErr
		}
		scopeID = scope.TenantID
	}
	query := `INSERT INTO news_items (id,locale,slug,title,body,cover_media_id,excerpt,key_takeaways,category,author_name,featured,tags,seo_title,seo_description,status,version,revision,schedule_generation,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,'draft',1,1,0,$15,$16)`
	args := []any{item.ID, item.Locale, item.Slug, item.Title, item.Body, item.CoverMediaID, item.Excerpt, item.KeyTakeaways, item.Category, item.AuthorName, boolArg(s.Dialect, item.Featured), item.Tags, item.SEOTitle, item.SEODescription, now, now}
	if s.Dialect == database.SQLite {
		query = `INSERT INTO news_items (id,locale,slug,title,body,cover_media_id,excerpt,key_takeaways,category,author_name,featured,tags,seo_title,seo_description,status,version,revision,schedule_generation,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,'draft',1,1,0,?,?)`
		args[14], args[15] = now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)
	}
	if s.RequireTenantScope {
		query = `INSERT INTO news_items (id,locale,slug,title,body,cover_media_id,excerpt,key_takeaways,category,author_name,featured,tags,seo_title,seo_description,status,version,revision,schedule_generation,created_at,updated_at,tenant_id) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `,` + s.p(9) + `,` + s.p(10) + `,` + s.p(11) + `,` + s.p(12) + `,` + s.p(13) + `,` + s.p(14) + `,'draft',1,1,0,` + s.p(15) + `,` + s.p(16) + `,` + s.p(17) + `)`
		args = append(args, scopeID)
		if s.Dialect == database.SQLite {
			args[14], args[15] = now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)
		}
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (s Store) UpdateDraft(ctx context.Context, tx database.Executor, item news.Item, expectedVersion int) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := time.Now().UTC()
	filter, scopeID, err := s.tenantFilter(ctx, 17)
	if err != nil {
		return err
	}
	query := `UPDATE news_items SET locale=$1,slug=$2,title=$3,body=$4,cover_media_id=$5,excerpt=$6,key_takeaways=$7,category=$8,author_name=$9,featured=$10,tags=$11,seo_title=$12,seo_description=$13,status='draft',revision=revision+1,version=version+1,updated_at=$14 WHERE id=$15 AND version=$16 AND status IN ('draft','retracted')`
	args := []any{item.Locale, item.Slug, item.Title, item.Body, item.CoverMediaID, item.Excerpt, item.KeyTakeaways, item.Category, item.AuthorName, boolArg(s.Dialect, item.Featured), item.Tags, item.SEOTitle, item.SEODescription, now, item.ID, expectedVersion}
	if s.Dialect == database.SQLite {
		query = `UPDATE news_items SET locale=?,slug=?,title=?,body=?,cover_media_id=?,excerpt=?,key_takeaways=?,category=?,author_name=?,featured=?,tags=?,seo_title=?,seo_description=?,status='draft',revision=revision+1,version=version+1,updated_at=? WHERE id=? AND version=? AND status IN ('draft','retracted')`
		args[13] = now.Format(time.RFC3339Nano)
	}
	if filter != "" {
		query += filter
		args = append(args, scopeID)
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return news.ErrConflict
	}
	return nil
}

func (s Store) Schedule(ctx context.Context, tx database.Executor, id string, expectedVersion int, at time.Time) (int64, error) {
	if err := database.RequireExecutor(tx); err != nil {
		return 0, err
	}
	filter, scopeID, err := s.tenantFilter(ctx, 4)
	if err != nil {
		return 0, err
	}
	query := `UPDATE news_items SET status='scheduled',scheduled_at=$1,schedule_generation=schedule_generation+1,version=version+1,updated_at=$1 WHERE id=$2 AND version=$3 AND status IN ('draft','retracted')` + filter
	args := []any{at, id, expectedVersion}
	if s.Dialect == database.SQLite {
		query = `UPDATE news_items SET status='scheduled',scheduled_at=?,schedule_generation=schedule_generation+1,version=version+1,updated_at=? WHERE id=? AND version=? AND status IN ('draft','retracted')` + filter
		stamp := at.Format(time.RFC3339Nano)
		args = []any{stamp, stamp, id, expectedVersion}
	}
	if filter != "" {
		args = append(args, scopeID)
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return 0, news.ErrConflict
	}
	var generation int64
	selectFilter, selectScopeID, selectErr := s.tenantFilter(ctx, 2)
	if selectErr != nil {
		return 0, selectErr
	}
	selectArgs := []any{id}
	if selectFilter != "" {
		selectArgs = append(selectArgs, selectScopeID)
	}
	if err := tx.QueryRowContext(ctx, `SELECT schedule_generation FROM news_items WHERE id = `+s.p(1)+selectFilter, selectArgs...).Scan(&generation); err != nil {
		return 0, err
	}
	return generation, nil
}

func (s Store) PublishScheduled(ctx context.Context, tx database.Executor, id string, generation int64, now time.Time) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	filter, scopeID, err := s.tenantFilter(ctx, 4)
	if err != nil {
		return err
	}
	query := `UPDATE news_items SET status='published',published_revision=revision,scheduled_at=NULL,version=version+1,updated_at=$1 WHERE id=$2 AND status='scheduled' AND schedule_generation=$3 AND scheduled_at <= $1` + filter
	args := []any{now, id, generation}
	if s.Dialect == database.SQLite {
		stamp := now.Format(time.RFC3339Nano)
		query = `UPDATE news_items SET status='published',published_revision=revision,scheduled_at=NULL,version=version+1,updated_at=? WHERE id=? AND status='scheduled' AND schedule_generation=? AND scheduled_at <= ?` + filter
		args = []any{stamp, id, generation, stamp}
	}
	if filter != "" {
		args = append(args, scopeID)
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return news.ErrScheduleStale
	}
	return nil
}

func (s Store) PublishNow(ctx context.Context, tx database.Executor, id string, expectedVersion int) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	filter, scopeID, err := s.tenantFilter(ctx, 4)
	if err != nil {
		return err
	}
	query := `UPDATE news_items SET status='published',published_revision=revision,scheduled_at=NULL,version=version+1,updated_at=$1 WHERE id=$2 AND version=$3 AND status IN ('draft','scheduled','retracted')` + filter
	args := []any{time.Now().UTC(), id, expectedVersion}
	if s.Dialect == database.SQLite {
		stamp := time.Now().UTC().Format(time.RFC3339Nano)
		query = `UPDATE news_items SET status='published',published_revision=revision,scheduled_at=NULL,version=version+1,updated_at=? WHERE id=? AND version=? AND status IN ('draft','scheduled','retracted')` + filter
		args[0] = stamp
	}
	if filter != "" {
		args = append(args, scopeID)
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return news.ErrConflict
	}
	return nil
}

func (s Store) Retract(ctx context.Context, tx database.Executor, id string, expectedVersion int) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	filter, scopeID, err := s.tenantFilter(ctx, 4)
	if err != nil {
		return err
	}
	query := `UPDATE news_items SET status='retracted',version=version+1,updated_at=$1 WHERE id=$2 AND version=$3 AND status='published'` + filter
	args := []any{time.Now().UTC(), id, expectedVersion}
	if s.Dialect == database.SQLite {
		stamp := time.Now().UTC().Format(time.RFC3339Nano)
		query = `UPDATE news_items SET status='retracted',version=version+1,updated_at=? WHERE id=? AND version=? AND status='published'` + filter
		args[0] = stamp
	}
	if filter != "" {
		args = append(args, scopeID)
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return news.ErrConflict
	}
	return nil
}

func (s Store) Get(ctx context.Context, db database.Executor, id string) (news.Item, error) {
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return news.Item{}, err
	}
	args := []any{id}
	if filter != "" {
		args = append(args, scopeID)
	}
	query := `SELECT id,locale,slug,title,body,cover_media_id,excerpt,key_takeaways,category,author_name,featured,tags,seo_title,seo_description,status,version,revision,published_revision,schedule_generation,scheduled_at,created_at,updated_at FROM news_items WHERE id = ` + s.p(1) + filter
	return s.scan(ctx, db, query, args...)
}

func (s Store) GetPublishedByRoute(ctx context.Context, db database.Executor, locale, slug string) (news.Item, error) {
	filter, scopeID, err := s.tenantFilter(ctx, 3)
	if err != nil {
		return news.Item{}, err
	}
	args := []any{locale, slug}
	if filter != "" {
		args = append(args, scopeID)
	}
	query := `SELECT id,locale,slug,title,body,cover_media_id,excerpt,key_takeaways,category,author_name,featured,tags,seo_title,seo_description,status,version,revision,published_revision,schedule_generation,scheduled_at,created_at,updated_at FROM news_items WHERE locale = ` + s.p(1) + ` AND slug = ` + s.p(2) + ` AND status='published' AND published_revision IS NOT NULL` + filter
	return s.scan(ctx, db, query, args...)
}

// ListPublished returns only the published projection for a locale. Callers
// supply a bounded limit; drafts, scheduled and retracted items never cross
// this repository boundary into a public response.
func (s Store) ListPublished(ctx context.Context, db database.Executor, locale string, limit int) ([]news.Item, error) {
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
		return nil, errors.New("news published list requires a query-capable executor")
	}
	filter, scopeID, err := s.tenantFilter(ctx, 3)
	if err != nil {
		return nil, err
	}
	query := `SELECT id,locale,slug,title,body,cover_media_id,excerpt,key_takeaways,category,author_name,featured,tags,seo_title,seo_description,status,version,revision,published_revision,schedule_generation,scheduled_at,created_at,updated_at FROM news_items WHERE locale = ` + s.p(1) + ` AND status='published' AND published_revision IS NOT NULL` + filter + ` ORDER BY updated_at DESC, id ASC LIMIT ` + s.p(2)
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
	items := make([]news.Item, 0)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// StreamPublishedRoutes emits only the public route projection for sitemap
// generation. Rows are read in key order without loading the catalog.
func (s Store) StreamPublishedRoutes(ctx context.Context, db database.QueryExecutor, visit func(news.Item) error) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if visit == nil {
		return errors.New("published route visitor is required")
	}
	filter, scopeID, err := s.tenantFilter(ctx, 1)
	if err != nil {
		return err
	}
	query := `SELECT id,locale,slug,title,seo_description,updated_at FROM news_items WHERE status='published' AND published_revision IS NOT NULL` + filter + ` ORDER BY locale,slug,id`
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
		var item news.Item
		var updated any
		if err := rows.Scan(&item.ID, &item.Locale, &item.Slug, &item.Title, &item.SEODescription, &updated); err != nil {
			return err
		}
		item.UpdatedAt = parseTime(updated)
		if err := visit(item); err != nil {
			return err
		}
	}
	return rows.Err()
}

// ListRelatedPublished returns a bounded published projection sharing at least
// one taxonomy term with the source item. It never exposes drafts or relation
// storage details to callers.
func (s Store) ListRelatedPublished(ctx context.Context, db database.Executor, contentID, locale string, limit int) ([]news.Item, error) {
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
		return nil, errors.New("related news query requires a query-capable executor")
	}
	filter, scopeID, err := s.tenantFilter(ctx, 4)
	if err != nil {
		return nil, err
	}
	query := `SELECT n.id,n.locale,n.slug,n.title,n.category,n.cover_media_id,n.excerpt,n.updated_at,COUNT(DISTINCT rel.term_id) AS relevance FROM news_items n JOIN news_taxonomy_relations rel ON rel.content_id=n.id WHERE rel.term_id IN (SELECT term_id FROM news_taxonomy_relations WHERE content_id=$1) AND n.id<>$1 AND n.locale=$2 AND n.status='published' AND n.published_revision IS NOT NULL` + filter + ` GROUP BY n.id,n.locale,n.slug,n.title,n.category,n.cover_media_id,n.excerpt,n.updated_at ORDER BY relevance DESC,n.updated_at DESC,n.id ASC LIMIT $3`
	if s.Dialect == database.SQLite {
		query = `SELECT n.id,n.locale,n.slug,n.title,n.category,n.cover_media_id,n.excerpt,n.updated_at,COUNT(DISTINCT rel.term_id) AS relevance FROM news_items n JOIN news_taxonomy_relations rel ON rel.content_id=n.id WHERE rel.term_id IN (SELECT term_id FROM news_taxonomy_relations WHERE content_id=?) AND n.id<>? AND n.locale=? AND n.status='published' AND n.published_revision IS NOT NULL` + filter + ` GROUP BY n.id,n.locale,n.slug,n.title,n.category,n.cover_media_id,n.excerpt,n.updated_at ORDER BY relevance DESC,n.updated_at DESC,n.id ASC LIMIT ?`
	}
	args := []any{contentID, locale, limit}
	if s.Dialect == database.SQLite {
		args = []any{contentID, contentID, locale, limit}
	}
	if filter != "" {
		if s.Dialect == database.Postgres {
			args = []any{contentID, locale, limit, scopeID}
		} else {
			args = []any{contentID, contentID, locale, scopeID, limit}
		}
	}
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]news.Item, 0, limit)
	for rows.Next() {
		var item news.Item
		var updated any
		var relevance int
		if err := rows.Scan(&item.ID, &item.Locale, &item.Slug, &item.Title, &item.Category, &item.CoverMediaID, &item.Excerpt, &updated, &relevance); err != nil {
			return nil, err
		}
		item.UpdatedAt = parseTime(updated)
		item.Status = news.StatusPublished
		items = append(items, item)
	}
	return items, rows.Err()
}

// ListAdmin returns bounded editorial summaries for the protected Studio list.
// Unlike ListPublished it may include drafts, but never returns body content.
func (s Store) ListAdmin(ctx context.Context, db database.Executor, limit int) ([]news.Item, error) {
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
		return nil, errors.New("news admin list requires a query-capable executor")
	}
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return nil, err
	}
	query := `SELECT id,locale,slug,title,body,cover_media_id,excerpt,key_takeaways,category,author_name,featured,tags,seo_title,seo_description,status,version,revision,published_revision,schedule_generation,scheduled_at,created_at,updated_at FROM news_items WHERE 1=1` + filter + ` ORDER BY updated_at DESC, id ASC LIMIT ` + s.p(1)
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
	items := make([]news.Item, 0)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s Store) ListScheduled(ctx context.Context, db database.Executor, now time.Time) ([]news.Item, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return nil, err
	}
	query := `SELECT id,schedule_generation,scheduled_at FROM news_items WHERE status='scheduled' AND scheduled_at IS NOT NULL AND scheduled_at <= ` + s.p(1) + filter + ` ORDER BY scheduled_at,id LIMIT 100`
	argument := any(now.UTC())
	if s.Dialect == database.SQLite {
		argument = now.UTC().Format(time.RFC3339Nano)
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("news scheduled query requires a query-capable executor")
	}
	args := []any{argument}
	if filter != "" {
		args = append(args, scopeID)
	}
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]news.Item, 0)
	for rows.Next() {
		var item news.Item
		var scheduled any
		if err := rows.Scan(&item.ID, &item.ScheduleGeneration, &scheduled); err != nil {
			return nil, err
		}
		value := parseTime(scheduled)
		item.ScheduledAt = &value
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s Store) scan(ctx context.Context, db database.Executor, query string, args ...any) (news.Item, error) {
	if err := database.RequireExecutor(db); err != nil {
		return news.Item{}, err
	}
	var item news.Item
	var status string
	var published any
	var featured any
	var excerpt, keyTakeaways, category, authorName, tags, seoTitle, seoDescription string
	var scheduled, created, updated any
	err := db.QueryRowContext(ctx, query, args...).Scan(&item.ID, &item.Locale, &item.Slug, &item.Title, &item.Body, &item.CoverMediaID, &excerpt, &keyTakeaways, &category, &authorName, &featured, &tags, &seoTitle, &seoDescription, &status, &item.Version, &item.Revision, &published, &item.ScheduleGeneration, &scheduled, &created, &updated)
	if err != nil {
		return news.Item{}, err
	}
	item.Excerpt, item.KeyTakeaways, item.Category, item.AuthorName, item.Tags, item.SEOTitle, item.SEODescription = excerpt, keyTakeaways, category, authorName, tags, seoTitle, seoDescription
	item.Featured = boolValue(featured)
	return itemFromColumns(item, status, published, scheduled, created, updated), nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanItem(row rowScanner) (news.Item, error) {
	var item news.Item
	var status string
	var published any
	var featured any
	var excerpt, keyTakeaways, category, authorName, tags, seoTitle, seoDescription string
	var scheduled, created, updated any
	if err := row.Scan(&item.ID, &item.Locale, &item.Slug, &item.Title, &item.Body, &item.CoverMediaID, &excerpt, &keyTakeaways, &category, &authorName, &featured, &tags, &seoTitle, &seoDescription, &status, &item.Version, &item.Revision, &published, &item.ScheduleGeneration, &scheduled, &created, &updated); err != nil {
		return news.Item{}, err
	}
	item.Excerpt, item.KeyTakeaways, item.Category, item.AuthorName, item.Tags, item.SEOTitle, item.SEODescription = excerpt, keyTakeaways, category, authorName, tags, seoTitle, seoDescription
	item.Featured = boolValue(featured)
	return itemFromColumns(item, status, published, scheduled, created, updated), nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func boolArg(dialect database.Dialect, value bool) any {
	if dialect == database.Postgres {
		return value
	}
	return boolInt(value)
}

func boolValue(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case int64:
		return v != 0
	case int:
		return v != 0
	case []byte:
		return string(v) == "1" || string(v) == "true"
	case string:
		return v == "1" || v == "true"
	default:
		return false
	}
}

func itemFromColumns(item news.Item, status string, published, scheduled, created, updated any) news.Item {
	item.Status = news.Status(status)
	if published != nil {
		var value int
		switch v := published.(type) {
		case int64:
			value = int(v)
		case int:
			value = v
		}
		item.PublishedRevision = &value
	}
	if scheduled != nil {
		value := parseTime(scheduled)
		if !value.IsZero() {
			item.ScheduledAt = &value
		}
	}
	item.CreatedAt, item.UpdatedAt = parseTime(created), parseTime(updated)
	return item
}

func parseTime(value any) time.Time {
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
