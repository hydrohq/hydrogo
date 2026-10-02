package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func (s Store) ListGarbageCandidates(ctx context.Context, db database.Executor, before time.Time, limit int) ([]media.GarbageAsset, error) {
	if limit < 1 || limit > 200 {
		return nil, errors.New("invalid media cleanup batch size")
	}
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return nil, err
	}
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	query := `SELECT id, original_filename, size_bytes+COALESCE((SELECT SUM(size_bytes) FROM media_variants WHERE media_id=media_assets.id),0), created_at, status FROM media_assets WHERE deleted_at IS NULL AND ((status='ready' AND created_at<=` + s.p(1) + `) OR status='deleting')`
	args := []any{s.dbTimeValue(before)}
	if scoped {
		query = `SELECT id, original_filename, size_bytes+COALESCE((SELECT SUM(size_bytes) FROM media_variants WHERE media_id=media_assets.id AND tenant_id=media_assets.tenant_id),0), created_at, status FROM media_assets WHERE deleted_at IS NULL AND ((status='ready' AND created_at<=` + s.p(1) + `) OR status='deleting') AND tenant_id=` + s.p(2)
		args = append(args, scope.TenantID)
	}
	query += ` ORDER BY created_at ASC, id ASC LIMIT ` + s.p(len(args)+1)
	args = append(args, limit)
	rows, err := queryContext(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]media.GarbageAsset, 0, limit)
	for rows.Next() {
		var item media.GarbageAsset
		var created any
		if err := rows.Scan(&item.ID, &item.Filename, &item.SizeBytes, &created, &item.Status); err != nil {
			return nil, err
		}
		item.CreatedAt = parseMediaTime(created)
		if item.CreatedAt.IsZero() {
			return nil, fmt.Errorf("invalid media asset creation time")
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s Store) ClaimGarbageAsset(ctx context.Context, db database.Executor, id string, createdAt time.Time) (bool, error) {
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return false, err
	}
	query := `UPDATE media_assets SET status='deleting', updated_at=` + s.p(1) + ` WHERE id=` + s.p(2) + ` AND status='ready' AND created_at=` + s.p(3)
	args := []any{s.dbNow(), id, s.dbTimeValue(createdAt)}
	if scoped {
		query += ` AND tenant_id=` + s.p(4)
		args = append(args, scope.TenantID)
	}
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (s Store) GarbageObjectKeys(ctx context.Context, db database.Executor, id string) ([]string, error) {
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT storage_key FROM media_assets WHERE id=` + s.p(1) + ` AND status='deleting'`
	args := []any{id}
	if scoped {
		query += ` AND tenant_id=` + s.p(2)
		args = append(args, scope.TenantID)
	}
	var master string
	if err := db.QueryRowContext(ctx, query, args...).Scan(&master); err != nil {
		return nil, err
	}
	keys := []string{master}
	variantQuery := `SELECT storage_key FROM media_variants WHERE media_id=` + s.p(1)
	variantArgs := []any{id}
	if scoped {
		variantQuery += ` AND tenant_id=` + s.p(2)
		variantArgs = append(variantArgs, scope.TenantID)
	}
	rows, err := queryContext(ctx, db, variantQuery, variantArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func (s Store) RestoreGarbageAsset(ctx context.Context, db database.Executor, id string) error {
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return err
	}
	query := `UPDATE media_assets SET status='ready', updated_at=` + s.p(1) + ` WHERE id=` + s.p(2) + ` AND status='deleting'`
	args := []any{s.dbNow(), id}
	if scoped {
		query += ` AND tenant_id=` + s.p(3)
		args = append(args, scope.TenantID)
	}
	_, err = db.ExecContext(ctx, query, args...)
	return err
}

func (s Store) DeleteGarbageAsset(ctx context.Context, db database.Executor, id string) error {
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return err
	}
	query := `DELETE FROM media_assets WHERE id=` + s.p(1) + ` AND status='deleting'`
	args := []any{id}
	if scoped {
		query += ` AND tenant_id=` + s.p(2)
		args = append(args, scope.TenantID)
	}
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if changed, err := result.RowsAffected(); err == nil && changed == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type MediaReferences struct {
	Dialect            database.Dialect
	RequireTenantScope bool
	Pages              bool
	News               bool
	Site               bool
	Forms              bool
}

func (s MediaReferences) ReferencedMediaIDs(ctx context.Context, db database.Executor) (map[string]struct{}, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	scope, scopeErr := tenant.FromContext(ctx)
	scoped := scopeErr == nil
	if scopeErr != nil && s.RequireTenantScope {
		return nil, scopeErr
	}
	referenced := make(map[string]struct{})
	queryTexts := make([]string, 0, 5)
	if s.Pages {
		queryTexts = append(queryTexts, "SELECT body FROM pages", "SELECT body, blocks FROM page_revisions")
	}
	if s.News {
		queryTexts = append(queryTexts, "SELECT body, cover_media_id FROM news_items")
	}
	if s.Site {
		queryTexts = append(queryTexts, "SELECT value FROM site_settings")
	}
	if s.Forms {
		queryTexts = append(queryTexts, "SELECT payload FROM form_inbox")
	}
	var err error
	for _, query := range queryTexts {
		if scoped && s.RequireTenantScope {
			query += " WHERE tenant_id=" + s.placeholder(1)
		}
		var rows *sql.Rows
		if scoped && s.RequireTenantScope {
			rows, err = queryContext(ctx, db, query, scope.TenantID)
		} else {
			rows, err = queryContext(ctx, db, query)
		}
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			values := make([]sql.NullString, strings.Count(query, ",")+1)
			dest := make([]any, len(values))
			for i := range values {
				dest[i] = &values[i]
			}
			if err := rows.Scan(dest...); err != nil {
				rows.Close()
				return nil, err
			}
			for _, value := range values {
				for id := range media.ReferencedIDs(value.String) {
					referenced[id] = struct{}{}
				}
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		_ = rows.Close()
	}
	return referenced, nil
}

func (s MediaReferences) placeholder(index int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func parseMediaTime(value any) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v.UTC()
	case string:
		return parseMediaTimeString(v)
	case []byte:
		return parseMediaTimeString(string(v))
	default:
		return time.Time{}
	}
}

func parseMediaTimeString(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}
