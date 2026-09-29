package sqlstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestLoginSecurityTrackerSnapshotsPreviousLoginAndAttempts(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "auth.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := NewUserStore(db, database.SQLite)
	if err := store.Create(ctx, &auth.User{ID: "u1", Email: "owner@example.test", PasswordHash: "hash", Role: auth.RoleAdmin, Active: true}); err != nil {
		t.Fatal(err)
	}
	first := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	firstNotice, err := store.RecordSuccessfulLogin(ctx, "u1", "192.0.2.10", first)
	if err != nil || firstNotice.PreviousLoginAt != nil || firstNotice.FailedAttempts != 0 {
		t.Fatalf("first notice=%+v err=%v", firstNotice, err)
	}
	if err := store.RecordFailedLogin(ctx, "OWNER@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordFailedLogin(ctx, "unknown@example.test"); err != nil {
		t.Fatal(err)
	}
	second := first.Add(24 * time.Hour)
	notice, err := store.RecordSuccessfulLogin(ctx, "u1", "192.0.2.11", second)
	if err != nil {
		t.Fatal(err)
	}
	if notice.PreviousLoginAt == nil || !notice.PreviousLoginAt.Equal(first) || notice.FailedAttempts != 1 {
		t.Fatalf("notice=%+v", notice)
	}
	var ip string
	var failed int
	if err := db.QueryRowContext(ctx, "SELECT last_login_ip,failed_login_attempts FROM users WHERE id='u1'").Scan(&ip, &failed); err != nil || ip != "192.0.2.11" || failed != 0 {
		t.Fatalf("ip=%q failed=%d err=%v", ip, failed, err)
	}
}
