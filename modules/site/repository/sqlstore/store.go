package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type Store struct{ Dialect database.Dialect }

func (s Store) p(n int) string {
	if s.Dialect == database.SQLite {
		return "?"
	}
	return "$" + strconv.Itoa(n)
}

func (s Store) UpsertSetting(ctx context.Context, tx database.Executor, value site.Setting) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := value.UpdatedAt.UTC().Format(time.RFC3339Nano)
	if value.UpdatedAt.IsZero() {
		now = time.Now().UTC().Format(time.RFC3339Nano)
	}
	query := `INSERT INTO site_settings (key, value, is_public, updated_at) VALUES ($1,$2,$3,$4) ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value,is_public=EXCLUDED.is_public,updated_at=EXCLUDED.updated_at`
	args := []any{value.Key, value.Value, value.Public, now}
	if s.Dialect == database.SQLite {
		query = `INSERT INTO site_settings (key, value, is_public, updated_at) VALUES (?,?,?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value,is_public=excluded.is_public,updated_at=excluded.updated_at`
		args[2] = boolInt(value.Public)
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (s Store) ListPublicSettings(ctx context.Context, db database.Executor) ([]site.Setting, error) {
	query := `SELECT key,value,is_public,updated_at FROM site_settings WHERE is_public = TRUE ORDER BY key LIMIT 1000`
	if s.Dialect == database.SQLite {
		query = `SELECT key,value,is_public,updated_at FROM site_settings WHERE is_public = 1 ORDER BY key LIMIT 1000`
	}
	rows, err := queryContext(ctx, db, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]site.Setting, 0)
	for rows.Next() {
		var value site.Setting
		var public any
		var updated any
		if err := rows.Scan(&value.Key, &value.Value, &public, &updated); err != nil {
			return nil, err
		}
		value.Public = parseBool(public)
		value.UpdatedAt = parseTime(updated)
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s Store) ReplaceNavigation(ctx context.Context, tx database.Executor, locale string, items []site.NavigationItem) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	deleteQuery := `DELETE FROM navigation_items WHERE locale = $1`
	if s.Dialect == database.SQLite {
		deleteQuery = `DELETE FROM navigation_items WHERE locale = ?`
	}
	if _, err := tx.ExecContext(ctx, deleteQuery, locale); err != nil {
		return err
	}
	for _, item := range items {
		query := `INSERT INTO navigation_items (id,locale,label,href,position,is_visible,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`
		args := []any{item.ID, locale, item.Label, item.Href, item.Position, item.Visible, item.UpdatedAt.UTC().Format(time.RFC3339Nano)}
		if item.UpdatedAt.IsZero() {
			args[6] = time.Now().UTC().Format(time.RFC3339Nano)
		}
		if s.Dialect == database.SQLite {
			query = `INSERT INTO navigation_items (id,locale,label,href,position,is_visible,updated_at) VALUES (?,?,?,?,?,?,?)`
			args[5] = boolInt(item.Visible)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) ListNavigation(ctx context.Context, db database.Executor, locale string) ([]site.NavigationItem, error) {
	query := `SELECT id,locale,label,href,position,is_visible,updated_at FROM navigation_items WHERE locale = $1 AND is_visible = TRUE ORDER BY position,id LIMIT 500`
	if s.Dialect == database.SQLite {
		query = `SELECT id,locale,label,href,position,is_visible,updated_at FROM navigation_items WHERE locale = ? AND is_visible = 1 ORDER BY position,id LIMIT 500`
	}
	rows, err := queryContext(ctx, db, query, locale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]site.NavigationItem, 0)
	for rows.Next() {
		var item site.NavigationItem
		var visible any
		var updated any
		if err := rows.Scan(&item.ID, &item.Locale, &item.Label, &item.Href, &item.Position, &visible, &updated); err != nil {
			return nil, err
		}
		item.Visible = parseBool(visible)
		item.UpdatedAt = parseTime(updated)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s Store) PutRedirect(ctx context.Context, tx database.Executor, value site.Redirect) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := value.CreatedAt.UTC().Format(time.RFC3339Nano)
	if value.CreatedAt.IsZero() {
		now = time.Now().UTC().Format(time.RFC3339Nano)
	}
	query := `INSERT INTO page_redirects (source_path,target_path,status,created_at) VALUES ($1,$2,$3,$4) ON CONFLICT (source_path) DO UPDATE SET target_path=EXCLUDED.target_path,status=EXCLUDED.status`
	if s.Dialect == database.SQLite {
		query = `INSERT INTO page_redirects (source_path,target_path,status,created_at) VALUES (?,?,?,?) ON CONFLICT(source_path) DO UPDATE SET target_path=excluded.target_path,status=excluded.status`
	}
	_, err := tx.ExecContext(ctx, query, value.SourcePath, value.TargetPath, value.Status, now)
	return err
}

func (s Store) GetRedirect(ctx context.Context, db database.Executor, source string) (site.Redirect, error) {
	if err := database.RequireExecutor(db); err != nil {
		return site.Redirect{}, err
	}
	query := `SELECT source_path,target_path,status,created_at FROM page_redirects WHERE source_path = $1`
	if s.Dialect == database.SQLite {
		query = `SELECT source_path,target_path,status,created_at FROM page_redirects WHERE source_path = ?`
	}
	var value site.Redirect
	var created any
	err := db.QueryRowContext(ctx, query, source).Scan(&value.SourcePath, &value.TargetPath, &value.Status, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return site.Redirect{}, sql.ErrNoRows
	}
	value.CreatedAt = parseTime(created)
	return value, err
}

func (s Store) CreateSubmission(ctx context.Context, tx database.Executor, value site.Submission) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	query := `INSERT INTO form_inbox (id,form_key,email,payload,source_ip_hash,status,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`
	if s.Dialect == database.SQLite {
		query = `INSERT INTO form_inbox (id,form_key,email,payload,source_ip_hash,status,created_at) VALUES (?,?,?,?,?,?,?)`
	}
	created := value.CreatedAt.UTC().Format(time.RFC3339Nano)
	if value.CreatedAt.IsZero() {
		created = time.Now().UTC().Format(time.RFC3339Nano)
	}
	status := value.Status
	if status == "" {
		status = "new"
	}
	_, err := tx.ExecContext(ctx, query, value.ID, value.FormKey, value.Email, value.Payload, value.SourceIPHash, status, created)
	return err
}

