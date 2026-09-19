package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/jobs"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type Store struct {
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s Store) scope(ctx context.Context) (tenant.Scope, bool, error) {
	scope, err := tenant.FromContext(ctx)
	if err == nil {
		return scope, true, nil
	}
	if s.RequireTenantScope {
		return tenant.Scope{}, false, err
	}
	return tenant.Scope{}, false, nil
}

func (s Store) CreateProcessing(ctx context.Context, db database.Executor, variant media.Variant) error {
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if scoped {
		query := `INSERT INTO media_variants (id, media_id, kind, storage_key, mime_type, size_bytes, width, height, duration_ms, status, error_message, created_at, updated_at, tenant_id) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,0,0,0,0,'processing','',` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `) ON CONFLICT(media_id, kind) DO UPDATE SET status='processing', error_message='', updated_at=` + s.p(9) + ` WHERE media_variants.tenant_id=` + s.p(10)
		_, err = db.ExecContext(ctx, query, variant.ID, variant.MediaID, variant.Kind, variant.ID, variant.MimeType, now, now, scope.TenantID, now, scope.TenantID)
		return err
	}
	query := `INSERT INTO media_variants (id, media_id, kind, storage_key, mime_type, size_bytes, width, height, duration_ms, status, error_message, created_at, updated_at) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,0,0,0,0,'processing','',` + s.p(6) + `,` + s.p(7) + `) ON CONFLICT(media_id, kind) DO UPDATE SET status='processing', error_message='', updated_at=` + s.p(8)
	_, err = db.ExecContext(ctx, query, variant.ID, variant.MediaID, variant.Kind, variant.ID, variant.MimeType, now, now, now)
	return err
}

func (s Store) MarkResult(ctx context.Context, db database.Executor, id, status string, size int64, detail string) error {
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if status == "ready" {
		query := "UPDATE media_variants SET status='ready', storage_key=" + s.p(1) + ", size_bytes=" + s.p(2) + ", error_message='', updated_at=" + s.p(3) + " WHERE id=" + s.p(4)
		args := []any{detail, size, now, id}
		if scoped {
			query += " AND tenant_id=" + s.p(5)
			args = append(args, scope.TenantID)
		}
		_, err := db.ExecContext(ctx, query, args...)
		return err
	}
	query := "UPDATE media_variants SET status='failed', error_message=" + s.p(1) + ", updated_at=" + s.p(2) + " WHERE id=" + s.p(3)
	args := []any{jobs.BoundError(detail), now, id}
	if scoped {
		query += " AND tenant_id=" + s.p(4)
		args = append(args, scope.TenantID)
	}
	_, err = db.ExecContext(ctx, query, args...)
	return err
}

func (s Store) GetReady(ctx context.Context, db database.Executor, mediaID, kind string) (media.Variant, error) {
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return media.Variant{}, err
	}
	if err := database.RequireExecutor(db); err != nil {
		return media.Variant{}, err
	}
	var v media.Variant
	query := "SELECT id, media_id, kind, storage_key, mime_type, size_bytes, width, height, duration_ms, status, error_message FROM media_variants WHERE media_id=" + s.p(1) + " AND kind=" + s.p(2) + " AND status='ready'"
	args := []any{mediaID, kind}
	if scoped {
		query += " AND tenant_id=" + s.p(3)
		args = append(args, scope.TenantID)
	}
	err = db.QueryRowContext(ctx, query, args...).Scan(&v.ID, &v.MediaID, &v.Kind, &v.StorageKey, &v.MimeType, &v.SizeBytes, &v.Width, &v.Height, &v.DurationMs, &v.Status, &v.ErrorMessage)
	return v, err
}

