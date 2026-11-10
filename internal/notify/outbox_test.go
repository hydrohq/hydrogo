package notify_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/notify"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

type mockFailingSender struct {
	err       error
	callCount int
}

func (m *mockFailingSender) Send(_ context.Context, _ notify.Message) error {
	m.callCount++
	return m.err
}

func setupOutboxTestDB(t *testing.T) (*sql.DB, notify.OutboxStore) {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test_notify_outbox.sqlite")
	db, err := database.Open(ctx, database.SQLite, dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	store := notify.OutboxStore{Dialect: database.SQLite, RequireTenantScope: false}
	return db, store
}

func TestOutboxWorkerRetriesWithExponentialBackoffOnTelegramFailure(t *testing.T) {
	ctx := context.Background()
	db, store := setupOutboxTestDB(t)

	// Enqueue a notification
	msg := notify.Message{Title: "Test Title", Body: "Order #1234 paid"}
	if err := store.Enqueue(ctx, db, "telegram", msg); err != nil {
		t.Fatalf("enqueue notification: %v", err)
	}

	telegramErr := errors.New("telegram: 502 Bad Gateway")
	sender := &mockFailingSender{err: telegramErr}

	initialBackoff := 100 * time.Millisecond
	worker := notify.Worker{
		DB:                  db,
		Store:               store,
		Sender:              sender,
		MaxDeliveryAttempts: 3,
		InitialBackoff:      initialBackoff,
		MaxBackoff:          10 * time.Second,
	}

	runTime1 := time.Now().UTC().Add(time.Second)

	// Attempt 1: should fail, record retry, backoff = 100ms
	processed, err := worker.RunOnce(ctx, runTime1)
	if err != nil {
		t.Fatalf("run once attempt 1: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 processed, got %d", processed)
	}
	if sender.callCount != 1 {
		t.Fatalf("expected 1 sender call, got %d", sender.callCount)
	}

	var attempts int
	var nextRaw string
	var sentRaw *string
	var lastErr string
	if err := db.QueryRowContext(ctx, "SELECT attempts, next_attempt_at, sent_at, last_error FROM notification_outbox LIMIT 1").Scan(&attempts, &nextRaw, &sentRaw, &lastErr); err != nil {
		t.Fatalf("query record attempt 1: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected attempts=1, got %d", attempts)
	}
	if sentRaw != nil {
		t.Fatalf("expected sent_at=nil, got %v", *sentRaw)
	}
	if !strings.Contains(lastErr, "502 Bad Gateway") {
		t.Fatalf("expected last_error to contain telegram error, got %q", lastErr)
	}
	expectedNext1 := runTime1.Add(initialBackoff)
	next1, _ := time.Parse(time.RFC3339Nano, nextRaw)
	if !next1.Equal(expectedNext1) {
		t.Fatalf("expected next_attempt_at=%v, got %v", expectedNext1, next1)
	}

	// At runTime1 + 50ms, the notification is NOT due yet
	claimedBeforeDue, err := store.ClaimDue(ctx, db, runTime1.Add(50*time.Millisecond), 3, 10)
	if err != nil {
		t.Fatalf("claim due: %v", err)
	}
	if len(claimedBeforeDue) != 0 {
		t.Fatalf("expected 0 claimed before due, got %d", len(claimedBeforeDue))
	}

	// Attempt 2: at expectedNext1 (due), should fail again, backoff = 200ms
	now2 := expectedNext1
	processed2, err := worker.RunOnce(ctx, now2)
	if err != nil {
		t.Fatalf("run once attempt 2: %v", err)
	}
	if processed2 != 1 {
		t.Fatalf("expected 1 processed on attempt 2, got %d", processed2)
	}
	if sender.callCount != 2 {
		t.Fatalf("expected 2 sender calls, got %d", sender.callCount)
	}

	if err := db.QueryRowContext(ctx, "SELECT attempts, next_attempt_at, sent_at, last_error FROM notification_outbox LIMIT 1").Scan(&attempts, &nextRaw, &sentRaw, &lastErr); err != nil {
		t.Fatalf("query record attempt 2: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected attempts=2, got %d", attempts)
	}
	expectedNext2 := now2.Add(2 * initialBackoff)
	next2, _ := time.Parse(time.RFC3339Nano, nextRaw)
	if !next2.Equal(expectedNext2) {
		t.Fatalf("expected next_attempt_at=%v, got %v", expectedNext2, next2)
	}

	// Attempt 3: at now2 + 200ms (due), reaches maxAttempts (3) -> MUST DEAD-LETTER
	now3 := expectedNext2
	processed3, err := worker.RunOnce(ctx, now3)
	if err != nil {
		t.Fatalf("run once attempt 3: %v", err)
	}
	if processed3 != 1 {
		t.Fatalf("expected 1 processed on attempt 3, got %d", processed3)
	}
	if sender.callCount != 3 {
		t.Fatalf("expected 3 sender calls, got %d", sender.callCount)
	}

	if err := db.QueryRowContext(ctx, "SELECT attempts, sent_at, last_error FROM notification_outbox LIMIT 1").Scan(&attempts, &sentRaw, &lastErr); err != nil {
		t.Fatalf("query record attempt 3: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected attempts=3, got %d", attempts)
	}
	if sentRaw != nil {
		t.Fatalf("expected sent_at to remain nil on dead letter, got %v", *sentRaw)
	}
	if !strings.HasPrefix(lastErr, "dead_letter:") {
		t.Fatalf("expected last_error to be marked as dead_letter, got %q", lastErr)
	}

	// Dead-lettered records must not be claimed anymore
	claimedAfterDead, err := store.ClaimDue(ctx, db, now3.Add(time.Hour), 3, 10)
	if err != nil {
		t.Fatalf("claim due after dead-letter: %v", err)
	}
	if len(claimedAfterDead) != 0 {
		t.Fatalf("expected 0 claimable records after dead letter, got %d", len(claimedAfterDead))
	}
}

func TestOutboxWorkerSuccessfulDelivery(t *testing.T) {
	ctx := context.Background()
	db, store := setupOutboxTestDB(t)

	msg := notify.Message{Title: "Success", Body: "Delivered cleanly"}
	if err := store.Enqueue(ctx, db, "telegram", msg); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	sender := &mockFailingSender{err: nil}
	worker := notify.Worker{
		DB:                  db,
		Store:               store,
		Sender:              sender,
		MaxDeliveryAttempts: 3,
	}

	now := time.Now().UTC()
	processed, err := worker.RunOnce(ctx, now)
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 processed, got %d", processed)
	}

	var sentRaw *string
	if err := db.QueryRowContext(ctx, "SELECT sent_at FROM notification_outbox LIMIT 1").Scan(&sentRaw); err != nil {
		t.Fatalf("query sent_at: %v", err)
	}
	if sentRaw == nil {
		t.Fatal("expected sent_at to be populated after success")
	}
}
