package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type Store struct{ Dialect database.Dialect }
type Notification struct {
	ID, Title, Message, TargetPath string
	CreatedAt                      time.Time
	Read                           bool
}

func (s Store) WeeklyDigestStats(ctx context.Context, db database.QueryExecutor, start, end time.Time, analyticsEnabled bool) (adminstudio.WeeklyDigestStats, error) {
	if err := database.RequireExecutor(db); err != nil {
		return adminstudio.WeeklyDigestStats{}, err
	}
	tenantID := ""
	if scope, err := tenant.FromContext(ctx); err == nil {
		tenantID = scope.TenantID
	}
	stats := adminstudio.WeeklyDigestStats{}
	startValue, endValue := start.UTC().Format(time.RFC3339Nano), end.UTC().Format(time.RFC3339Nano)
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM form_inbox WHERE tenant_id="+s.p(1)+" AND created_at>="+s.p(2)+" AND created_at<"+s.p(3), tenantID, startValue, endValue).Scan(&stats.NewLeads); err != nil {
		return stats, err
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM form_inbox WHERE tenant_id="+s.p(1)+" AND lead_status IN ('new','contacted')", tenantID).Scan(&stats.UnresolvedLeads); err != nil {
		return stats, err
	}
	if analyticsEnabled {
		rows, err := db.QueryContext(ctx, "SELECT dimension,SUM(event_count) FROM analytics_daily_stats WHERE tenant_id="+s.p(1)+" AND metric='referrer' AND day>="+s.p(2)+" AND day<"+s.p(3)+" GROUP BY dimension ORDER BY SUM(event_count) DESC,dimension LIMIT 5", tenantID, start.Format("2006-01-02"), end.Format("2006-01-02"))
		if err != nil {
			return stats, err
		}
		defer rows.Close()
		for rows.Next() {
			var item adminstudio.ReferrerCount
			if err := rows.Scan(&item.Host, &item.Count); err != nil {
				return stats, err
			}
			stats.TopReferrers = append(stats.TopReferrers, item)
		}
		if err := rows.Err(); err != nil {
			return stats, err
		}
	}
	return stats, nil
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

func (s Store) SetDigestPreference(ctx context.Context, db database.Executor, userID, timezone string, enabled bool) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if userID == "" {
		return ErrInvalid
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return ErrInvalid
	}
	tenantID := ""
	if scope, err := tenant.FromContext(ctx); err == nil {
		tenantID = scope.TenantID
	}
	query := "INSERT INTO admin_digest_preferences (tenant_id,user_id,enabled,timezone) VALUES (" + s.p(1) + "," + s.p(2) + "," + s.p(3) + "," + s.p(4) + ") ON CONFLICT (tenant_id,user_id) DO UPDATE SET enabled=excluded.enabled,timezone=excluded.timezone,updated_at=" + s.p(5)
	_, err := db.ExecContext(ctx, query, tenantID, userID, enabled, timezone, time.Now().UTC())
	return err
}

func (s Store) DigestPreferences(ctx context.Context, db database.QueryExecutor) ([]adminstudio.DigestPreference, error) {
	return s.DigestPreferencesAfter(ctx, db, "", 500)
}

func (s Store) DigestPreferencesAfter(ctx context.Context, db database.QueryExecutor, after string, limit int) ([]adminstudio.DigestPreference, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 500 { return nil, ErrInvalid }
	query := "SELECT user_id,timezone,last_sent_week FROM admin_digest_preferences WHERE enabled="
	if s.Dialect == database.Postgres {
		query += "TRUE"
	} else {
		query += "1"
	}
	args := make([]any,0,3)
	if scope, err := tenant.FromContext(ctx); err == nil {
		args=append(args,scope.TenantID)
		query += " AND tenant_id=" + s.p(len(args))
	}
	if _,err:=tenant.FromContext(ctx);err!=nil { query += " AND tenant_id=''" }
	args=append(args,after)
	query += " AND user_id>"+s.p(len(args))+" ORDER BY user_id LIMIT "+s.p(len(args)+1)
	args=append(args,limit)
	rows,err:=db.QueryContext(ctx,query,args...);if err!=nil{return nil,err};defer rows.Close()
	items:=make([]adminstudio.DigestPreference,0,limit)
	for rows.Next(){var item adminstudio.DigestPreference;if err:=rows.Scan(&item.UserID,&item.Timezone,&item.LastSentWeek);err!=nil{return nil,err};if _,err:=time.LoadLocation(item.Timezone);err==nil{items=append(items,item)}}
	return items,rows.Err()
}

func (s Store) DigestPreferenceFor(ctx context.Context, db database.Executor, userID string) (bool, string, error) {
	if err := database.RequireExecutor(db); err != nil {
		return false, "UTC", err
	}
	tenantID := ""
	if scope, err := tenant.FromContext(ctx); err == nil {
		tenantID = scope.TenantID
	}
	var enabled bool
	var timezone string
	err := db.QueryRowContext(ctx, "SELECT enabled,timezone FROM admin_digest_preferences WHERE tenant_id="+s.p(1)+" AND user_id="+s.p(2), tenantID, userID).Scan(&enabled, &timezone)
	if errors.Is(err, sql.ErrNoRows) {
		return false, "UTC", nil
	}
	return enabled, timezone, err
}

// ClaimDigestWeek must run in the same transaction as mail-spool insertion.
// A repeated scheduler tick or a second process then cannot enqueue duplicates.
func (s Store) ClaimDigestWeek(ctx context.Context, tx database.Executor, userID, week string) (bool, error) {
	if err := database.RequireExecutor(tx); err != nil {
		return false, err
	}
	if userID == "" || len(week) != 10 {
		return false, ErrInvalid
	}
	tenantID := ""
	if scope, err := tenant.FromContext(ctx); err == nil {
		tenantID = scope.TenantID
	}
	result, err := tx.ExecContext(ctx, "UPDATE admin_digest_preferences SET last_sent_week="+s.p(1)+",updated_at="+s.p(2)+" WHERE tenant_id="+s.p(3)+" AND user_id="+s.p(4)+" AND enabled="+s.boolean(true)+" AND last_sent_week<>"+s.p(5), week, time.Now().UTC(), tenantID, userID, week)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (s Store) boolean(value bool) string {
	if s.Dialect == database.Postgres {
		if value {
			return "TRUE"
		}
		return "FALSE"
	}
	if value {
		return "1"
	}
	return "0"
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
