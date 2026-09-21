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

func (s Store) validate() error {
	if err := database.RequireExecutor(s.Executor); err != nil {
		return err
	}
	if s.Dialect != database.SQLite && s.Dialect != database.Postgres {
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
