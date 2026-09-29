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

func TestExpiredNewsIsHiddenPubliclyAndRetainedForAdmin(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "news-expiry.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, row := range []struct {
		id, slug string
		expires  any
	}{
		{id: "news-expired", slug: "expired", expires: now.Add(-time.Minute).Format(time.RFC3339Nano)},
		{id: "news-active", slug: "active", expires: now.Add(time.Hour).Format(time.RFC3339Nano)},
		{id: "news-evergreen", slug: "evergreen", expires: nil},
	} {
		_, err := db.ExecContext(ctx, `INSERT INTO news_items (id,locale,slug,title,body,status,version,revision,published_revision,schedule_generation,expires_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`, row.id, "en", row.slug, row.slug, "body", "published", 1, 1, 1, 0, row.expires, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
		if err != nil {
			t.Fatal(err)
		}
	}
	store := Store{Dialect: database.SQLite}
	for _, slug := range []string{"expired"} {
		if _, err := store.GetPublishedByRoute(ctx, db, "en", slug); err == nil {
			t.Fatalf("expired item %q was returned by public route", slug)
		}
	}
	items, err := store.ListPublished(ctx, db, "en", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Slug == "expired" || items[1].Slug == "expired" {
		t.Fatalf("public list includes expired news: %#v", items)
	}
	adminItems, err := store.ListAdmin(ctx, db, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(adminItems) != 3 {
		t.Fatalf("admin list did not retain expired news: %#v", adminItems)
	}
}

func TestStreamPublishedRoutesExcludesExpiredItems(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "news-routes-expiry.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, row := range []struct {
		id, slug string
		expires  any
	}{
		{id: "route-expired", slug: "expired", expires: now.Add(-time.Minute).Format(time.RFC3339Nano)},
		{id: "route-active", slug: "active", expires: nil},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO news_items (id,locale,slug,title,body,status,version,revision,published_revision,schedule_generation,expires_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`, row.id, "en", row.slug, row.slug, row.slug, "published", 1, 1, 1, 0, row.expires, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
			t.Fatal(err)
		}
	}
	var got []news.Item
	store := Store{Dialect: database.SQLite}
	if err := store.StreamPublishedRoutes(ctx, db, func(item news.Item) error { got = append(got, item); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "route-active" {
		t.Fatalf("route stream included an expired item: %#v", got)
	}
}
