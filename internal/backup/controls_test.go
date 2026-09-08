package backup

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/jobs"
	jobstore "github.com/ardaozyildirim/hydrogo/internal/jobs/sqlstore"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteRestoreControlsRevokeSessionsAndQuarantineJobs(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "restore.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id,email,password_hash,role,created_at,updated_at) VALUES ('restore-user','restore@example.test','hash','admin','2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sessions (id,user_id,token_hash,expires_at,created_at,last_seen_at) VALUES ('restore-session','restore-user','token','2027-01-01','2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	store := jobstore.Store{Dialect: database.SQLite}
	job := jobs.Job{ID: "restore-job", Type: "news.publish", Payload: "{}", AvailableAt: now, CreatedAt: now}
	if err := store.Enqueue(ctx, db, job); err != nil {
		t.Fatal(err)
	}
	if err := ApplyRestoreControls(ctx, db, database.SQLite, RestoreControls{SessionsRevoked: true, EgressQuarantined: true, OldWritesBlocked: true}); err != nil {
		t.Fatal(err)
	}
	var sessions int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions").Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if sessions != 0 {
		t.Fatalf("restored sessions = %d", sessions)
	}
	blocked, err := RestoreWritesBlocked(ctx, db, database.SQLite)
	if err != nil || !blocked {
		t.Fatalf("old writes blocked = %v/%v", blocked, err)
	}
	if _, _, ok, err := store.Claim(ctx, db, "restore-worker", now, time.Minute); err != nil || ok {
		t.Fatalf("quarantined job claim = %v/%v", ok, err)
	}
	if err := ReleaseRestoreQuarantine(ctx, db, database.SQLite); err != nil {
		t.Fatal(err)
	}
	if _, _, ok, err := store.Claim(ctx, db, "restore-worker", now, time.Minute); err != nil || !ok {
		t.Fatalf("released job claim = %v/%v", ok, err)
	}
}
