package sqlstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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
	store := Store{Dialect: dialect, MaxEntries: 1}
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
	deadLetters, err := store.ListDeadLetters(ctx, db, 10)
	if err != nil || len(deadLetters) != 1 || deadLetters[0].ID != job.ID || deadLetters[0].LastError != "permanent failure" {
		t.Fatalf("dead-letter listing = %#v/%v", deadLetters, err)
	}
	if err := store.RequeueDeadLetter(ctx, db, job.ID, now); err != nil {
		t.Fatal(err)
	}
	requeued, requeueToken, ok, err := store.Claim(ctx, db, "worker-d", now, time.Minute)
	if err != nil || !ok || requeued.ID != job.ID {
		t.Fatalf("requeued claim = %+v/%q/%v/%v", requeued, requeueToken, ok, err)
	}
	if err := store.Ack(ctx, db, requeued.ID, "worker-d", requeueToken); err != nil {
		t.Fatal(err)
	}
	if err := store.Enqueue(ctx, db, jobs.Job{ID: "job-2", Type: "news.publish", Payload: "{}", AvailableAt: now, CreatedAt: now}); err != nil {
		t.Fatalf("dead-lettered job consumed queue capacity: %v", err)
	}
	claimed, token, ok, err = store.Claim(ctx, db, "worker-e", now, time.Minute)
	if err != nil || !ok {
		t.Fatalf("second job claim = %+v/%q/%v/%v", claimed, token, ok, err)
	}
	if err := store.DeadLetter(ctx, db, claimed.ID, "worker-e", token, "cleanup me"); err != nil {
		t.Fatal(err)
	}
	removed, err := store.PurgeDeadLetters(ctx, db, now.Add(time.Second), 10)
	if err != nil || removed != 1 {
		t.Fatalf("dead-letter purge = %d/%v", removed, err)
	}
}

func TestDialectBooleanLiterals(t *testing.T) {
	if got := (Store{Dialect: database.SQLite}).boolLiteral(true); got != "1" {
		t.Fatalf("sqlite true literal = %q", got)
	}
	if got := (Store{Dialect: database.SQLite}).boolLiteral(false); got != "0" {
		t.Fatalf("sqlite false literal = %q", got)
	}
	if got := (Store{Dialect: database.Postgres}).boolLiteral(true); got != "TRUE" {
		t.Fatalf("postgres true literal = %q", got)
	}
	if got := (Store{Dialect: database.Postgres}).boolLiteral(false); got != "FALSE" {
		t.Fatalf("postgres false literal = %q", got)
	}
}

func TestSQLiteQueueCapacityIsBoundedForConcurrentProducers(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "jobs.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runConcurrentCapacityContract(t, db, database.SQLite)
}

func TestPostgresQueueCapacityIsBoundedForConcurrentProducers(t *testing.T) {
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
	runConcurrentCapacityContract(t, db, database.Postgres)
}

func runConcurrentCapacityContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	store := Store{Dialect: dialect, MaxEntries: 1}
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 1; i <= 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
			errs <- store.Enqueue(ctx, db, jobs.Job{ID: fmt.Sprintf("concurrent-%d", id), Type: "news.publish", Payload: "{}", AvailableAt: time.Now().UTC()})
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)

	successes := 0
	for err := range errs {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent enqueue successes = %d, want 1", successes)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs WHERE dead_lettered = 0").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count > 1 {
		t.Fatalf("claimable jobs = %d, exceeded capacity 1", count)
	}
}
