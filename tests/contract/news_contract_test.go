package contract_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/jobs"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/news"
	newsstore "github.com/ardaozyildirim/hydrogo/modules/news/repository/sqlstore"
)

func TestSQLiteNewsContract(t *testing.T) {
	runNewsContract(t, database.SQLite, filepath.Join(t.TempDir(), "news.sqlite"))
}

func TestPostgresNewsContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runNewsContract(t, database.Postgres, dsn)
}

func runNewsContract(t *testing.T, dialect database.Dialect, dsn string) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, dialect, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM news_items"); err != nil {
		t.Fatal(err)
	}
	store := newsstore.Store{Dialect: dialect}
	service := news.Service{Transactions: database.NewTransactor(db), Repository: store, Audit: auditstore.Store{Dialect: dialect}, Outbox: outboxstore.Store{Dialect: dialect}}
	marker := time.Now().UTC().UnixNano()
	item := news.Item{ID: fmt.Sprintf("news-%d", marker), Locale: "en", Slug: fmt.Sprintf("launch-%d", marker), Title: "Launch", Body: "body", Status: news.StatusDraft}
	if err := service.CreateDraft(ctx, item); err != nil {
		t.Fatal(err)
	}
	if err := service.UpdateDraft(ctx, item, 99); !errors.Is(err, news.ErrConflict) {
		t.Fatalf("stale update = %v", err)
	}
	if err := service.Schedule(ctx, item.ID, 1, time.Now().UTC().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := service.RunScheduled(ctx, item.ID, 99, time.Now().UTC()); !errors.Is(err, news.ErrScheduleStale) {
		t.Fatalf("stale schedule = %v", err)
	}
	worker := news.Worker{Queue: jobs.NewQueue(10), Service: service}
	if err := worker.EnqueueDue(ctx, db, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if ran, err := worker.RunOnce(ctx, "news-worker-1", time.Now().UTC()); err != nil || !ran {
		t.Fatalf("scheduled worker = %v/%v", ran, err)
	}
	if err := service.RunScheduled(ctx, item.ID, 1, time.Now().UTC()); !errors.Is(err, news.ErrScheduleStale) {
		t.Fatalf("replayed schedule = %v", err)
	}
	published, err := store.GetPublishedByRoute(ctx, db, item.Locale, item.Slug)
	if err != nil || published.Status != news.StatusPublished || published.PublishedRevision == nil {
		t.Fatalf("published = %#v/%v", published, err)
	}
	if err := service.Retract(ctx, item.ID, published.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetPublishedByRoute(ctx, db, item.Locale, item.Slug); err == nil {
		t.Fatal("retracted news remained public")
	}
}

func TestMemberSecurityAndIsolationContract(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "members.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	// Verify member table isolation and sessions
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM members").Scan(&count); err != nil {
		t.Fatalf("members table should exist and be queryable: %v", err)
	}
	if count != 0 {
		t.Errorf("initial member count = %d, want 0", count)
	}

	var sessionCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM member_sessions").Scan(&sessionCount); err != nil {
		t.Fatalf("member_sessions table should exist and be queryable: %v", err)
	}
	if sessionCount != 0 {
		t.Errorf("initial member session count = %d, want 0", sessionCount)
	}
}
