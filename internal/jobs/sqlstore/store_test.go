package sqlstore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/jobs"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteDurableJobLeaseContract(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "jobs.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runDurableJobLeaseContract(t, db, database.SQLite)
}

func TestPostgresDurableJobLeaseContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.Postgres, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM jobs"); err != nil {
		t.Fatal(err)
	}
	runDurableJobLeaseContract(t, db, database.Postgres)
}

func runDurableJobLeaseContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	store := Store{Dialect: dialect, MaxEntries: 2}
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	job := jobs.Job{ID: "job-1", Type: "news.publish", Payload: "{}", AvailableAt: now, CreatedAt: now}
	if err := store.Enqueue(ctx, db, job); err != nil {
		t.Fatal(err)
	}
	if err := store.Enqueue(ctx, db, job); err != nil {
		t.Fatalf("idempotent enqueue: %v", err)
	}
	if err := store.Enqueue(ctx, db, jobs.Job{ID: "job-1", Type: "other", Payload: "{}", AvailableAt: now, CreatedAt: now}); !errors.Is(err, jobs.ErrJobConflict) {
		t.Fatalf("conflicting enqueue = %v", err)
	}
	claimed, token, ok, err := store.Claim(ctx, db, "worker-a", now, time.Minute)
	if err != nil || !ok || claimed.ID != job.ID || token == "" || claimed.Attempts != 1 {
		t.Fatalf("claim = %+v/%q/%v/%v", claimed, token, ok, err)
	}
	if _, _, ok, err := store.Claim(ctx, db, "worker-b", now, time.Minute); err != nil || ok {
		t.Fatalf("leased job was claimed twice: %v/%v", ok, err)
	}
	if err := store.Nack(ctx, db, job.ID, "worker-b", token, now); !errors.Is(err, jobs.ErrLeaseInvalid) {
		t.Fatalf("wrong worker nack = %v", err)
	}
	if err := store.NackWithError(ctx, db, job.ID, "worker-a", token, now, "temporary failure"); err != nil {
		t.Fatal(err)
	}
	claimed, token, ok, err = store.Claim(ctx, db, "worker-b", now, time.Minute)
	if err != nil || !ok || claimed.Attempts != 2 || claimed.LastError != "temporary failure" {
		t.Fatalf("reclaim = %+v/%q/%v/%v", claimed, token, ok, err)
	}
	if err := store.DeadLetter(ctx, db, job.ID, "worker-b", token, "permanent failure"); err != nil {
		t.Fatal(err)
	}
	if _, _, ok, err := store.Claim(ctx, db, "worker-c", now, time.Minute); err != nil || ok {
		t.Fatalf("dead-lettered job remained claimable: %v/%v", ok, err)
	}
}
