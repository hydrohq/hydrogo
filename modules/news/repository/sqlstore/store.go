package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

type Store struct{ Dialect database.Dialect }

var ErrInvalidListLimit = errors.New("news list limit must be between 1 and 100")

func (s Store) p(n int) string {
	if s.Dialect == database.SQLite {
		return "?"
	}
	return fmt.Sprintf("$%d", n)
}

func (s Store) CreateDraft(ctx context.Context, tx database.Executor, item news.Item) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := time.Now().UTC()
	query := `INSERT INTO news_items (id,locale,slug,title,body,cover_media_id,excerpt,key_takeaways,category,author_name,featured,tags,seo_title,seo_description,status,version,revision,schedule_generation,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,'draft',1,1,0,$15,$16)`
	args := []any{item.ID, item.Locale, item.Slug, item.Title, item.Body, item.CoverMediaID, item.Excerpt, item.KeyTakeaways, item.Category, item.AuthorName, boolArg(s.Dialect, item.Featured), item.Tags, item.SEOTitle, item.SEODescription, now, now}
	if s.Dialect == database.SQLite {
		query = `INSERT INTO news_items (id,locale,slug,title,body,cover_media_id,excerpt,key_takeaways,category,author_name,featured,tags,seo_title,seo_description,status,version,revision,schedule_generation,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,'draft',1,1,0,?,?)`
		args[14], args[15] = now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (s Store) UpdateDraft(ctx context.Context, tx database.Executor, item news.Item, expectedVersion int) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := time.Now().UTC()
	query := `UPDATE news_items SET locale=$1,slug=$2,title=$3,body=$4,cover_media_id=$5,excerpt=$6,key_takeaways=$7,category=$8,author_name=$9,featured=$10,tags=$11,seo_title=$12,seo_description=$13,status='draft',revision=revision+1,version=version+1,updated_at=$14 WHERE id=$15 AND version=$16 AND status IN ('draft','retracted')`
	args := []any{item.Locale, item.Slug, item.Title, item.Body, item.CoverMediaID, item.Excerpt, item.KeyTakeaways, item.Category, item.AuthorName, boolArg(s.Dialect, item.Featured), item.Tags, item.SEOTitle, item.SEODescription, now, item.ID, expectedVersion}
	if s.Dialect == database.SQLite {
		query = `UPDATE news_items SET locale=?,slug=?,title=?,body=?,cover_media_id=?,excerpt=?,key_takeaways=?,category=?,author_name=?,featured=?,tags=?,seo_title=?,seo_description=?,status='draft',revision=revision+1,version=version+1,updated_at=? WHERE id=? AND version=? AND status IN ('draft','retracted')`
		args[13] = now.Format(time.RFC3339Nano)
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
	query := `UPDATE news_items SET status='scheduled',scheduled_at=$1,schedule_generation=schedule_generation+1,version=version+1,updated_at=$1 WHERE id=$2 AND version=$3 AND status IN ('draft','retracted')`
	args := []any{at, id, expectedVersion}
	if s.Dialect == database.SQLite {
		query = `UPDATE news_items SET status='scheduled',scheduled_at=?,schedule_generation=schedule_generation+1,version=version+1,updated_at=? WHERE id=? AND version=? AND status IN ('draft','retracted')`
		stamp := at.Format(time.RFC3339Nano)
		args = []any{stamp, stamp, id, expectedVersion}
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
	if err := tx.QueryRowContext(ctx, `SELECT schedule_generation FROM news_items WHERE id = `+s.p(1), id).Scan(&generation); err != nil {
		return 0, err
	}
	return generation, nil
}

func (s Store) PublishScheduled(ctx context.Context, tx database.Executor, id string, generation int64, now time.Time) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	query := `UPDATE news_items SET status='published',published_revision=revision,scheduled_at=NULL,version=version+1,updated_at=$1 WHERE id=$2 AND status='scheduled' AND schedule_generation=$3 AND scheduled_at <= $1`
	args := []any{now, id, generation}
	if s.Dialect == database.SQLite {
		stamp := now.Format(time.RFC3339Nano)
		query = `UPDATE news_items SET status='published',published_revision=revision,scheduled_at=NULL,version=version+1,updated_at=? WHERE id=? AND status='scheduled' AND schedule_generation=? AND scheduled_at <= ?`
		args = []any{stamp, id, generation, stamp}
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
	query := `UPDATE news_items SET status='published',published_revision=revision,scheduled_at=NULL,version=version+1,updated_at=$1 WHERE id=$2 AND version=$3 AND status IN ('draft','scheduled','retracted')`
	args := []any{time.Now().UTC(), id, expectedVersion}
	if s.Dialect == database.SQLite {
		stamp := time.Now().UTC().Format(time.RFC3339Nano)
		query = `UPDATE news_items SET status='published',published_revision=revision,scheduled_at=NULL,version=version+1,updated_at=? WHERE id=? AND version=? AND status IN ('draft','scheduled','retracted')`
		args[0] = stamp
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
	query := `UPDATE news_items SET status='retracted',version=version+1,updated_at=$1 WHERE id=$2 AND version=$3 AND status='published'`
	args := []any{time.Now().UTC(), id, expectedVersion}
	if s.Dialect == database.SQLite {
		stamp := time.Now().UTC().Format(time.RFC3339Nano)
		query = `UPDATE news_items SET status='retracted',version=version+1,updated_at=? WHERE id=? AND version=? AND status='published'`
		args[0] = stamp
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
	query := `SELECT id,locale,slug,title,body,cover_media_id,excerpt,key_takeaways,category,author_name,featured,tags,seo_title,seo_description,status,version,revision,published_revision,schedule_generation,scheduled_at,created_at,updated_at FROM news_items WHERE id = ` + s.p(1)
	return s.scan(ctx, db, query, id)
}

func (s Store) GetPublishedByRoute(ctx context.Context, db database.Executor, locale, slug string) (news.Item, error) {
	query := `SELECT id,locale,slug,title,body,cover_media_id,excerpt,key_takeaways,category,author_name,featured,tags,seo_title,seo_description,status,version,revision,published_revision,schedule_generation,scheduled_at,created_at,updated_at FROM news_items WHERE locale = ` + s.p(1) + ` AND slug = ` + s.p(2) + ` AND status='published' AND published_revision IS NOT NULL`
	return s.scan(ctx, db, query, locale, slug)
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
	query := `SELECT id,locale,slug,title,body,cover_media_id,excerpt,key_takeaways,category,author_name,featured,tags,seo_title,seo_description,status,version,revision,published_revision,schedule_generation,scheduled_at,created_at,updated_at FROM news_items WHERE locale = ` + s.p(1) + ` AND status='published' AND published_revision IS NOT NULL ORDER BY updated_at DESC, id ASC LIMIT ` + s.p(2)
	rows, err := queryer.QueryContext(ctx, query, locale, limit)
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
	query := `SELECT DISTINCT n.id,n.locale,n.slug,n.title,n.category,n.updated_at FROM news_items n JOIN news_taxonomy_relations rel ON rel.content_id=n.id WHERE rel.term_id IN (SELECT term_id FROM news_taxonomy_relations WHERE content_id=$1) AND n.id<>$1 AND n.locale=$2 AND n.status='published' AND n.published_revision IS NOT NULL ORDER BY n.updated_at DESC,n.id ASC LIMIT $3`
	if s.Dialect == database.SQLite {
		query = `SELECT DISTINCT n.id,n.locale,n.slug,n.title,n.category,n.updated_at FROM news_items n JOIN news_taxonomy_relations rel ON rel.content_id=n.id WHERE rel.term_id IN (SELECT term_id FROM news_taxonomy_relations WHERE content_id=?) AND n.id<>? AND n.locale=? AND n.status='published' AND n.published_revision IS NOT NULL ORDER BY n.updated_at DESC,n.id ASC LIMIT ?`
	}
	args := []any{contentID, locale, limit}
	if s.Dialect == database.SQLite {
		args = []any{contentID, contentID, locale, limit}
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
		if err := rows.Scan(&item.ID, &item.Locale, &item.Slug, &item.Title, &item.Category, &updated); err != nil {
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
	query := `SELECT id,locale,slug,title,body,cover_media_id,excerpt,key_takeaways,category,author_name,featured,tags,seo_title,seo_description,status,version,revision,published_revision,schedule_generation,scheduled_at,created_at,updated_at FROM news_items ORDER BY updated_at DESC, id ASC LIMIT ` + s.p(1)
	rows, err := queryer.QueryContext(ctx, query, limit)
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
	query := `SELECT id,schedule_generation,scheduled_at FROM news_items WHERE status='scheduled' AND scheduled_at IS NOT NULL AND scheduled_at <= ` + s.p(1) + ` ORDER BY scheduled_at,id LIMIT 100`
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
	rows, err := queryer.QueryContext(ctx, query, argument)
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