func (s Store) Get(ctx context.Context, db database.Executor, id string) (media.Asset, error) {
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return media.Asset{}, err
	}
	if err := database.RequireExecutor(db); err != nil {
		return media.Asset{}, err
	}
	var asset media.Asset
	var visibility, status string
	query := "SELECT id, storage_key, original_filename, mime_type, size_bytes, width, height, visibility, status FROM media_assets WHERE id = " + s.p(1)
	args := []any{id}
	if scoped {
		query += " AND tenant_id=" + s.p(2)
		args = append(args, scope.TenantID)
	}
	err = db.QueryRowContext(ctx, query, args...).Scan(&asset.ID, &asset.StorageKey, &asset.OriginalFilename, &asset.MimeType, &asset.SizeBytes, &asset.Width, &asset.Height, &visibility, &status)
	if err != nil {
		return media.Asset{}, err
	}
	asset.Visibility = visibility
	asset.Status = status
	return asset, nil
}

func (s Store) List(ctx context.Context, db database.Executor, limit int) ([]media.Asset, error) {
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return nil, err
	}
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 200 {
		return nil, fmt.Errorf("media list limit must be between 1 and 200")
	}
	query := "SELECT id, storage_key, original_filename, mime_type, size_bytes, width, height, visibility, status FROM media_assets"
	args := []any{}
	if scoped {
		query += " WHERE tenant_id=" + s.p(1)
		args = append(args, scope.TenantID)
	}
	query += " ORDER BY updated_at DESC, id ASC LIMIT " + s.p(len(args)+1)
	args = append(args, limit)
	rows, err := queryContext(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	assets := make([]media.Asset, 0, limit)
	for rows.Next() {
		var asset media.Asset
		if err := rows.Scan(&asset.ID, &asset.StorageKey, &asset.OriginalFilename, &asset.MimeType, &asset.SizeBytes, &asset.Width, &asset.Height, &asset.Visibility, &asset.Status); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return assets, nil
}

func (s Store) CreatePending(ctx context.Context, tx database.Executor, asset media.Asset) error {
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	query := `INSERT INTO media_assets
        (id, storage_key, original_filename, mime_type, size_bytes, width, height, visibility, status, created_at, updated_at)
        VALUES (` + s.p(1) + `, ` + s.p(2) + `, ` + s.p(3) + `, ` + s.p(4) + `, ` + s.p(5) + `, ` + s.p(6) + `, ` + s.p(7) + `, 'private', 'pending', ` + s.p(8) + `, ` + s.p(9) + `)`
	args := []any{asset.ID, asset.StorageKey, asset.OriginalFilename, asset.MimeType, asset.SizeBytes, asset.Width, asset.Height, now, now}
	if scoped {
		query = `INSERT INTO media_assets
        (id, storage_key, original_filename, mime_type, size_bytes, width, height, visibility, status, created_at, updated_at, tenant_id)
        VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `, 'private', 'pending', ` + s.p(8) + `,` + s.p(9) + `,` + s.p(10) + `)`
		args = append(args, scope.TenantID)
	}
	_, err = tx.ExecContext(ctx, query, args...)
	return err
}

func (s Store) MarkReady(ctx context.Context, tx database.Executor, id, visibility string) error {
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	if visibility != media.VisibilityPrivate && visibility != media.VisibilityPublic {
		return media.ErrInvalidVisibility
	}
	query := "UPDATE media_assets SET status = 'ready', visibility = " + s.p(1) + ", updated_at = " + s.p(2) + " WHERE id = " + s.p(3) + " AND status = 'pending'"
	args := []any{visibility, time.Now().UTC().Format(time.RFC3339Nano), id}
	if scoped {
		query += " AND tenant_id=" + s.p(4)
		args = append(args, scope.TenantID)
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return media.ErrAssetState
	}
	return nil
}

func (s Store) SetVisibility(ctx context.Context, tx database.Executor, id, visibility string) error {
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	if visibility != media.VisibilityPrivate && visibility != media.VisibilityPublic {
		return media.ErrInvalidVisibility
	}
	query := "UPDATE media_assets SET visibility = " + s.p(1) + ", updated_at = " + s.p(2) + " WHERE id = " + s.p(3) + " AND status = 'ready'"
	args := []any{visibility, time.Now().UTC().Format(time.RFC3339Nano), id}
	if scoped {
		query += " AND tenant_id=" + s.p(4)
		args = append(args, scope.TenantID)
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return media.ErrAssetState
	}
	return nil
}

func (s Store) p(position int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
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
