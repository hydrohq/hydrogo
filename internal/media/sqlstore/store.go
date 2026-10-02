package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
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
	if rawDB, ok := db.(*sql.DB); ok {
		return database.NewTenantTransactor(rawDB, s.Dialect, s.RequireTenantScope).Within(ctx, func(tx database.Executor) error {
			return s.CreateProcessing(ctx, tx, variant)
		})
	}
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	now := s.dbNow()
	if scoped {
		if s.Dialect == database.MySQL {
			query := `INSERT INTO media_variants (id, media_id, kind, storage_key, mime_type, size_bytes, width, height, duration_ms, status, error_message, created_at, updated_at, tenant_id) VALUES (?,?,?,?,?,0,0,0,0,'processing','',?,?,?) ON DUPLICATE KEY UPDATE status=IF(tenant_id=VALUES(tenant_id),'processing',status), error_message=IF(tenant_id=VALUES(tenant_id),'',error_message), updated_at=IF(tenant_id=VALUES(tenant_id),VALUES(updated_at),updated_at)`
			_, err = db.ExecContext(ctx, query, variant.ID, variant.MediaID, variant.Kind, variant.ID, variant.MimeType, now, now, scope.TenantID)
			return err
		}
		query := `INSERT INTO media_variants (id, media_id, kind, storage_key, mime_type, size_bytes, width, height, duration_ms, status, error_message, created_at, updated_at, tenant_id) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,0,0,0,0,'processing','',` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `) ON CONFLICT(media_id, kind) DO UPDATE SET status='processing', error_message='', updated_at=` + s.p(9) + ` WHERE media_variants.tenant_id=` + s.p(10)
		_, err = db.ExecContext(ctx, query, variant.ID, variant.MediaID, variant.Kind, variant.ID, variant.MimeType, now, now, scope.TenantID, now, scope.TenantID)
		return err
	}
	if s.Dialect == database.MySQL {
		query := `INSERT INTO media_variants (id, media_id, kind, storage_key, mime_type, size_bytes, width, height, duration_ms, status, error_message, created_at, updated_at) VALUES (?,?,?,?,?,0,0,0,0,'processing','',?,?) ON DUPLICATE KEY UPDATE status='processing', error_message='', updated_at=VALUES(updated_at)`
		_, err = db.ExecContext(ctx, query, variant.ID, variant.MediaID, variant.Kind, variant.ID, variant.MimeType, now, now)
		return err
	}
	query := `INSERT INTO media_variants (id, media_id, kind, storage_key, mime_type, size_bytes, width, height, duration_ms, status, error_message, created_at, updated_at) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,0,0,0,0,'processing','',` + s.p(6) + `,` + s.p(7) + `) ON CONFLICT(media_id, kind) DO UPDATE SET status='processing', error_message='', updated_at=` + s.p(8)
	_, err = db.ExecContext(ctx, query, variant.ID, variant.MediaID, variant.Kind, variant.ID, variant.MimeType, now, now, now)
	return err
}

func (s Store) MarkResult(ctx context.Context, db database.Executor, id, status string, size int64, detail string) error {
	if rawDB, ok := db.(*sql.DB); ok {
		return database.NewTenantTransactor(rawDB, s.Dialect, s.RequireTenantScope).Within(ctx, func(tx database.Executor) error {
			return s.MarkResult(ctx, tx, id, status, size, detail)
		})
	}
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	now := s.dbNow()
	if status == "ready" {
		storageKey, keyErr := media.StorageKeyForContext(ctx, detail)
		if keyErr != nil {
			return keyErr
		}
		query := "UPDATE media_variants SET status='ready', storage_key=" + s.p(1) + ", size_bytes=" + s.p(2) + ", error_message='', updated_at=" + s.p(3) + " WHERE id=" + s.p(4)
		args := []any{storageKey, size, now, id}
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
	if rawDB, ok := db.(*sql.DB); ok {
		var result media.Variant
		err := database.NewTenantTransactor(rawDB, s.Dialect, s.RequireTenantScope).Within(ctx, func(tx database.Executor) error {
			var err error
			result, err = s.GetReady(ctx, tx, mediaID, kind)
			return err
		})
		return result, err
	}
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
	if rawDB, ok := db.(*sql.DB); ok {
		var result media.Asset
		err := database.NewTenantTransactor(rawDB, s.Dialect, s.RequireTenantScope).Within(ctx, func(tx database.Executor) error {
			var err error
			result, err = s.Get(ctx, tx, id)
			return err
		})
		return result, err
	}
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return media.Asset{}, err
	}
	if err := database.RequireExecutor(db); err != nil {
		return media.Asset{}, err
	}
	var asset media.Asset
	var visibility, status string
	query := "SELECT id, storage_key, original_filename, mime_type, size_bytes, width, height, visibility, status, blur_hash, focal_x, focal_y, dominant_colors, folder_id, tags_json FROM media_assets WHERE id = " + s.p(1) + " AND deleted_at IS NULL"
	args := []any{id}
	if scoped {
		query += " AND tenant_id=" + s.p(2)
		args = append(args, scope.TenantID)
	}
	var blurHash, colors, tags sql.NullString
	var folder sql.NullString
	err = db.QueryRowContext(ctx, query, args...).Scan(&asset.ID, &asset.StorageKey, &asset.OriginalFilename, &asset.MimeType, &asset.SizeBytes, &asset.Width, &asset.Height, &visibility, &status, &blurHash, &asset.FocalX, &asset.FocalY, &colors, &folder, &tags)
	if err != nil {
		return media.Asset{}, err
	}
	asset.Visibility = visibility
	asset.Status = status
	asset.BlurHash = blurHash.String
	_ = json.Unmarshal([]byte(colors.String), &asset.DominantColors)
	_ = json.Unmarshal([]byte(tags.String), &asset.Tags)
	asset.FolderID = folder.String
	return asset, nil
}

