package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func (s Store) RecordFormView(ctx context.Context, tenantID, formID, visitorKey string, day time.Time) (bool, error) {
	if err := s.validate(); err != nil {
		return false, err
	}
	if err := contextErr(ctx); err != nil {
		return false, err
	}
	if !safeDimension(tenantID, 128) || !safeDimension(formID, 128) || len(visitorKey) != 64 || !isHexDigest(visitorKey) || day.IsZero() {
		return false, errors.New("invalid form analytics view")
	}
	beginner, ok := s.Executor.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return false, errors.New("form analytics requires transaction support")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	dayValue := day.UTC().Format("2006-01-02")
	visitorQuery := `INSERT INTO analytics_form_visitors(day,tenant_id,form_id,visitor_key) VALUES (` + placeholder(s.Dialect, 1) + `,` + placeholder(s.Dialect, 2) + `,` + placeholder(s.Dialect, 3) + `,` + placeholder(s.Dialect, 4) + `) ON CONFLICT(day,tenant_id,form_id,visitor_key) DO NOTHING` // #nosec G202 -- only placeholders and fixed identifiers are concatenated; every value is bound
	if s.Dialect == database.MySQL {
		visitorQuery = `INSERT IGNORE INTO analytics_form_visitors(day,tenant_id,form_id,visitor_key) VALUES (?,?,?,?)`
	}
	inserted, err := tx.ExecContext(ctx, visitorQuery, dayValue, tenantID, formID, visitorKey)
	if err != nil {
		return false, err
	}
	count, err := inserted.RowsAffected()
	if err != nil {
		return false, err
	}
	if count == 0 {
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}
	viewQuery := `INSERT INTO analytics_form_daily(day,tenant_id,form_id,view_count,submission_count) VALUES (` + placeholder(s.Dialect, 1) + `,` + placeholder(s.Dialect, 2) + `,` + placeholder(s.Dialect, 3) + `,1,0) ON CONFLICT(day,tenant_id,form_id) DO UPDATE SET view_count=analytics_form_daily.view_count+1` // #nosec G202 -- only placeholders and fixed identifiers are concatenated; every value is bound
	if s.Dialect == database.SQLite {
		viewQuery = `INSERT INTO analytics_form_daily(day,tenant_id,form_id,view_count,submission_count) VALUES (?,?,?,1,0) ON CONFLICT(day,tenant_id,form_id) DO UPDATE SET view_count=view_count+1`
	} else if s.Dialect == database.MySQL {
		viewQuery = `INSERT INTO analytics_form_daily(day,tenant_id,form_id,view_count,submission_count) VALUES (?,?,?,1,0) ON DUPLICATE KEY UPDATE view_count=view_count+1`
	}
	if _, err := tx.ExecContext(ctx, viewQuery, dayValue, tenantID, formID); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (s Store) RecordFormSubmission(ctx context.Context, tx database.Executor, tenantID, formID string, day time.Time) error {
	if err := s.validate(); err != nil {
		return err
	}
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	if !safeDimension(tenantID, 128) || !safeDimension(formID, 128) || day.IsZero() {
		return errors.New("invalid form analytics submission")
	}
	query := `INSERT INTO analytics_form_daily(day,tenant_id,form_id,view_count,submission_count) VALUES (` + placeholder(s.Dialect, 1) + `,` + placeholder(s.Dialect, 2) + `,` + placeholder(s.Dialect, 3) + `,0,1) ON CONFLICT(day,tenant_id,form_id) DO UPDATE SET submission_count=analytics_form_daily.submission_count+1`
	if s.Dialect == database.SQLite {
		query = `INSERT INTO analytics_form_daily(day,tenant_id,form_id,view_count,submission_count) VALUES (?,?,?,0,1) ON CONFLICT(day,tenant_id,form_id) DO UPDATE SET submission_count=submission_count+1`
	} else if s.Dialect == database.MySQL {
		query = `INSERT INTO analytics_form_daily(day,tenant_id,form_id,view_count,submission_count) VALUES (?,?,?,0,1) ON DUPLICATE KEY UPDATE submission_count=submission_count+1`
	}
	_, err := tx.ExecContext(ctx, query, day.UTC().Format("2006-01-02"), tenantID, formID)
	return err
}

func (s Store) ListFormDaily(ctx context.Context, tenantID, formID, fromDay, toDay string) ([]analytics.FormDailyMetric, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if !safeDimension(tenantID, 128) || (formID != "" && !safeDimension(formID, 128)) {
		return nil, errors.New("invalid form analytics query")
	}
	if _, err := time.Parse("2006-01-02", fromDay); err != nil {
		return nil, errors.New("invalid form analytics date range")
	}
	if _, err := time.Parse("2006-01-02", toDay); err != nil || fromDay > toDay {
		return nil, errors.New("invalid form analytics date range")
	}
	queryer, ok := s.Executor.(database.QueryExecutor)
	if !ok {
		return nil, errors.New("form analytics requires query executor")
	}
	query := `SELECT day,tenant_id,form_id,view_count,submission_count FROM analytics_form_daily WHERE tenant_id=` + placeholder(s.Dialect, 1) + ` AND day BETWEEN ` + placeholder(s.Dialect, 2) + ` AND ` + placeholder(s.Dialect, 3)
	args := []any{tenantID, fromDay, toDay}
	if formID != "" {
		query += ` AND form_id=` + placeholder(s.Dialect, 4)
		args = append(args, formID)
	}
	query += ` ORDER BY day,form_id LIMIT 1000`
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]analytics.FormDailyMetric, 0)
	for rows.Next() {
		var item analytics.FormDailyMetric
		if err := rows.Scan(&item.Day, &item.TenantID, &item.FormID, &item.Views, &item.Submissions); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func isHexDigest(value string) bool {
	for _, char := range value {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}