func (s Store) ListSubmissions(ctx context.Context, db database.Executor, formKey string, limit int) ([]site.Submission, error) {
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	query := `SELECT id,form_key,email,payload,source_ip_hash,status,created_at FROM form_inbox WHERE form_key = $1 ORDER BY created_at DESC LIMIT $2`
	if s.Dialect == database.SQLite {
		query = `SELECT id,form_key,email,payload,source_ip_hash,status,created_at FROM form_inbox WHERE form_key = ? ORDER BY created_at DESC LIMIT ?`
	}
	rows, err := queryContext(ctx, db, query, formKey, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]site.Submission, 0)
	for rows.Next() {
		var value site.Submission
		var created any
		if err := rows.Scan(&value.ID, &value.FormKey, &value.Email, &value.Payload, &value.SourceIPHash, &value.Status, &created); err != nil {
			return nil, err
		}
		value.CreatedAt = parseTime(created)
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s Store) UpdateSubmissionStatus(ctx context.Context, tx database.Executor, id, status string) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	query := `UPDATE form_inbox SET status = $1 WHERE id = $2`
	if s.Dialect == database.SQLite {
		query = `UPDATE form_inbox SET status = ? WHERE id = ?`
	}
	result, err := tx.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (s Store) PurgeSubmissionsBefore(ctx context.Context, tx database.Executor, before time.Time) (int64, error) {
	if err := database.RequireExecutor(tx); err != nil {
		return 0, err
	}
	query := `DELETE FROM form_inbox WHERE created_at < $1`
	arg := any(before)
	if s.Dialect == database.SQLite {
		query = `DELETE FROM form_inbox WHERE created_at < ?`
		arg = before.Format(time.RFC3339Nano)
	}
	result, err := tx.ExecContext(ctx, query, arg)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func parseBool(value any) bool {
	switch value := value.(type) {
	case bool:
		return value
	case int64:
		return value != 0
	case int:
		return value != 0
	case []byte:
		return string(value) == "1" || strings.EqualFold(string(value), "true")
	case string:
		return value == "1" || strings.EqualFold(value, "true")
	default:
		return false
	}
}

type queryExecutor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func queryContext(ctx context.Context, db database.Executor, query string, args ...any) (*sql.Rows, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	queryer, ok := db.(queryExecutor)
	if !ok {
		return nil, fmt.Errorf("database executor does not support row queries")
	}
	return queryer.QueryContext(ctx, query, args...)
}

func parseTime(value any) time.Time {
	switch value := value.(type) {
	case time.Time:
		return value.UTC()
	case string:
		parsed, _ := time.Parse(time.RFC3339Nano, value)
		return parsed.UTC()
	case []byte:
		parsed, _ := time.Parse(time.RFC3339Nano, string(value))
		return parsed.UTC()
	default:
		return time.Time{}
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