func (s Store) List(ctx context.Context, db database.Executor, limit int) ([]media.Asset, error) {
	if rawDB, ok := db.(*sql.DB); ok {
		var result []media.Asset
		err := database.NewTenantTransactor(rawDB, s.Dialect, s.RequireTenantScope).WithinQuery(ctx, func(tx database.QueryExecutor) error {
			var err error
			result, err = s.List(ctx, tx, limit)
			return err
		})
		return result, err
	}
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
	query := "SELECT id, storage_key, original_filename, mime_type, size_bytes, width, height, visibility, status, blur_hash, focal_x, focal_y, dominant_colors, folder_id, tags_json FROM media_assets WHERE deleted_at IS NULL"
	args := []any{}
	if scoped {
		query += " AND tenant_id=" + s.p(1)
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
		var blurHash, colors, tags sql.NullString
		var folder sql.NullString
		if err := rows.Scan(&asset.ID, &asset.StorageKey, &asset.OriginalFilename, &asset.MimeType, &asset.SizeBytes, &asset.Width, &asset.Height, &asset.Visibility, &asset.Status, &blurHash, &asset.FocalX, &asset.FocalY, &colors, &folder, &tags); err != nil {
			return nil, err
		}
		asset.BlurHash = blurHash.String
		_ = json.Unmarshal([]byte(colors.String), &asset.DominantColors)
		_ = json.Unmarshal([]byte(tags.String), &asset.Tags)
		asset.FolderID = folder.String
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
	now := s.dbNow()
	colors, err := json.Marshal(asset.DominantColors)
	if err != nil {
		return err
	}
	tags, err := json.Marshal(asset.Tags)
	if err != nil {
		return err
	}
	var folder any
	if asset.FolderID != "" {
		folder = asset.FolderID
	}
	query := `INSERT INTO media_assets
		(id, storage_key, original_filename, mime_type, size_bytes, width, height, visibility, status, created_at, updated_at, blur_hash, focal_x, focal_y, dominant_colors, folder_id, tags_json)
		VALUES (` + s.p(1) + `, ` + s.p(2) + `, ` + s.p(3) + `, ` + s.p(4) + `, ` + s.p(5) + `, ` + s.p(6) + `, ` + s.p(7) + `, 'private', 'pending', ` + s.p(8) + `, ` + s.p(9) + `, ` + s.p(10) + `, ` + s.p(11) + `, ` + s.p(12) + `, ` + s.p(13) + `, ` + s.p(14) + `, ` + s.p(15) + `)`
	args := []any{asset.ID, asset.StorageKey, asset.OriginalFilename, asset.MimeType, asset.SizeBytes, asset.Width, asset.Height, now, now, asset.BlurHash, asset.FocalX, asset.FocalY, string(colors), folder, string(tags)}
	if scoped {
		query = `INSERT INTO media_assets
			(id, storage_key, original_filename, mime_type, size_bytes, width, height, visibility, status, created_at, updated_at, tenant_id, blur_hash, focal_x, focal_y, dominant_colors, folder_id, tags_json)
			VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `, 'private', 'pending', ` + s.p(8) + `,` + s.p(9) + `,` + s.p(10) + `,` + s.p(11) + `,` + s.p(12) + `,` + s.p(13) + `,` + s.p(14) + `,` + s.p(15) + `,` + s.p(16) + `)`
		args = []any{asset.ID, asset.StorageKey, asset.OriginalFilename, asset.MimeType, asset.SizeBytes, asset.Width, asset.Height, now, now, scope.TenantID, asset.BlurHash, asset.FocalX, asset.FocalY, string(colors), folder, string(tags)}
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
	args := []any{visibility, s.dbNow(), id}
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

func (s Store) MarkReadyVerified(ctx context.Context, tx database.Executor, id, storageKey, visibility string) error {
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
	query := "UPDATE media_assets SET status = 'ready', visibility = " + s.p(1) + ", updated_at = " + s.p(2) + " WHERE id = " + s.p(3) + " AND storage_key = " + s.p(4) + " AND status = 'pending'"
	args := []any{visibility, s.dbNow(), id, storageKey}
	if scoped {
		query += " AND tenant_id=" + s.p(5)
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
	args := []any{visibility, s.dbNow(), id}
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

func (s Store) dbNow() any {
	return s.dbTimeValue(time.Now().UTC())
}

func (s Store) dbTimeValue(t time.Time) any {
	t = t.UTC()
	if s.Dialect == database.SQLite {
		return t.Format(time.RFC3339Nano)
	}
	return t
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
