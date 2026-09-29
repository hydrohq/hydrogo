package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/testmysql"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteNoCookieDailyStatsContract(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "no-cookie.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runNoCookieStatsContract(t, db, database.SQLite)
}

func TestSQLiteFeedbackUsesTenantScopedDailyAggregates(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "feedback.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runFeedbackContract(t, db, database.SQLite)
	runSearchInsightsContract(t, db, database.SQLite)
}

func runFeedbackContract(t *testing.T, db database.QueryExecutor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	store := Store{Executor: db, Dialect: dialect}
	day := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	for _, reaction := range []string{"up", "up", "down"} {
		if err := store.RecordFeedback(ctx, "tenant-a", "news", "article-a", reaction, day); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.RecordFeedback(ctx, "tenant-b", "news", "article-a", "up", day); err != nil {
		t.Fatal(err)
	}
	totals, err := store.FeedbackTotals(ctx, "tenant-a", "news", "article-a")
	if err != nil || totals.Up != 2 || totals.Down != 1 {
		t.Fatalf("totals=%+v err=%v", totals, err)
	}
	counts, err := store.FeedbackCounts(ctx, "tenant-b", "news", []string{"article-a", "other"})
	if err != nil || counts["article-a"].Up != 1 || counts["article-a"].Down != 0 {
		t.Fatalf("counts=%+v err=%v", counts, err)
	}
	if err := store.RecordFeedback(ctx, "tenant-a", "news", "article-a", "maybe", day); err == nil {
		t.Fatal("invalid reaction accepted")
	}
	if err := store.RecordFeedback(ctx, "tenant-a", "news", "article-a", "up", day.AddDate(0, 0, 2)); err != nil {
		t.Fatal(err)
	}
	if err := store.PurgeBefore(ctx, day.AddDate(0, 0, 1)); err != nil {
		t.Fatal(err)
	}
	totals, err = store.FeedbackTotals(ctx, "tenant-a", "news", "article-a")
	if err != nil || totals.Up != 1 || totals.Down != 0 {
		t.Fatalf("retained totals=%+v err=%v", totals, err)
	}
}

func TestMySQLNoCookieDailyStatsContract(t *testing.T) {
	db := testmysql.Open(t)
	testmysql.MigrateAll(t, db)
	runFeedbackContract(t, db, database.MySQL)
	runSearchInsightsContract(t, db, database.MySQL)
	runNoCookieStatsContract(t, db, database.MySQL)
}

func TestPostgresNoCookieDailyStatsContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	db, err := database.Open(context.Background(), database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.Postgres, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runFeedbackContract(t, db, database.Postgres)
	runSearchInsightsContract(t, db, database.Postgres)
	runNoCookieStatsContract(t, db, database.Postgres)
}

func runSearchInsightsContract(t *testing.T, db database.QueryExecutor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	store := Store{Executor: db, Dialect: dialect}
	tenantA := fmt.Sprintf("search-a-%d", time.Now().UnixNano())
	tenantB := tenantA + "-b"
	for _, event := range []analytics.SearchQueryEvent{{Day: "2026-09-29", TenantID: tenantA, Term: "Hydro", Locale: "tr", ResultCount: 0}, {Day: "2026-09-29", TenantID: tenantA, Term: "hydro", Locale: "tr", ResultCount: 2}, {Day: "2026-09-29", TenantID: tenantB, Term: "hydro", Locale: "tr", ResultCount: 0}} {
		if err := store.RecordSearchQuery(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := store.ListSearchInsights(ctx, tenantA, "2026-09-29", "2026-09-29", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Term != "hydro" || rows[0].Locale != "tr" || rows[0].SearchCount != 2 || rows[0].ZeroResultCount != 1 {
		t.Fatalf("tenant-a rows=%+v", rows)
	}
	zeroRows, err := store.ListZeroResultInsights(ctx, tenantA, "2026-09-29", "2026-09-29", 10)
	if err != nil || len(zeroRows) != 1 || zeroRows[0].ZeroResultCount != 1 {
		t.Fatalf("zero result rows=%+v err=%v", zeroRows, err)
	}
	rows, err = store.ListSearchInsights(ctx, tenantB, "2026-09-29", "2026-09-29", 10)
	if err != nil || len(rows) != 1 || rows[0].SearchCount != 1 || rows[0].ZeroResultCount != 1 {
		t.Fatalf("tenant-b rows=%+v err=%v", rows, err)
	}
	if err := store.PurgeBefore(ctx, time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	rows, err = store.ListSearchInsights(ctx, tenantA, "2026-09-29", "2026-09-29", 10)
	if err != nil || len(rows) != 0 {
		t.Fatalf("retained rows=%+v err=%v", rows, err)
	}
}

func runNoCookieStatsContract(t *testing.T, db database.QueryExecutor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	store := Store{Executor: db, Dialect: dialect}
	day := "2026-09-28"
	tenantID := fmt.Sprintf("analytics-%d", time.Now().UnixNano())
	visit := analytics.NoCookieVisit{Day: day, TenantID: tenantID, Page: "/public/pages/en/home", Referrer: "search.example", Device: "desktop", Browser: "Chrome", OS: "Linux", VisitorKey: strings.Repeat("a", 64)}
	if unique, err := store.RecordNoCookieVisit(ctx, visit); err != nil || !unique {
		t.Fatalf("first visit unique=%v err=%v", unique, err)
	}
	if unique, err := store.RecordNoCookieVisit(ctx, visit); err != nil || unique {
		t.Fatalf("repeat visit unique=%v err=%v", unique, err)
	}
	stats, err := store.ListDailyStats(ctx, day, tenantID, 20)
	if err != nil {
		t.Fatal(err)
	}
	counts := make(map[string]int64, len(stats))
	for _, row := range stats {
		counts[row.Metric+"\x00"+row.Dimension] = row.EventCount
	}
	if counts["unique_visitors\x00"] != 1 || counts["page\x00/public/pages/en/home"] != 2 || counts["browser\x00Chrome"] != 2 || len(stats) != 6 {
		t.Fatalf("daily aggregate contract = %#v", counts)
	}
	ranged, err := store.ListDailyStatsBetween(ctx, "2026-09-01", day, tenantID, 100)
	if err != nil || len(ranged) != len(stats) {
		t.Fatalf("daily range rows=%d err=%v", len(ranged), err)
	}
	if err := store.PurgeNoCookieBefore(ctx, "2026-09-29", "2026-09-29"); err != nil {
		t.Fatal(err)
	}
	stats, err = store.ListDailyStats(ctx, day, tenantID, 20)
	if err != nil || len(stats) != 0 {
		t.Fatalf("retention cleanup rows=%d err=%v", len(stats), err)
	}
}

func testStore(t *testing.T) Store {
	t.Helper()
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "analytics.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE analytics_consent (consent_id TEXT PRIMARY KEY, decision TEXT NOT NULL, purpose_version TEXT NOT NULL, decided_at TEXT NOT NULL, revoked_at TEXT NULL); CREATE TABLE analytics_daily (day TEXT NOT NULL, event_type TEXT NOT NULL, country TEXT NOT NULL, purpose_version TEXT NOT NULL, event_count INTEGER NOT NULL, duration_seconds INTEGER NOT NULL, PRIMARY KEY(day,event_type,country,purpose_version)); CREATE TABLE analytics_content_daily (day TEXT NOT NULL, tenant_id TEXT NOT NULL, content_type TEXT NOT NULL, content_id TEXT NOT NULL, locale TEXT NOT NULL, purpose_version TEXT NOT NULL, view_count INTEGER NOT NULL, PRIMARY KEY(day,tenant_id,content_type,content_id,locale,purpose_version))`); err != nil {
		t.Fatal(err)
	}
	return Store{Executor: db, Dialect: database.SQLite}
}

func TestStoreRecordAndListContentViews(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE analytics_consent (consent_id TEXT PRIMARY KEY, decision TEXT NOT NULL, purpose_version TEXT NOT NULL, decided_at TEXT NOT NULL, revoked_at TEXT NULL); CREATE TABLE analytics_content_daily (day TEXT NOT NULL, tenant_id TEXT NOT NULL, content_type TEXT NOT NULL, content_id TEXT NOT NULL, locale TEXT NOT NULL, purpose_version TEXT NOT NULL, view_count INTEGER NOT NULL, PRIMARY KEY(day,tenant_id,content_type,content_id,locale,purpose_version))`)
	if err != nil {
		t.Fatal(err)
	}
	store := Store{Executor: db, Dialect: database.SQLite}
	consentID := "content-consent"
	if err := store.SaveConsent(context.Background(), consentID, v1.AnalyticsConsent{Decision: v1.ConsentGranted, PurposeVersion: "v1", DecidedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	event := v1.ContentViewEvent{Day: "2026-09-22", TenantID: "tenant-a", ContentType: "article", ContentID: "a-1", Locale: "tr"}
	if err := store.RecordContentView(context.Background(), consentID, "v1", event); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordContentView(context.Background(), consentID, "v1", event); err != nil {
		t.Fatal(err)
	}
	rows, err := store.ListContentViews(context.Background(), ContentViewFilter{TenantID: "tenant-a", ContentType: "article"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ViewCount != 2 || rows[0].ContentID != "a-1" {
		t.Fatalf("content view rows = %#v", rows)
	}
}

func TestContentViewCountsAreBoundedToTenantAndType(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE analytics_content_daily (day TEXT NOT NULL, tenant_id TEXT NOT NULL, content_type TEXT NOT NULL, content_id TEXT NOT NULL, locale TEXT NOT NULL, purpose_version TEXT NOT NULL, view_count INTEGER NOT NULL, PRIMARY KEY(day,tenant_id,content_type,content_id,locale,purpose_version)); INSERT INTO analytics_content_daily VALUES ('2026-09-01','tenant-a','page','p1','en','v1',3),('2026-09-02','tenant-a','page','p1','tr','v1',4),('2026-09-01','tenant-a','page','p2','en','v1',8),('2026-09-01','tenant-b','page','p1','en','v1',100),('2026-09-01','tenant-a','news','p1','en','v1',200)`)
	if err != nil {
		t.Fatal(err)
	}
	store := Store{Executor: db, Dialect: database.SQLite}
	counts, err := store.ContentViewCounts(context.Background(), "tenant-a", "page", []string{"p1", "p2", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if counts["p1"] != 7 || counts["p2"] != 8 || counts["missing"] != 0 || len(counts) != 2 {
		t.Fatalf("view counts = %#v", counts)
	}
	if _, err := store.ContentViewCounts(context.Background(), "tenant-a", "page", make([]string, 101)); err == nil {
		t.Fatal("accepted an unbounded view count query")
	}
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
