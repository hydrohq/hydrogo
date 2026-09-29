package site_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	"github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
)

type formMetricsCounter struct {
	calls int
	err   error
}

func (m *formMetricsCounter) RecordFormSubmission(context.Context, database.Executor, string, string, time.Time) error {
	if m.err != nil {
		return m.err
	}
	m.calls++
	return nil
}

func TestSubmitIdempotentReplaysAndRejectsChangedPayload(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	metrics := &formMetricsCounter{}
	service := site.Service{Transactions: database.NewTransactor(db), Store: sqlstore.Store{Dialect: database.SQLite}, FormMetrics: metrics}
	value := site.Submission{ID: "form-1", FormKey: "contact", Email: "user@example.com", Payload: `{"message":"hello"}`, SourceIPHash: "hash", IdempotencyKey: "0123456789abcdef", RequestHash: "request-1"}
	if replayed, err := service.SubmitIdempotent(context.Background(), value); err != nil || replayed {
		t.Fatalf("first submit replayed=%v err=%v", replayed, err)
	}
	if metrics.calls != 1 {
		t.Fatalf("first submission metrics = %d, want 1", metrics.calls)
	}
	value.ID = "form-2"
	if replayed, err := service.SubmitIdempotent(context.Background(), value); err != nil || !replayed {
		t.Fatalf("retry replayed=%v err=%v", replayed, err)
	}
	if metrics.calls != 1 {
		t.Fatalf("replayed submission metrics = %d, want 1", metrics.calls)
	}
	value.RequestHash = "request-2"
	if _, err := service.SubmitIdempotent(context.Background(), value); err != site.ErrIdempotencyConflict {
		t.Fatalf("changed payload error=%v", err)
	}
	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM form_inbox WHERE form_key = 'contact'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("stored submissions = %d, want 1", count)
	}
}

func TestSubmitRollsBackIfTransactionalFormMetricFails(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site-metric-rollback.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	metrics := &formMetricsCounter{err: context.DeadlineExceeded}
	service := site.Service{Transactions: database.NewTransactor(db), Store: sqlstore.Store{Dialect: database.SQLite}, FormMetrics: metrics}
	value := site.Submission{ID: "form-rollback", FormKey: "contact", Email: "user@example.com", Payload: `{"message":"hello"}`, SourceIPHash: "hash", IdempotencyKey: "0123456789abcdef", RequestHash: "request-1"}
	if _, err := service.SubmitIdempotent(context.Background(), value); err != context.DeadlineExceeded {
		t.Fatalf("submit error=%v", err)
	}
	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM form_inbox WHERE form_key = 'contact'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("failed metric left %d submissions committed", count)
	}
}
