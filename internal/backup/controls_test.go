package backup

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	runRestoreControlsContract(t, db, database.SQLite, "sqlite")
}

func TestRestoreControlsRejectMissingDatabaseAndDialect(t *testing.T) {
	ctx := context.Background()
	controls := RestoreControls{SessionsRevoked: true, EgressQuarantined: true, OldWritesBlocked: true}
	if err := ApplyRestoreControls(ctx, nil, database.SQLite, controls); err == nil {
		t.Fatal("expected missing database error")
	}
	if err := ApplyRestoreControls(ctx, &sql.DB{}, database.Dialect("oracle"), controls); err == nil {
		t.Fatal("expected unsupported dialect error")
	}
	if _, err := RestoreWritesBlocked(ctx, nil, database.SQLite); err == nil {
		t.Fatal("expected missing database error")
	}
}

func TestPostgresRestoreControlsRevokeSessionsAndQuarantineJobs(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	admin, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "restore_contract_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	defer func() {
		_, _ = admin.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE")
		_ = admin.Close()
	}()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema)
	parsed.RawQuery = query.Encode()
	db, err := database.Open(ctx, database.Postgres, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.Postgres, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runRestoreControlsContract(t, db, database.Postgres, "postgres")
}

func runRestoreControlsContract(t *testing.T, db *sql.DB, dialect database.Dialect, suffix string) {
	t.Helper()
	ctx := context.Background()
	userID := "restore-user-" + suffix
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id,email,password_hash,role,created_at,updated_at) VALUES (`+placeholders(dialect, 6)+`)`, userID, userID+"@example.test", "hash", "admin", "2026-01-01", "2026-01-01"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sessions (id,user_id,token_hash,expires_at,created_at,last_seen_at) VALUES (`+placeholders(dialect, 6)+`)`, "restore-session-"+suffix, userID, "token", "2027-01-01", "2026-01-01", "2026-01-01"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	store := jobstore.Store{Dialect: dialect}
	job := jobs.Job{ID: "restore-job-" + suffix, Type: "news.publish", Payload: "{}", AvailableAt: now, CreatedAt: now}
	if err := store.Enqueue(ctx, db, job); err != nil {
		t.Fatal(err)
	}
	if err := ApplyRestoreControls(ctx, db, dialect, RestoreControls{SessionsRevoked: true, EgressQuarantined: true, OldWritesBlocked: true}); err != nil {
		t.Fatal(err)
	}
	var sessions int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions").Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if sessions != 0 {
		t.Fatalf("restored sessions = %d", sessions)
	}
	blocked, err := RestoreWritesBlocked(ctx, db, dialect)
	if err != nil || !blocked {
		t.Fatalf("old writes blocked = %v/%v", blocked, err)
	}
	if _, _, ok, err := store.Claim(ctx, db, "restore-worker", now, time.Minute); err != nil || ok {
		t.Fatalf("quarantined job claim = %v/%v", ok, err)
	}
	if _, err := db.ExecContext(ctx, pageInsert(dialect), pageArgs("blocked-page-"+suffix, "blocked")...); err == nil {
		t.Fatal("old deployment write unexpectedly succeeded")
	}
	if err := ReleaseRestoreQuarantine(ctx, db, dialect); err != nil {
		t.Fatal(err)
	}
	if err := ReleaseRestoreWrites(ctx, db, dialect); err != nil {
		t.Fatal(err)
	}
	if _, _, ok, err := store.Claim(ctx, db, "restore-worker", now, time.Minute); err != nil || !ok {
		t.Fatalf("released job claim = %v/%v", ok, err)
	}
	if _, err := db.ExecContext(ctx, pageInsert(dialect), pageArgs("new-page-"+suffix, "new")...); err != nil {
		t.Fatalf("write fence release: %v", err)
	}
}

func pageInsert(dialect database.Dialect) string {
	return `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES (` + placeholders(dialect, 11) + `)`
}

func pageArgs(id, slug string) []any {
	return []any{id, "Title", 1, "en", slug, "body", "draft", 1, 0, "2026-01-01", "2026-01-01"}
}

func placeholders(dialect database.Dialect, count int) string {
	parts := make([]string, count)
	for i := range parts {
		if dialect == database.Postgres {
			parts[i] = fmt.Sprintf("$%d", i+1)
		} else {
			parts[i] = "?"
		}
	}
	return strings.Join(parts, ",")
}
