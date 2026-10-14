package sqlstore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type ContentViewFilter struct {
	TenantID    string
	ContentType string
	ContentID   string
	Locale      string
	FromDay     string
	ToDay       string
	Limit       int
}

type ContentViewAggregate struct {
	Day         string
	TenantID    string
	ContentType string
	ContentID   string
	Locale      string
	ViewCount   int64
}

// ContentViewCounts returns bounded lifetime totals for the selected content
// IDs. The count remains consent-gated because only analytics aggregates are
// read here.
func (s Store) ContentViewCounts(ctx context.Context, tenantID, contentType string, contentIDs []string) (map[string]int64, error) {
	if len(contentIDs) == 0 || len(contentIDs) > 100 || strings.TrimSpace(tenantID) == "" || len(tenantID) > 128 || strings.TrimSpace(contentType) == "" || len(contentType) > 32 {
		return nil, errors.New("invalid content view count query")
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if err := s.validate(); err != nil {
		return nil, err
	}
	queryExecutor, ok := s.Executor.(database.QueryExecutor)
	if !ok {
		return nil, errors.New("analytics content counts require query executor")
	}
	args := []any{tenantID, contentType}
	placeholders := make([]string, 0, len(contentIDs))
	for _, id := range contentIDs {
		if strings.TrimSpace(id) == "" || len(id) > 128 {
			return nil, errors.New("invalid content view content ID")
		}
		args = append(args, id)
		placeholders = append(placeholders, placeholder(s.Dialect, len(args)))
	}
	query := "SELECT content_id,SUM(view_count) FROM analytics_content_daily WHERE tenant_id=" + placeholder(s.Dialect, 1) + " AND content_type=" + placeholder(s.Dialect, 2) + " AND content_id IN (" + strings.Join(placeholders, ",") + ") GROUP BY content_id"
	rows, err := queryExecutor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int64, len(contentIDs))
	for rows.Next() {
		var id string
		var count int64
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		counts[id] = count
	}
	return counts, rows.Err()
}

// RecordContentView updates a consent-gated daily aggregate. No visitor or
// request identity is accepted, making this safe for privacy-preserving
// content reports when called from a bounded asynchronous queue.
func (s Store) RecordContentView(ctx context.Context, consentID, purposeVersion string, event v1.ContentViewEvent) error {
	if err := s.validate(); err != nil {
		return err
	}
	if err := contextErr(ctx); err != nil {
		return err
	}
	consentID = strings.TrimSpace(consentID)
	if consentID == "" || len(consentID) > MaxConsentIDLength {
		return ErrConsentIDRequired
	}
	consentID = digestConsentID(consentID)
	if strings.TrimSpace(purposeVersion) == "" || len(purposeVersion) > v1.MaxAnalyticsPurposeVersion {
		return ErrConsentRequired
	}
	if err := event.Validate(); err != nil {
		return err
	}
	if err := s.requireGrantedConsent(ctx, consentID, purposeVersion); err != nil {
		return err
	}
	if s.Dialect == database.Postgres {
		_, err := s.Executor.ExecContext(ctx, `INSERT INTO analytics_content_daily (day,tenant_id,content_type,content_id,locale,purpose_version,view_count) VALUES ($1,$2,$3,$4,$5,$6,1)
			ON CONFLICT (day,tenant_id,content_type,content_id,locale,purpose_version) DO UPDATE SET view_count=analytics_content_daily.view_count+1`, event.Day, event.TenantID, event.ContentType, event.ContentID, event.Locale, purposeVersion)
		return err
	}
	if s.Dialect == database.MySQL {
		_, err := s.Executor.ExecContext(ctx, `INSERT INTO analytics_content_daily (day,tenant_id,content_type,content_id,locale,purpose_version,view_count) VALUES (?,?,?,?,?,?,1)
			ON DUPLICATE KEY UPDATE view_count=view_count+1`, event.Day, event.TenantID, event.ContentType, event.ContentID, event.Locale, purposeVersion)
		return err
	}
	_, err := s.Executor.ExecContext(ctx, `INSERT INTO analytics_content_daily (day,tenant_id,content_type,content_id,locale,purpose_version,view_count) VALUES (?,?,?,?,?,?,1)
		ON CONFLICT(day,tenant_id,content_type,content_id,locale,purpose_version) DO UPDATE SET view_count=view_count+1`, event.Day, event.TenantID, event.ContentType, event.ContentID, event.Locale, purposeVersion)
	return err
}

func (s Store) ListContentViews(ctx context.Context, filter ContentViewFilter) ([]ContentViewAggregate, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	queryExecutor, ok := s.Executor.(database.QueryExecutor)
	if !ok {
		return nil, errors.New("analytics content report requires query executor")
	}
	if strings.TrimSpace(filter.TenantID) == "" || len(filter.TenantID) > 128 {
		return nil, errors.New("invalid analytics tenant scope")
	}
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 100
	}
	clauses := []string{"1=1"}
	args := make([]any, 0, 6)
	add := func(column, operator, value string) {
		if strings.TrimSpace(value) != "" {
			args = append(args, strings.TrimSpace(value))
			clauses = append(clauses, column+" "+operator+" "+placeholder(s.Dialect, len(args)))
		}
	}
	add("tenant_id", "=", filter.TenantID)
	add("content_type", "=", filter.ContentType)
	add("content_id", "=", filter.ContentID)
	add("locale", "=", filter.Locale)
	add("day", ">=", filter.FromDay)
	add("day", "<=", filter.ToDay)
	query := "SELECT day,tenant_id,content_type,content_id,locale,view_count FROM analytics_content_daily WHERE " + strings.Join(clauses, " AND ") + " ORDER BY day DESC,view_count DESC LIMIT " + fmt.Sprintf("%d", filter.Limit)
	rows, err := queryExecutor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]ContentViewAggregate, 0)
	for rows.Next() {
		var item ContentViewAggregate
		if err := rows.Scan(&item.Day, &item.TenantID, &item.ContentType, &item.ContentID, &item.Locale, &item.ViewCount); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
