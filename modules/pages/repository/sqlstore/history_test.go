package sqlstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func TestListRevisionsResolvesEditorWithoutReturningTechnicalIDToView(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "history.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO users (id,email,password_hash,role,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?)", "editor-internal-id", "editor@example.test", "unused", "editor", 1, time.Now().UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	store := Store{Dialect: database.SQLite}
	writerCtx := auth.WithUser(ctx, &auth.User{ID: "editor-internal-id", Role: auth.RoleEditor, Active: true})
	page := pages.Page{ID: "page-history-1", Locale: "en", Slug: "about", Title: "About", Body: "Before", Status: pages.StatusDraft}
	if err := store.CreateDraftContent(writerCtx, db, page); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListRevisions(ctx, db, page.ID, 10)
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if items[0].ActorID != "editor-internal-id" || items[0].ActorEmail != "editor@example.test" {
		t.Fatalf("actor attribution=%+v", items[0])
	}
}
