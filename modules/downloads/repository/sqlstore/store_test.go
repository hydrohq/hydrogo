package sqlstore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/downloads"
)

func TestSQLiteDownloadCounterIsAtomicAndVisibleInListings(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "downloads.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{"downloads": true}); err != nil {
		t.Fatal(err)
	}
	store := New(database.SQLite)
	item := downloads.Download{Base: sectoral.Base{ID: "catalog", Locale: "en", Slug: "catalog", Title: "Catalog", Status: "published"}, FileURL: "/files/catalog.pdf", ContentType: "application/pdf", FileSize: 500, DownloadCount: 500}
	if err := store.Create(ctx, db, item); err != nil {
		t.Fatal(err)
	}
	for want := int64(1); want <= 3; want++ {
		got, err := store.IncrementDownload(ctx, db, item.ID)
		if err != nil || got != want {
			t.Fatalf("increment=%d err=%v; want %d", got, err, want)
		}
	}
	got, err := store.Get(ctx, db, item.ID)
	if err != nil || got.DownloadCount != 3 {
		t.Fatalf("Get() count=%d err=%v", got.DownloadCount, err)
	}
	items, err := store.ListPublished(ctx, db, "en", 10)
	if err != nil || len(items) != 1 || items[0].DownloadCount != 3 {
		t.Fatalf("ListPublished()=%+v err=%v", items, err)
	}
}
