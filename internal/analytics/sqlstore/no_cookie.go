package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// RecordNoCookieVisit persists the temporary daily key and its daily
// aggregates atomically. The retention worker deletes the key after the day
// closes; the key is never included in aggregate report queries.
func (s Store) RecordNoCookieVisit(ctx context.Context, visit analytics.NoCookieVisit) (bool, error) {
	if err := s.validate(); err != nil {
		return false, err
	}
	if err := contextErr(ctx); err != nil {
		return false, err
	}
	if _, err := time.Parse("2006-01-02", visit.Day); err != nil || len(visit.TenantID) > 128 || !safeDimension(visit.Page, 1024) || !safeDimension(visit.Referrer, 253) || !safeDimension(visit.Device, 16) || !safeDimension(visit.Browser, 32) || !safeDimension(visit.OS, 32) {
		return false, errors.New("invalid no-cookie analytics aggregate")
	}
	if !safeDimension(visit.VisitorKey, 64) {
		return false, errors.New("invalid daily visitor key")
	}
	beginner, ok := s.Executor.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return false, errors.New("daily visitor aggregation requires transaction support")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	visitorQuery := `INSERT INTO analytics_daily_visitors(day,tenant_id,visitor_key) VALUES (` + placeholder(s.Dialect, 1) + `,` + placeholder(s.Dialect, 2) + `,` + placeholder(s.Dialect, 3) + `) ON CONFLICT(day,tenant_id,visitor_key) DO NOTHING`
	if s.Dialect == database.MySQL {
		visitorQuery = `INSERT IGNORE INTO analytics_daily_visitors(day,tenant_id,visitor_key) VALUES (?,?,?)`
	}
	inserted, err := tx.ExecContext(ctx, visitorQuery, visit.Day, visit.TenantID, visit.VisitorKey)
	if err != nil {
		return false, err
	}
	insertedCount, err := inserted.RowsAffected()
	if err != nil {
		return false, err
	}
	unique := insertedCount == 1
	rows := [][2]string{{"page", visit.Page}, {"referrer", visit.Referrer}, {"device", visit.Device}, {"browser", visit.Browser}, {"os", visit.OS}}
	if unique {
		rows = append(rows, [2]string{"unique_visitors", ""})
	}
	values := make([]string, 0, len(rows))
	args := make([]any, 0, len(rows)*4)
	for _, row := range rows {
		start := len(args) + 1
		values = append(values, "("+placeholder(s.Dialect, start)+","+placeholder(s.Dialect, start+1)+","+placeholder(s.Dialect, start+2)+","+placeholder(s.Dialect, start+3)+",1)")
		args = append(args, visit.Day, visit.TenantID, row[0], row[1])
	}
	query := `INSERT INTO analytics_daily_stats(day,tenant_id,metric,dimension,event_count) VALUES ` + strings.Join(values, ",") + ` ON CONFLICT(day,tenant_id,metric,dimension) DO UPDATE SET event_count=analytics_daily_stats.event_count+EXCLUDED.event_count`
	if s.Dialect == database.SQLite {
		query = `INSERT INTO analytics_daily_stats(day,tenant_id,metric,dimension,event_count) VALUES ` + strings.Join(values, ",") + ` ON CONFLICT(day,tenant_id,metric,dimension) DO UPDATE SET event_count=event_count+excluded.event_count`
	} else if s.Dialect == database.MySQL {
		query = `INSERT INTO analytics_daily_stats(day,tenant_id,metric,dimension,event_count) VALUES ` + strings.Join(values, ",") + ` ON DUPLICATE KEY UPDATE event_count=event_count+VALUES(event_count)`
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return unique, nil
}

func safeDimension(value string, max int) bool {
	return value != "" && len(value) <= max && !strings.ContainsAny(value, "\x00\r\n")
}

func (s Store) PurgeNoCookieBefore(ctx context.Context, day, visitorDay string) error {
	if err := s.validate(); err != nil {
		return err
	}
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return errors.New("invalid analytics retention date")
	}
	if _, err := time.Parse("2006-01-02", visitorDay); err != nil {
		return errors.New("invalid analytics visitor retention date")
	}
	beginner, ok := s.Executor.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return errors.New("analytics retention requires transaction support")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM analytics_daily_visitors WHERE day < "+placeholder(s.Dialect, 1), visitorDay); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM analytics_daily_stats WHERE day < "+placeholder(s.Dialect, 1), day); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM analytics_form_visitors WHERE day < "+placeholder(s.Dialect, 1), visitorDay); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM analytics_form_daily WHERE day < "+placeholder(s.Dialect, 1), day); err != nil {
		return err
	}
	return tx.Commit()
}
