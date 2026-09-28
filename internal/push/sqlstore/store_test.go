package sqlstore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/push"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSubscriptionTenantOwnerAndPlatformScope(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "push.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := Store{DB: db, Dialect: database.SQLite}
	sub := push.Subscription{ID: "web-id", UserID: "admin", Platform: push.PlatformWeb, Endpoint: "https://push.example.test/e", Keys: []byte(`{"endpoint":"https://push.example.test/e","keys":{"auth":"secret","p256dh":"public"}}`)}
	if err := store.Save(ctx, sub); err != nil {
		t.Fatal(err)
	}
	items, err := store.List(ctx, "", push.PlatformWeb)
	if err != nil || len(items) != 1 || items[0].ID != sub.ID {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	mobile, err := store.List(ctx, "", push.PlatformMobile)
	if err != nil || len(mobile) != 0 {
		t.Fatalf("mobile=%+v err=%v", mobile, err)
	}
	if err := store.Delete(ctx, "web-id", "other-user", push.PlatformWeb, sub.Endpoint); err != nil {
		t.Fatal(err)
	}
	items, err = store.List(ctx, "admin", push.PlatformWeb)
	if err != nil || len(items) != 1 {
		t.Fatalf("wrong owner deleted subscription: %v %v", items, err)
	}
	if err := store.Disable(ctx, "web-id"); err != nil {
		t.Fatal(err)
	}
	items, err = store.List(ctx, "", push.PlatformWeb)
	if err != nil || len(items) != 0 {
		t.Fatalf("disabled subscriptions=%+v err=%v", items, err)
	}
}
