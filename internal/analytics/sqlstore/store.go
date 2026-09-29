// Package sqlstore provides the optional, aggregate-only analytics store.
package sqlstore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var (
	ErrConsentRequired   = errors.New("current analytics consent is required")
	ErrConsentIDRequired = errors.New("analytics consent id is required")
)

const MaxConsentIDLength = 128

// Store persists only purpose-versioned consent and daily aggregate events.
// consentID is an opaque caller-generated value; only its fixed-length digest
// is written. No identity, IP or device information is accepted or written.
type Store struct {
	Executor database.Executor
	Dialect  database.Dialect
}

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

// FeedbackTotals contains only site-wide aggregate counts; no visitor identity is stored.
type FeedbackTotals = analytics.FeedbackTotals

type SearchInsight = analytics.SearchInsight

func (s Store) RecordSearchQuery(ctx context.Context, event analytics.SearchQueryEvent) error {
	if err := s.validate(); err != nil {
		return err
	}
	if err := contextErr(ctx); err != nil {
		return err
	}
	day, err := time.Parse("2006-01-02", event.Day)
	term := analytics.NormalizeSearchTerm(event.Term)
	if err != nil || strings.TrimSpace(event.TenantID) == "" || len(event.TenantID) > 128 || term == "" || event.ResultCount < 0 {
		return errors.New("invalid search analytics event")
	}
	zero := 0
	if event.ResultCount == 0 {
		zero = 1
	}
	date := day.UTC().Format("2006-01-02")
	locale := event.Locale
	if locale == "" {
		locale = "tr"
	}
	if len(locale) > 35 || strings.ContainsAny(locale, "\x00\r\n") {
		return errors.New("invalid search analytics locale")
	}
	query := `INSERT INTO analytics_search_daily(day,tenant_id,term,locale,search_count,zero_result_count) VALUES ($1,$2,$3,$4,1,$5) ON CONFLICT(day,tenant_id,term,locale) DO UPDATE SET search_count=analytics_search_daily.search_count+1,zero_result_count=analytics_search_daily.zero_result_count+EXCLUDED.zero_result_count`
	if s.Dialect == database.SQLite {
		query = `INSERT INTO analytics_search_daily(day,tenant_id,term,locale,search_count,zero_result_count) VALUES (?,?,?,?,1,?) ON CONFLICT(day,tenant_id,term,locale) DO UPDATE SET search_count=search_count+1,zero_result_count=zero_result_count+excluded.zero_result_count`
	}
	if s.Dialect == database.MySQL {
		query = `INSERT INTO analytics_search_daily(day,tenant_id,term,locale,search_count,zero_result_count) VALUES (?,?,?,?,1,?) ON DUPLICATE KEY UPDATE search_count=search_count+1,zero_result_count=zero_result_count+VALUES(zero_result_count)`
	}
	_, err = s.Executor.ExecContext(ctx, query, date, event.TenantID, term, locale, zero)
	return err
}

func (s Store) ListSearchInsights(ctx context.Context, tenantID, fromDay, toDay string, limit int) ([]SearchInsight, error) {
	return s.listSearchInsights(ctx, tenantID, fromDay, toDay, limit, false)
}

func (s Store) ListZeroResultInsights(ctx context.Context, tenantID, fromDay, toDay string, limit int) ([]SearchInsight, error) {
	return s.listSearchInsights(ctx, tenantID, fromDay, toDay, limit, true)
}

