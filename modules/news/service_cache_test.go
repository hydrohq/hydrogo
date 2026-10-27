package news_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/news"
	newsstore "github.com/ardaozyildirim/hydrogo/modules/news/repository/sqlstore"
)

// countingPublicRepository wraps the real sqlstore so the test can tell how
// many times GetPublishedByRoute actually reached the database, instead of
// being answered by Service.PublicCache.
type countingPublicRepository struct {
	newsstore.Store
	lookups int
}

func (r *countingPublicRepository) GetPublishedByRoute(ctx context.Context, db database.Executor, locale, slug string) (news.Item, error) {
	r.lookups++
	return r.Store.GetPublishedByRoute(ctx, db, locale, slug)
}

func TestGetPublishedByRouteCachesUntilPublishInvalidates(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "news-cache.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	repo := &countingPublicRepository{Store: newsstore.Store{Dialect: database.SQLite}}
	item := news.Item{ID: "item-1", Locale: "en", Slug: "home", Title: "Launch", Body: "{}", Status: news.StatusDraft}
	if err := repo.Store.CreateDraft(ctx, db, item); err != nil {
		t.Fatal(err)
	}
	service := news.Service{Transactions: database.NewTransactor(db), Repository: repo, PublicCache: cache.New[news.Item](4, time.Minute)}

	if err := service.Publish(ctx, item.ID, 1); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		got, err := service.GetPublishedByRoute(ctx, db, "en", "home")
		if err != nil || got.Title != "Launch" {
			t.Fatalf("lookup %d = %+v/%v", i, got, err)
		}
	}
	if repo.lookups != 1 {
		t.Fatalf("repository lookups = %d, want 1 (cached after the first miss)", repo.lookups)
	}

	// Retract invalidates the cache; a stale cached entry would otherwise
	// keep answering with the now-retracted item.
	if err := service.Retract(ctx, item.ID, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetPublishedByRoute(ctx, db, "en", "home"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("lookup after retract = %v, want sql.ErrNoRows", err)
	}
	if repo.lookups != 2 {
		t.Fatalf("repository lookups after retract = %d, want 2 (cache was cleared)", repo.lookups)
	}
}
