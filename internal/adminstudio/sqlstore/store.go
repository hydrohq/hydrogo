package sqlstore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type Store struct{ Dialect database.Dialect }
type Notification struct {
	ID, Title, Message, TargetPath string
	CreatedAt                      time.Time
	Read                           bool
}

var ErrInvalid = errors.New("invalid notification request")

func (s Store) Add(ctx context.Context, db database.Executor, userID, title, message, target string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if userID == "" || len(title) > 160 || len(message) > 1000 || !safeTarget(target) {
		return ErrInvalid
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return err
	}
	id := hex.EncodeToString(idBytes)
	var tenantID any
	if scope, err := tenant.FromContext(ctx); err == nil {
		tenantID = scope.TenantID
	}
	_, err := db.ExecContext(ctx, "INSERT INTO admin_notifications (id,tenant_id,user_id,title,message,target_path) VALUES ("+s.p(1)+","+s.p(2)+","+s.p(3)+","+s.p(4)+","+s.p(5)+","+s.p(6)+")", id, tenantID, userID, title, message, target)
	return err
}

func (s Store) List(ctx context.Context, db database.QueryExecutor, userID string, limit int) ([]Notification, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	if userID == "" || limit < 1 || limit > 100 {
		return nil, ErrInvalid
	}
	query := "SELECT id,title,message,target_path,created_at,read_at FROM admin_notifications WHERE user_id=" + s.p(1) + " AND dismissed_at IS NULL"
	args := []any{userID}
	if scope, err := tenant.FromContext(ctx); err == nil {
		query += " AND tenant_id=" + s.p(2)
		args = append(args, scope.TenantID)
	} else {
		query += " AND tenant_id IS NULL"
	}
	query += " ORDER BY created_at DESC LIMIT " + s.p(len(args)+1)
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Notification, 0, limit)
	for rows.Next() {
		var item Notification
		var created any
		var read any
		if err := rows.Scan(&item.ID, &item.Title, &item.Message, &item.TargetPath, &created, &read); err != nil {
			return nil, err
		}
		switch value := created.(type) {
		case time.Time:
			item.CreatedAt = value
		case string:
			item.CreatedAt, _ = time.Parse(time.RFC3339Nano, value)
		case []byte:
			item.CreatedAt, _ = time.Parse(time.RFC3339Nano, string(value))
		}
		item.Read = read != nil
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s Store) MarkAllRead(ctx context.Context, db database.Executor, userID string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if userID == "" {
		return ErrInvalid
	}
	query := "UPDATE admin_notifications SET read_at=" + s.p(1) + " WHERE user_id=" + s.p(2) + " AND read_at IS NULL AND dismissed_at IS NULL"
	args := []any{time.Now().UTC(), userID}
	if scope, err := tenant.FromContext(ctx); err == nil {
		query += " AND tenant_id=" + s.p(3)
		args = append(args, scope.TenantID)
	} else {
		query += " AND tenant_id IS NULL"
	}
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func (s Store) Dismiss(ctx context.Context, db database.Executor, userID, id string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if userID == "" || id == "" {
		return ErrInvalid
	}
	query := "UPDATE admin_notifications SET dismissed_at=" + s.p(1) + " WHERE id=" + s.p(2) + " AND user_id=" + s.p(3)
	args := []any{time.Now().UTC(), id, userID}
	if scope, err := tenant.FromContext(ctx); err == nil {
		query += " AND tenant_id=" + s.p(4)
		args = append(args, scope.TenantID)
	} else {
		query += " AND tenant_id IS NULL"
	}
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func (s Store) UnreadCount(ctx context.Context, db database.Executor, userID string) (int, error) {
	if database.RequireExecutor(db) != nil || userID == "" {
		return 0, ErrInvalid
	}
	query := "SELECT COUNT(*) FROM admin_notifications WHERE user_id=" + s.p(1) + " AND read_at IS NULL AND dismissed_at IS NULL"
	args := []any{userID}
	if scope, err := tenant.FromContext(ctx); err == nil {
		query += " AND tenant_id=" + s.p(2)
		args = append(args, scope.TenantID)
	} else {
		query += " AND tenant_id IS NULL"
	}
	var count int
	err := db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}
func (s Store) p(n int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}
func safeTarget(value string) bool {
	return len(value) > 0 && value[0] == '/' && (len(value) == 1 || value[1] != '/')
}
