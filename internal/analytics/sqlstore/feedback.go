package sqlstore

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hydrohq/hydrogo/internal/analytics"
	"github.com/hydrohq/hydrogo/internal/database"
)

// FeedbackTotals contains only site-wide aggregate counts; no visitor identity is stored.
type FeedbackTotals = analytics.FeedbackTotals

func (s Store) RecordFeedback(ctx context.Context, tenantID, contentType, contentID, reaction string, day time.Time) error {
	if err := s.validate(); err != nil {
		return err
	}
	if err := contextErr(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(tenantID) == "" || len(tenantID) > 128 || (contentType != "news" && contentType != "faq") || strings.TrimSpace(contentID) == "" || len(contentID) > 128 || (reaction != "up" && reaction != "down") {
		return errors.New("invalid feedback")
	}
	dayValue := day.UTC().Format("2006-01-02")
	var query string
	switch s.Dialect {
	case database.Postgres:
		query = `INSERT INTO analytics_feedback_daily (tenant_id,content_type,content_id,day,reaction,event_count) VALUES ($1,$2,$3,$4,$5,1) ON CONFLICT (tenant_id,content_type,content_id,day,reaction) DO UPDATE SET event_count=analytics_feedback_daily.event_count+1`
	case database.MySQL:
		query = `INSERT INTO analytics_feedback_daily (tenant_id,content_type,content_id,day,reaction,event_count) VALUES (?,?,?,?,?,1) ON DUPLICATE KEY UPDATE event_count=event_count+1`
	default:
		query = `INSERT INTO analytics_feedback_daily (tenant_id,content_type,content_id,day,reaction,event_count) VALUES (?,?,?,?,?,1) ON CONFLICT(tenant_id,content_type,content_id,day,reaction) DO UPDATE SET event_count=event_count+1`
	}
	return s.withTenant(ctx, tenantID, func(exec database.Executor) error {
		_, err := exec.ExecContext(ctx, query, tenantID, contentType, contentID, dayValue, reaction)
		return err
	})
}

func (s Store) FeedbackTotals(ctx context.Context, tenantID, contentType, contentID string) (FeedbackTotals, error) {
	if err := s.validate(); err != nil {
		return FeedbackTotals{}, err
	}
	if err := contextErr(ctx); err != nil {
		return FeedbackTotals{}, err
	}
	if strings.TrimSpace(tenantID) == "" || len(tenantID) > 128 || (contentType != "news" && contentType != "faq") || strings.TrimSpace(contentID) == "" || len(contentID) > 128 {
		return FeedbackTotals{}, errors.New("invalid feedback query")
	}
	query := `SELECT reaction,SUM(event_count) FROM analytics_feedback_daily WHERE tenant_id=` + placeholder(s.Dialect, 1) + ` AND content_type=` + placeholder(s.Dialect, 2) + ` AND content_id=` + placeholder(s.Dialect, 3) + ` GROUP BY reaction`
	var totals FeedbackTotals
	err := s.withTenantQuery(ctx, tenantID, func(queryer database.QueryExecutor) error {
		rows, err := queryer.QueryContext(ctx, query, tenantID, contentType, contentID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var reaction string
			var count int64
			if err := rows.Scan(&reaction, &count); err != nil {
				return err
			}
			if reaction == "up" {
				totals.Up = count
			} else if reaction == "down" {
				totals.Down = count
			}
		}
		return rows.Err()
	})
	if err != nil {
		return FeedbackTotals{}, err
	}
	return totals, nil
}

func (s Store) FeedbackCounts(ctx context.Context, tenantID, contentType string, contentIDs []string) (map[string]analytics.FeedbackTotals, error) {
	if len(contentIDs) == 0 {
		return map[string]analytics.FeedbackTotals{}, nil
	}
	if len(contentIDs) > 100 || strings.TrimSpace(tenantID) == "" || len(tenantID) > 128 || (contentType != "news" && contentType != "faq") {
		return nil, errors.New("invalid feedback count query")
	}
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	args := []any{tenantID, contentType}
	placeholders := make([]string, 0, len(contentIDs))
	for _, id := range contentIDs {
		if strings.TrimSpace(id) == "" || len(id) > 128 {
			return nil, errors.New("invalid feedback content ID")
		}
		args = append(args, id)
		placeholders = append(placeholders, placeholder(s.Dialect, len(args)))
	}
	query := `SELECT content_id,reaction,SUM(event_count) FROM analytics_feedback_daily WHERE tenant_id=` + placeholder(s.Dialect, 1) + ` AND content_type=` + placeholder(s.Dialect, 2) + ` AND content_id IN (` + strings.Join(placeholders, ",") + ") GROUP BY content_id,reaction"
	counts := make(map[string]analytics.FeedbackTotals, len(contentIDs))
	err := s.withTenantQuery(ctx, tenantID, func(queryer database.QueryExecutor) error {
		rows, err := queryer.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id, reaction string
			var count int64
			if err := rows.Scan(&id, &reaction, &count); err != nil {
				return err
			}
			totals := counts[id]
			if reaction == "up" {
				totals.Up = count
			} else if reaction == "down" {
				totals.Down = count
			}
			counts[id] = totals
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return counts, nil
}
