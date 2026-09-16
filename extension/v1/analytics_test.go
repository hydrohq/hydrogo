package v1

import (
	"testing"
	"time"
)

func TestAnalyticsConsentAndEventContract(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	if err := (AnalyticsConsent{Decision: ConsentGranted, PurposeVersion: "v1", DecidedAt: now}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (AnalyticsEvent{Type: AnalyticsPageView, Day: "2026-09-16"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (AnalyticsEvent{Type: AnalyticsCountryVisit, Day: "2026-09-16", Country: "tr"}).Validate(); err == nil {
		t.Fatal("expected non-uppercase country rejection")
	}
}

func TestAnalyticsConsentRejectsInvalidRevocation(t *testing.T) {
	if err := (AnalyticsConsent{Decision: ConsentRevoked, PurposeVersion: "v1", DecidedAt: time.Now()}).Validate(); err == nil {
		t.Fatal("expected revoked_at requirement")
	}
}

func TestAnalyticsEventRejectsImpossibleDate(t *testing.T) {
	if err := (AnalyticsEvent{Type: AnalyticsPageView, Day: "2026-02-30"}).Validate(); err == nil {
		t.Fatal("expected impossible date rejection")
	}
}

func TestAnalyticsConsentRejectsRevokeBeforeDecision(t *testing.T) {
	decided := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	revoked := decided.Add(-time.Minute)
	if err := (AnalyticsConsent{Decision: ConsentRevoked, PurposeVersion: "v1", DecidedAt: decided, RevokedAt: &revoked}).Validate(); err == nil {
		t.Fatal("expected invalid consent time ordering")
	}
}
