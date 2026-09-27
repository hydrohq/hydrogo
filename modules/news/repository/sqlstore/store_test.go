package sqlstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

func TestListRejectsUnboundedLimitsBeforeDatabaseAccess(t *testing.T) {
	store := Store{}
	for _, limit := range []int{0, -1, 101, 1000} {
		if _, err := store.ListPublished(context.Background(), nil, "tr", limit); err != ErrInvalidListLimit {
			t.Fatalf("published limit %d error = %v", limit, err)
		}
		if _, err := store.ListAdmin(context.Background(), nil, limit); err != ErrInvalidListLimit {
			t.Fatalf("admin limit %d error = %v", limit, err)
		}
	}
}

func TestStreamPublishedRoutesExcludesDraftsAndUsesBoundedProjection(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "news-routes.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		id, slug, status string
		published        any
	}{
		{"article-live", "live", "published", 1},
		{"article-draft", "draft", "draft", nil},
	} {
		_, err := db.ExecContext(ctx, `INSERT INTO news_items (id,locale,slug,title,body,status,version,revision,published_revision,schedule_generation,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, row.id, "en", row.slug, row.slug, "private body", row.status, 1, 1, row.published, 0, time.Now().UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			t.Fatal(err)
		}
	}
	var got []news.Item
	store := Store{Dialect: database.SQLite}
	if err := store.StreamPublishedRoutes(ctx, db, func(item news.Item) error { got = append(got, item); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "article-live" || got[0].Body != "" || got[0].Slug != "live" {
		t.Fatalf("streamed route projection = %#v", got)
	}
}
