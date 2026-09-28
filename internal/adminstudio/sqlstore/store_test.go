package sqlstore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestNotificationInboxIsUserScopedAndMutable(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "admin.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := Store{Dialect: database.SQLite}
	if err := store.Add(ctx, db, "user-a", "Page published", "Your page is live.", "/admin/pages"); err != nil {
		t.Fatal(err)
	}
	count, err := store.UnreadCount(ctx, db, "user-a")
	if err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	items, err := store.List(ctx, db, "user-a", 10)
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%v err=%v", items, err)
	}
	if err := store.MarkAllRead(ctx, db, "user-b"); err != nil {
		t.Fatal(err)
	}
	count, _ = store.UnreadCount(ctx, db, "user-a")
	if count != 1 {
		t.Fatalf("other user's mark-read changed count to %d", count)
	}
	if err := store.Dismiss(ctx, db, "user-a", items[0].ID); err != nil {
		t.Fatal(err)
	}
	items, err = store.List(ctx, db, "user-a", 10)
	if err != nil || len(items) != 0 {
		t.Fatalf("dismissed items=%v err=%v", items, err)
	}
}

func TestNotificationRejectsUnsafeTarget(t *testing.T) {
	store := Store{Dialect: database.SQLite}
	if err := store.Add(context.Background(), nil, "u", "t", "m", "//outside.example"); err == nil {
		t.Fatal("unsafe target accepted")
	}
}
