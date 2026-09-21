package v1

import (
	"fmt"
	"strings"
	"time"
)

const (
	MaxAnalyticsPurposeVersion = 64
	MaxAnalyticsCountryCode    = 2
	MaxAnalyticsEventType      = 32
	MaxAnalyticsTenantID       = 128
	MaxAnalyticsContentType    = 32
	MaxAnalyticsContentID      = 128
	MaxAnalyticsLocale         = 16
)

type ConsentDecision string

const (
	ConsentRejected ConsentDecision = "rejected"
	ConsentGranted  ConsentDecision = "granted"
	ConsentRevoked  ConsentDecision = "revoked"
)

// AnalyticsConsent is an auditable purpose-version decision. It is separate
// from technical session/CSRF cookies and contains no visitor identity.
type AnalyticsConsent struct {
	Decision       ConsentDecision `json:"decision"`
	PurposeVersion string          `json:"purpose_version"`
	DecidedAt      time.Time       `json:"decided_at"`
	RevokedAt      *time.Time      `json:"revoked_at,omitempty"`
}

type AnalyticsEventType string

const (
	AnalyticsPageView        AnalyticsEventType = "page_view"
	AnalyticsSessionDuration AnalyticsEventType = "session_duration"
	AnalyticsCountryVisit    AnalyticsEventType = "country_visit"
)

// AnalyticsEvent intentionally contains only aggregate-safe dimensions. IP,
// full URL, account IDs, fingerprints and advertising IDs are not contract fields.
type AnalyticsEvent struct {
	Type            AnalyticsEventType `json:"type"`
	Day             string             `json:"day"`
	Country         string             `json:"country,omitempty"`
	DurationSeconds int                `json:"duration_seconds,omitempty"`
}

// ContentViewEvent is an aggregate-safe content view dimension. It carries
// no IP, account, fingerprint, full URL or advertising identifier. Consent
// is checked by the analytics store before the daily aggregate is updated.
type ContentViewEvent struct {
	Day         string `json:"day"`
	TenantID    string `json:"tenant_id"`
	ContentType string `json:"content_type"`
	ContentID   string `json:"content_id"`
	Locale      string `json:"locale,omitempty"`
}

func (e ContentViewEvent) Validate() error {
	if err := validateAnalyticsDay(e.Day); err != nil {
		return err
	}
	for _, field := range []struct {
		name  string
		value string
		max   int
	}{{"tenant_id", e.TenantID, MaxAnalyticsTenantID}, {"content_type", e.ContentType, MaxAnalyticsContentType}, {"content_id", e.ContentID, MaxAnalyticsContentID}} {
		name, value, max := field.name, field.value, field.max
		if strings.TrimSpace(value) == "" || len(value) > max {
			return fmt.Errorf("analytics %s is required and bounded", name)
		}
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("analytics %s contains invalid control characters", name)
		}
	}
	if e.Locale != "" && (len(e.Locale) > MaxAnalyticsLocale || strings.ContainsAny(e.Locale, "\r\n")) {
		return fmt.Errorf("analytics locale is bounded")
	}
	return nil
}

func (c AnalyticsConsent) Validate() error {
	if c.Decision != ConsentRejected && c.Decision != ConsentGranted && c.Decision != ConsentRevoked {
		return fmt.Errorf("invalid analytics consent decision")
	}
	if strings.TrimSpace(c.PurposeVersion) == "" || len(c.PurposeVersion) > MaxAnalyticsPurposeVersion || c.DecidedAt.IsZero() {
		return fmt.Errorf("invalid analytics consent metadata")
	}
	if c.Decision == ConsentRevoked && (c.RevokedAt == nil || c.RevokedAt.IsZero()) {
		return fmt.Errorf("revoked consent requires revoked_at")
	}
	if c.RevokedAt != nil && c.RevokedAt.Before(c.DecidedAt) {
		return fmt.Errorf("revoked_at cannot precede decided_at")
	}
	if c.Decision != ConsentRevoked && c.RevokedAt != nil {
		return fmt.Errorf("only revoked consent may have revoked_at")
	}
	return nil
}

func (e AnalyticsEvent) Validate() error {
	if e.Type != AnalyticsPageView && e.Type != AnalyticsSessionDuration && e.Type != AnalyticsCountryVisit {
		return fmt.Errorf("invalid analytics event type")
	}
	if err := validateAnalyticsDay(e.Day); err != nil {
		return err
	}
	if e.Country != "" && (len(e.Country) != MaxAnalyticsCountryCode || strings.ToUpper(e.Country) != e.Country) {
		return fmt.Errorf("analytics country must be ISO alpha-2")
	}
	if e.DurationSeconds < 0 || e.DurationSeconds > 86400 {
		return fmt.Errorf("analytics duration is out of bounds")
	}
	if e.Type == AnalyticsCountryVisit && e.Country == "" {
		return fmt.Errorf("country visit requires country")
	}
	if e.Type == AnalyticsSessionDuration && e.DurationSeconds == 0 {
		return fmt.Errorf("session duration requires duration")
	}
	return nil
}

func validateAnalyticsDay(day string) error {
	if len(day) != 10 || day[4] != '-' || day[7] != '-' {
		return fmt.Errorf("analytics event day must be YYYY-MM-DD")
	}
	if parsed, err := time.Parse("2006-01-02", day); err != nil || parsed.Format("2006-01-02") != day {
		return fmt.Errorf("analytics event day must be a valid UTC date")
	}
	return nil
}
