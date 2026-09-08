package contract_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
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
	service := news.Service{Transactions: database.NewTransactor(db), Repository: store}
	item := news.Item{ID: "news-1", Locale: "en", Slug: "launch", Title: "Launch", Body: "body", Status: news.StatusDraft}
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
	if err := service.RunScheduled(ctx, item.ID, 1, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	published, err := store.GetPublishedByRoute(ctx, db, "en", "launch")
	if err != nil || published.Status != news.StatusPublished || published.PublishedRevision == nil {
		t.Fatalf("published = %#v/%v", published, err)
	}
	if err := service.Retract(ctx, item.ID, published.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetPublishedByRoute(ctx, db, "en", "launch"); err == nil {
		t.Fatal("retracted news remained public")
	}
}
