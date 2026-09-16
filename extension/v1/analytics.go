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
	if len(e.Day) != 10 || e.Day[4] != '-' || e.Day[7] != '-' {
		return fmt.Errorf("analytics event day must be YYYY-MM-DD")
	}
	if parsed, err := time.Parse("2006-01-02", e.Day); err != nil || parsed.Format("2006-01-02") != e.Day {
		return fmt.Errorf("analytics event day must be a valid UTC date")
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
