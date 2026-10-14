package sqlstore

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

// Single-tenant sites run without a tenant scope; tags must still be stored
// (tenant_id is NOT NULL with an empty-string default).
func TestTagsAreStoredWithoutTenantScope(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "page-tags.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{"pages": true}); err != nil {
		t.Fatal(err)
	}
	store := Store{Dialect: database.SQLite}
	page := pages.Page{ID: "tagged", Locale: "en", Slug: "tagged", Title: "Tagged", Body: "Body", Tags: []string{"Go", " hydro "}}
	if err := database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
		if err := store.CreateDraftContent(ctx, tx, page); err != nil {
			return err
		}
		return store.Publish(ctx, tx, page.ID, 1)
	}); err != nil {
		t.Fatalf("single-tenant page with tags: %v", err)
	}
	draft, err := store.GetContent(ctx, db, page.ID)
	if err != nil {
		t.Fatal(err)
	}
	published, err := store.GetPublishedByRoute(ctx, db, "en", "tagged")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(draft.Tags, ",") != "go,hydro" || strings.Join(published.Tags, ",") != "go,hydro" {
		t.Fatalf("draft tags=%v published tags=%v", draft.Tags, published.Tags)
	}
}
