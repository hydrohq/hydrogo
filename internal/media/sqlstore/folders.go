package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/media"
)

func (s Store) CreateFolder(ctx context.Context, db database.Executor, folder media.Folder) error {
	if raw, ok := db.(*sql.DB); ok {
		return database.NewTenantTransactor(raw, s.Dialect, s.RequireTenantScope).Within(ctx, func(tx database.Executor) error { return s.CreateFolder(ctx, tx, folder) })
	}
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if err = database.RequireExecutor(db); err != nil {
		return err
	}
	if folder.ID == "" || strings.TrimSpace(folder.Name) == "" || len(folder.Name) > 128 {
		return media.ErrInvalidFolder
	}
	if folder.ParentID != "" {
		q, args := "SELECT 1 FROM media_folders WHERE id="+s.p(1), []any{folder.ParentID}
		if scoped {
			q += " AND tenant_id=" + s.p(2)
			args = append(args, scope.TenantID)
		}
		var found int
		if err := db.QueryRowContext(ctx, q, args...).Scan(&found); err != nil {
			return media.ErrInvalidFolder
		}
	}
	parent, tenantID := any(nil), ""
	if folder.ParentID != "" {
		parent = folder.ParentID
	}
	if scoped {
		tenantID = scope.TenantID
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = db.ExecContext(ctx, "INSERT INTO media_folders (id,tenant_id,name,parent_id,created_at,updated_at) VALUES ("+s.p(1)+","+s.p(2)+","+s.p(3)+","+s.p(4)+","+s.p(5)+","+s.p(6)+")", folder.ID, tenantID, strings.TrimSpace(folder.Name), parent, now, now)
	return err
}

func (s Store) ListFolders(ctx context.Context, db database.Executor) ([]media.Folder, error) {
	if raw, ok := db.(*sql.DB); ok {
		var result []media.Folder
		err := database.NewTenantTransactor(raw, s.Dialect, s.RequireTenantScope).WithinQuery(ctx, func(tx database.QueryExecutor) error { var e error; result, e = s.ListFolders(ctx, tx); return e })
		return result, err
	}
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return nil, err
	}
	q, args := "SELECT id,name,parent_id,created_at FROM media_folders", []any{}
	if scoped {
		q += " WHERE tenant_id=" + s.p(1)
		args = append(args, scope.TenantID)
	}
	q += " ORDER BY name,id"
	rows, err := queryContext(ctx, db, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []media.Folder
	for rows.Next() {
		var f media.Folder
		var parent sql.NullString
		var created string
		if err := rows.Scan(&f.ID, &f.Name, &parent, &created); err != nil {
			return nil, err
		}
		f.ParentID = parent.String
		f.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			f.CreatedAt, err = time.Parse("2006-01-02 15:04:05.999999", created)
		}
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s Store) SetMetadata(ctx context.Context, db database.Executor, id string, metadata media.AssetMetadata) error {
	if raw, ok := db.(*sql.DB); ok {
		return database.NewTenantTransactor(raw, s.Dialect, s.RequireTenantScope).Within(ctx, func(tx database.Executor) error { return s.SetMetadata(ctx, tx, id, metadata) })
	}
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if err = database.RequireExecutor(db); err != nil {
		return err
	}
	if metadata.Focal.X < 0 || metadata.Focal.X > 1 || metadata.Focal.Y < 0 || metadata.Focal.Y > 1 || len(metadata.Tags) > 32 {
		return media.ErrInvalidFolder
	}
	for _, tag := range metadata.Tags {
		if strings.TrimSpace(tag) == "" || len(tag) > 64 {
			return media.ErrInvalidFolder
		}
	}
	if metadata.FolderID != "" {
		q, args := "SELECT 1 FROM media_folders WHERE id="+s.p(1), []any{metadata.FolderID}
		if scoped {
			q += " AND tenant_id=" + s.p(2)
			args = append(args, scope.TenantID)
		}
		var found int
		if err := db.QueryRowContext(ctx, q, args...).Scan(&found); err != nil {
			return media.ErrInvalidFolder
		}
	}
	tags, err := json.Marshal(metadata.Tags)
	if err != nil {
		return err
	}
	var folder any
	if metadata.FolderID != "" {
		folder = metadata.FolderID
	}
	q := "UPDATE media_assets SET focal_x=" + s.p(1) + ",focal_y=" + s.p(2) + ",folder_id=" + s.p(3) + ",tags_json=" + s.p(4) + ",updated_at=" + s.p(5) + " WHERE id=" + s.p(6) + " AND status IN ('pending','ready')"
	args := []any{metadata.Focal.X, metadata.Focal.Y, folder, string(tags), time.Now().UTC().Format(time.RFC3339Nano), id}
	if scoped {
		q += " AND tenant_id=" + s.p(7)
		args = append(args, scope.TenantID)
	}
	res, err := db.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n != 1 {
		return fmt.Errorf("%w: media asset unavailable", media.ErrAssetState)
	}
	return err
}