func (s Store) listSearchInsights(ctx context.Context, tenantID, fromDay, toDay string, limit int, zeroOnly bool) ([]SearchInsight, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(tenantID) == "" || len(tenantID) > 128 {
		return nil, errors.New("invalid search analytics tenant")
	}
	from, fromErr := time.Parse("2006-01-02", fromDay)
	to, toErr := time.Parse("2006-01-02", toDay)
	if fromErr != nil || toErr != nil || from.After(to) {
		return nil, errors.New("invalid search analytics range")
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	queryer, ok := s.Executor.(database.QueryExecutor)
	if !ok {
		return nil, errors.New("search insights require query executor")
	}
	filter := ""
	if zeroOnly {
		filter = " HAVING SUM(zero_result_count)>0"
	}
	query := `SELECT term,locale,SUM(search_count),SUM(zero_result_count) FROM analytics_search_daily WHERE day BETWEEN ` + placeholder(s.Dialect, 1) + ` AND ` + placeholder(s.Dialect, 2) + ` AND tenant_id=` + placeholder(s.Dialect, 3) + ` GROUP BY term,locale` + filter + ` ORDER BY `
	if zeroOnly {
		query += `SUM(zero_result_count) DESC,term,locale LIMIT `
	} else {
		query += `SUM(search_count) DESC,term,locale LIMIT `
	}
	query += fmt.Sprint(limit)
	rows, err := queryer.QueryContext(ctx, query, fromDay, toDay, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]SearchInsight, 0)
	for rows.Next() {
		var item SearchInsight
		if err := rows.Scan(&item.Term, &item.Locale, &item.SearchCount, &item.ZeroResultCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

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
	_, err := s.Executor.ExecContext(ctx, query, tenantID, contentType, contentID, dayValue, reaction)
	return err
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
	queryer, ok := s.Executor.(database.QueryExecutor)
	if !ok {
		return FeedbackTotals{}, errors.New("feedback totals require query executor")
	}
	query := `SELECT reaction,SUM(event_count) FROM analytics_feedback_daily WHERE tenant_id=` + placeholder(s.Dialect, 1) + ` AND content_type=` + placeholder(s.Dialect, 2) + ` AND content_id=` + placeholder(s.Dialect, 3) + ` GROUP BY reaction`
	rows, err := queryer.QueryContext(ctx, query, tenantID, contentType, contentID)
	if err != nil {
		return FeedbackTotals{}, err
	}
	defer rows.Close()
	var totals FeedbackTotals
	for rows.Next() {
		var reaction string
		var count int64
		if err := rows.Scan(&reaction, &count); err != nil {
			return FeedbackTotals{}, err
		}
		if reaction == "up" {
			totals.Up = count
		} else if reaction == "down" {
			totals.Down = count
		}
	}
	return totals, rows.Err()
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
	queryer, ok := s.Executor.(database.QueryExecutor)
	if !ok {
		return nil, errors.New("feedback counts require query executor")
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
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]analytics.FeedbackTotals, len(contentIDs))
	for rows.Next() {
		var id, reaction string
		var count int64
		if err := rows.Scan(&id, &reaction, &count); err != nil {
			return nil, err
		}
		totals := counts[id]
		if reaction == "up" {
			totals.Up = count
		} else if reaction == "down" {
			totals.Down = count
		}
		counts[id] = totals
	}
	return counts, rows.Err()
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

type DailyStat struct {
	Day, TenantID, Metric, Dimension string
	EventCount                       int64
}

func (s Store) validate() error {
	if err := database.RequireExecutor(s.Executor); err != nil {
		return err
	}
	if s.Dialect != database.SQLite && s.Dialect != database.Postgres && s.Dialect != database.MySQL {
		return fmt.Errorf("unsupported analytics dialect %q", s.Dialect)
	}
	return nil
}

func (s Store) SaveConsent(ctx context.Context, consentID string, consent v1.AnalyticsConsent) error {
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
	if err := consent.Validate(); err != nil {
		return err
	}
	if s.Dialect == database.Postgres {
		_, err := s.Executor.ExecContext(ctx, `INSERT INTO analytics_consent (consent_id,decision,purpose_version,decided_at,revoked_at) VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (consent_id) DO UPDATE SET decision=EXCLUDED.decision,purpose_version=EXCLUDED.purpose_version,decided_at=EXCLUDED.decided_at,revoked_at=EXCLUDED.revoked_at`, consentID, consent.Decision, consent.PurposeVersion, consent.DecidedAt.UTC(), nullableTime(consent.RevokedAt))
		return err
	}
	if s.Dialect == database.MySQL {
		_, err := s.Executor.ExecContext(ctx, `INSERT INTO analytics_consent (consent_id,decision,purpose_version,decided_at,revoked_at) VALUES (?,?,?,?,?)
			ON DUPLICATE KEY UPDATE decision=VALUES(decision),purpose_version=VALUES(purpose_version),decided_at=VALUES(decided_at),revoked_at=VALUES(revoked_at)`, consentID, consent.Decision, consent.PurposeVersion, consent.DecidedAt.UTC(), nullableTime(consent.RevokedAt))
		return err
	}
	_, err := s.Executor.ExecContext(ctx, `INSERT INTO analytics_consent (consent_id,decision,purpose_version,decided_at,revoked_at) VALUES (?,?,?,?,?)
		ON CONFLICT(consent_id) DO UPDATE SET decision=excluded.decision,purpose_version=excluded.purpose_version,decided_at=excluded.decided_at,revoked_at=excluded.revoked_at`, consentID, consent.Decision, consent.PurposeVersion, consent.DecidedAt.UTC(), nullableTime(consent.RevokedAt))
	return err
}

// RecordEvent accepts an event only when the opaque consent ID has a current
// granted decision for the same purpose version. It updates a daily aggregate,
// never a per-visitor event row.
func (s Store) RecordEvent(ctx context.Context, consentID, purposeVersion string, event v1.AnalyticsEvent) error {
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
		_, err := s.Executor.ExecContext(ctx, `INSERT INTO analytics_daily (day,event_type,country,purpose_version,event_count,duration_seconds) VALUES ($1,$2,$3,$4,1,$5)
			ON CONFLICT (day,event_type,country,purpose_version) DO UPDATE SET event_count=analytics_daily.event_count+1,duration_seconds=analytics_daily.duration_seconds+EXCLUDED.duration_seconds`, event.Day, event.Type, event.Country, purposeVersion, event.DurationSeconds)
		return err
	}
	if s.Dialect == database.MySQL {
		_, err := s.Executor.ExecContext(ctx, `INSERT INTO analytics_daily (day,event_type,country,purpose_version,event_count,duration_seconds) VALUES (?,?,?,?,1,?)
			ON DUPLICATE KEY UPDATE event_count=event_count+1,duration_seconds=duration_seconds+VALUES(duration_seconds)`, event.Day, event.Type, event.Country, purposeVersion, event.DurationSeconds)
		return err
	}
	_, err := s.Executor.ExecContext(ctx, `INSERT INTO analytics_daily (day,event_type,country,purpose_version,event_count,duration_seconds) VALUES (?,?,?,?,1,?)
		ON CONFLICT(day,event_type,country,purpose_version) DO UPDATE SET event_count=event_count+1,duration_seconds=duration_seconds+excluded.duration_seconds`, event.Day, event.Type, event.Country, purposeVersion, event.DurationSeconds)
	return err
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

func (s Store) requireGrantedConsent(ctx context.Context, consentID, purposeVersion string) error {
	var decision, version string
	query := "SELECT decision,purpose_version FROM analytics_consent WHERE consent_id = " + placeholder(s.Dialect, 1)
	if err := s.Executor.QueryRowContext(ctx, query, consentID).Scan(&decision, &version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrConsentRequired
		}
		return err
	}
	if decision != string(v1.ConsentGranted) || version != purposeVersion {
		return ErrConsentRequired
	}
	return nil
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
	visitorQuery := `INSERT INTO analytics_form_visitors(day,tenant_id,form_id,visitor_key) VALUES (` + placeholder(s.Dialect, 1) + `,` + placeholder(s.Dialect, 2) + `,` + placeholder(s.Dialect, 3) + `,` + placeholder(s.Dialect, 4) + `) ON CONFLICT(day,tenant_id,form_id,visitor_key) DO NOTHING`
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
	viewQuery := `INSERT INTO analytics_form_daily(day,tenant_id,form_id,view_count,submission_count) VALUES (` + placeholder(s.Dialect, 1) + `,` + placeholder(s.Dialect, 2) + `,` + placeholder(s.Dialect, 3) + `,1,0) ON CONFLICT(day,tenant_id,form_id) DO UPDATE SET view_count=analytics_form_daily.view_count+1`
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

func (s Store) ListDailyStats(ctx context.Context, day, tenantID string, limit int) ([]DailyStat, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	queryer, ok := s.Executor.(database.QueryExecutor)
	if !ok {
		return nil, errors.New("analytics stats query requires query executor")
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return nil, errors.New("invalid analytics report date")
	}
	query := `SELECT day,tenant_id,metric,dimension,event_count FROM analytics_daily_stats WHERE day=` + placeholder(s.Dialect, 1) + ` AND tenant_id=` + placeholder(s.Dialect, 2) + ` ORDER BY metric,event_count DESC,dimension LIMIT ` + fmt.Sprint(limit)
	rows, err := queryer.QueryContext(ctx, query, day, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]DailyStat, 0)
	for rows.Next() {
		var item DailyStat
		if err := rows.Scan(&item.Day, &item.TenantID, &item.Metric, &item.Dimension, &item.EventCount); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s Store) ListDailyStatsBetween(ctx context.Context, fromDay, toDay, tenantID string, limit int) ([]DailyStat, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	queryer, ok := s.Executor.(database.QueryExecutor)
	if !ok {
		return nil, errors.New("analytics stats query requires query executor")
	}
	if _, err := time.Parse("2006-01-02", fromDay); err != nil {
		return nil, errors.New("invalid analytics range start")
	}
	if _, err := time.Parse("2006-01-02", toDay); err != nil {
		return nil, errors.New("invalid analytics range end")
	}
	if fromDay > toDay {
		return nil, errors.New("invalid analytics date range")
	}
	if limit < 1 || limit > 5000 {
		limit = 1000
	}
	query := `SELECT day,tenant_id,metric,dimension,event_count FROM analytics_daily_stats WHERE day BETWEEN ` + placeholder(s.Dialect, 1) + ` AND ` + placeholder(s.Dialect, 2) + ` AND tenant_id=` + placeholder(s.Dialect, 3) + ` ORDER BY day DESC,metric,event_count DESC,dimension LIMIT ` + fmt.Sprint(limit)
	rows, err := queryer.QueryContext(ctx, query, fromDay, toDay, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]DailyStat, 0)
	for rows.Next() {
		var item DailyStat
		if err := rows.Scan(&item.Day, &item.TenantID, &item.Metric, &item.Dimension, &item.EventCount); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// PurgeBefore applies an operator-selected retention boundary to both consent
// metadata and daily aggregates. It is intentionally explicit; the caller
// owns the legal retention schedule.
func (s Store) PurgeBefore(ctx context.Context, before time.Time) error {
	if err := s.validate(); err != nil {
		return err
	}
	if err := contextErr(ctx); err != nil {
		return err
	}
	cutoff := before.UTC().Format("2006-01-02")
	if _, err := s.Executor.ExecContext(ctx, "DELETE FROM analytics_feedback_daily WHERE day < "+placeholder(s.Dialect, 1), cutoff); err != nil {
		return err
	}
	if _, err := s.Executor.ExecContext(ctx, "DELETE FROM analytics_search_daily WHERE day < "+placeholder(s.Dialect, 1), cutoff); err != nil {
		return err
	}
	if _, err := s.Executor.ExecContext(ctx, "DELETE FROM analytics_daily WHERE day < "+placeholder(s.Dialect, 1), cutoff); err != nil {
		return err
	}
	_, err := s.Executor.ExecContext(ctx, "DELETE FROM analytics_consent WHERE decided_at < "+placeholder(s.Dialect, 1), before.UTC())
	return err
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}

func placeholder(dialect database.Dialect, position int) string {
	if dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}

func digestConsentID(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func contextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
