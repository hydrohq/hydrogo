package sqlstore

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestNotificationInboxIsUserScopedAndMutable(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "admin.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := Store{Dialect: database.SQLite}
	if err := store.Add(ctx, db, "user-a", "Page published", "Your page is live.", "/admin/pages"); err != nil {
		t.Fatal(err)
	}
	count, err := store.UnreadCount(ctx, db, "user-a")
	if err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	items, err := store.List(ctx, db, "user-a", 10)
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%v err=%v", items, err)
	}
	if err := store.MarkAllRead(ctx, db, "user-b"); err != nil {
		t.Fatal(err)
	}
	count, _ = store.UnreadCount(ctx, db, "user-a")
	if count != 1 {
		t.Fatalf("other user's mark-read changed count to %d", count)
	}
	if err := store.Dismiss(ctx, db, "user-a", items[0].ID); err != nil {
		t.Fatal(err)
	}
	items, err = store.List(ctx, db, "user-a", 10)
	if err != nil || len(items) != 0 {
		t.Fatalf("dismissed items=%v err=%v", items, err)
	}
}

func TestNotificationRejectsUnsafeTarget(t *testing.T) {
	store := Store{Dialect: database.SQLite}
	if err := store.Add(context.Background(), nil, "u", "t", "m", "//outside.example"); err == nil {
		t.Fatal("unsafe target accepted")
	}
}

func TestDigestPreferenceOptInAndClaimAreIdempotent(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "digest.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := Store{Dialect: database.SQLite}
	if err := store.SetDigestPreference(ctx, db, "admin-a", "Europe/Istanbul", true); err != nil {
		t.Fatal(err)
	}
	enabled, zone, err := store.DigestPreferenceFor(ctx, db, "admin-a")
	if err != nil || !enabled || zone != "Europe/Istanbul" {
		t.Fatalf("preference=%v %q err=%v", enabled, zone, err)
	}
	items, err := store.DigestPreferences(ctx, db)
	if err != nil || len(items) != 1 || items[0].UserID != "admin-a" {
		t.Fatalf("items=%v err=%v", items, err)
	}
	week := "2026-09-21"
	if err := (database.NewTenantTransactor(db, database.SQLite, false)).Within(ctx, func(tx database.Executor) error {
		claimed, err := store.ClaimDigestWeek(ctx, tx, "admin-a", week)
		if err != nil {
			return err
		}
		if !claimed {
			t.Fatal("first digest claim was not acquired")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := (database.NewTenantTransactor(db, database.SQLite, false)).Within(ctx, func(tx database.Executor) error {
		claimed, err := store.ClaimDigestWeek(ctx, tx, "admin-a", week)
		if err != nil {
			return err
		}
		if claimed {
			t.Fatal("duplicate digest claim was acquired")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestWeeklyDigestStatsReportsLeadsOpenItemsAndReferrers(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "stats.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 7)
	for i, item := range []struct {
		status  string
		created time.Time
	}{{"new", start.Add(time.Hour)}, {"contacted", start.Add(2 * time.Hour)}, {"won", start.Add(3 * time.Hour)}} {
		id := fmt.Sprintf("digest-lead-%d", i)
		if _, err := db.ExecContext(ctx, "INSERT INTO form_inbox (id,form_key,email,payload,source_ip_hash,status,created_at,tenant_id,lead_status) VALUES (?,?,?,?,?,?,?,?,?)", id, "contact", "owner@example.test", "{}", "hash", "new", item.created.Format(time.RFC3339Nano), "", item.status); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range []struct {
		day, host string
		count     int
	}{{"2026-09-22", "search.example", 8}, {"2026-09-23", "search.example", 9}, {"2026-09-22", "social.example", 12}} {
		if _, err := db.ExecContext(ctx, "INSERT INTO analytics_daily_stats (day,tenant_id,metric,dimension,event_count) VALUES (?,?,?,?,?)", item.day, "", "referrer", item.host, item.count); err != nil {
			t.Fatal(err)
		}
	}
	stats, err := (Store{Dialect: database.SQLite}).WeeklyDigestStats(ctx, db, start, end, true)
	if err != nil {
		t.Fatal(err)
	}
	if stats.NewLeads != 3 || stats.UnresolvedLeads != 2 || len(stats.TopReferrers) != 2 || stats.TopReferrers[0].Host != "search.example" || stats.TopReferrers[0].Count != 17 || stats.TopReferrers[1].Host != "social.example" {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestDigestPreferencesUseBoundedKeysetPages(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "pages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	statement, err := db.PrepareContext(ctx, "INSERT INTO admin_digest_preferences (tenant_id,user_id,enabled,timezone) VALUES ('',?,1,'UTC')")
	if err != nil {
		t.Fatal(err)
	}
	defer statement.Close()
	for i := 0; i < 501; i++ {
		if _, err := statement.ExecContext(ctx, fmt.Sprintf("digest-%04d", i)); err != nil {
			t.Fatal(err)
		}
	}
	store := Store{Dialect: database.SQLite}
	first, err := store.DigestPreferencesAfter(ctx, db, "", 500)
	if err != nil || len(first) != 500 {
		t.Fatalf("first page=%d err=%v", len(first), err)
	}
	second, err := store.DigestPreferencesAfter(ctx, db, first[len(first)-1].UserID, 500)
	if err != nil || len(second) != 1 || second[0].UserID != "digest-0500" {
		t.Fatalf("second page=%v err=%v", second, err)
	}
}
