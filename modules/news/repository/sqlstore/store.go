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

func (s Store) p(n int) string {
	if s.Dialect == database.SQLite {
		return "?"
	}
	return fmt.Sprintf("$%d", n)
}

func (s Store) CreateDraft(ctx context.Context, tx database.Executor, item news.Item) error {
	now := time.Now().UTC()
	query := `INSERT INTO news_items (id,locale,slug,title,body,status,version,revision,schedule_generation,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,'draft',1,1,0,$6,$7)`
	args := []any{item.ID, item.Locale, item.Slug, item.Title, item.Body, now, now}
	if s.Dialect == database.SQLite {
		query = `INSERT INTO news_items (id,locale,slug,title,body,status,version,revision,schedule_generation,created_at,updated_at) VALUES (?,?,?,?,?,'draft',1,1,0,?,?)`
		args[5], args[6] = now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (s Store) UpdateDraft(ctx context.Context, tx database.Executor, item news.Item, expectedVersion int) error {
	now := time.Now().UTC()
	query := `UPDATE news_items SET locale=$1,slug=$2,title=$3,body=$4,status='draft',revision=revision+1,version=version+1,updated_at=$5 WHERE id=$6 AND version=$7 AND status IN ('draft','retracted')`
	args := []any{item.Locale, item.Slug, item.Title, item.Body, now, item.ID, expectedVersion}
	if s.Dialect == database.SQLite {
		query = `UPDATE news_items SET locale=?,slug=?,title=?,body=?,status='draft',revision=revision+1,version=version+1,updated_at=? WHERE id=? AND version=? AND status IN ('draft','retracted')`
		args[4] = now.Format(time.RFC3339Nano)
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
	query := `SELECT id,locale,slug,title,body,status,version,revision,published_revision,schedule_generation,scheduled_at,created_at,updated_at FROM news_items WHERE id = ` + s.p(1)
	return s.scan(ctx, db, query, id)
}

func (s Store) GetPublishedByRoute(ctx context.Context, db database.Executor, locale, slug string) (news.Item, error) {
	query := `SELECT id,locale,slug,title,body,status,version,revision,published_revision,schedule_generation,scheduled_at,created_at,updated_at FROM news_items WHERE locale = ` + s.p(1) + ` AND slug = ` + s.p(2) + ` AND status='published' AND published_revision IS NOT NULL`
	return s.scan(ctx, db, query, locale, slug)
}

func (s Store) ListScheduled(ctx context.Context, db database.Executor, now time.Time) ([]news.Item, error) {
	query := `SELECT id,schedule_generation,scheduled_at FROM news_items WHERE status='scheduled' AND scheduled_at IS NOT NULL AND scheduled_at <= ` + s.p(1) + ` ORDER BY scheduled_at,id`
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
	var item news.Item
	var status string
	var published any
	var scheduled, created, updated any
	err := db.QueryRowContext(ctx, query, args...).Scan(&item.ID, &item.Locale, &item.Slug, &item.Title, &item.Body, &status, &item.Version, &item.Revision, &published, &item.ScheduleGeneration, &scheduled, &created, &updated)
	if err != nil {
		return news.Item{}, err
	}
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
	return item, nil
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
