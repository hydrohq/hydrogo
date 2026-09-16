package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func testStore(t *testing.T) Store {
	t.Helper()
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "analytics.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE analytics_consent (consent_id TEXT PRIMARY KEY, decision TEXT NOT NULL, purpose_version TEXT NOT NULL, decided_at TEXT NOT NULL, revoked_at TEXT NULL); CREATE TABLE analytics_daily (day TEXT NOT NULL, event_type TEXT NOT NULL, country TEXT NOT NULL, purpose_version TEXT NOT NULL, event_count INTEGER NOT NULL, duration_seconds INTEGER NOT NULL, PRIMARY KEY(day,event_type,country,purpose_version))`); err != nil {
		t.Fatal(err)
	}
	return Store{Executor: db, Dialect: database.SQLite}
}

func TestStoreRequiresCurrentGrantedConsentAndAggregates(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	when := time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)
	event := v1.AnalyticsEvent{Type: v1.AnalyticsSessionDuration, Day: "2026-09-16", DurationSeconds: 12}
	if err := s.RecordEvent(ctx, "opaque", "v1", event); !errors.Is(err, ErrConsentRequired) {
		t.Fatalf("without consent=%v", err)
	}
	if err := s.SaveConsent(ctx, "opaque", v1.AnalyticsConsent{Decision: v1.ConsentGranted, PurposeVersion: "v1", DecidedAt: when}); err != nil {
		t.Fatal(err)
	}
	var storedID string
	if err := s.Executor.QueryRowContext(ctx, `SELECT consent_id FROM analytics_consent`).Scan(&storedID); err != nil {
		t.Fatal(err)
	}
	if storedID == "opaque" || len(storedID) != 64 {
		t.Fatalf("consent handle was not anonymized: %q", storedID)
	}
	if err := s.RecordEvent(ctx, "opaque", "v1", event); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordEvent(ctx, "opaque", "v1", event); err != nil {
		t.Fatal(err)
	}
	var count, duration int
	if err := s.Executor.QueryRowContext(ctx, `SELECT event_count,duration_seconds FROM analytics_daily`).Scan(&count, &duration); err != nil {
		t.Fatal(err)
	}
	if count != 2 || duration != 24 {
		t.Fatalf("aggregate=%d/%d", count, duration)
	}
	if err := s.SaveConsent(ctx, "opaque", v1.AnalyticsConsent{Decision: v1.ConsentRevoked, PurposeVersion: "v1", DecidedAt: when, RevokedAt: ptrTime(when.Add(time.Minute))}); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordEvent(ctx, "opaque", "v1", event); !errors.Is(err, ErrConsentRequired) {
		t.Fatalf("after revoke=%v", err)
	}
}

func ptrTime(value time.Time) *time.Time { return &value }
