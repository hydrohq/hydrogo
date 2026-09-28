package contract_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/push"
	pushstore "github.com/ardaozyildirim/hydrogo/internal/push/sqlstore"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLitePushSubscriptionContract(t *testing.T) {
	runPushContract(t, database.SQLite, filepath.Join(t.TempDir(), "push.sqlite"))
}
func TestPostgresPushSubscriptionContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runPushContract(t, database.Postgres, dsn)
}
func runPushContract(t *testing.T, dialect database.Dialect, dsn string) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, dialect, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplyMigrationsForModules(ctx, db, dialect, migrations.FS, map[string]bool{"push": true}); err != nil {
		t.Fatal(err)
	}
	repository := pushstore.Store{DB: db, Dialect: dialect}
	dispatcher := push.Dispatcher{Repository: repository}
	userID := fmt.Sprintf("push_%d", time.Now().UnixNano())
	token := fmt.Sprintf("fcm_token_%d", time.Now().UnixNano())
	if err := dispatcher.Subscribe(ctx, push.Subscription{UserID: userID, Platform: push.PlatformMobile, Endpoint: token}); err != nil {
		t.Fatal(err)
	}
	items, err := repository.List(ctx, userID, push.PlatformMobile)
	if err != nil || len(items) != 1 || items[0].Endpoint != token {
		t.Fatalf("subscriptions=%+v err=%v", items, err)
	}
	if err := dispatcher.Unsubscribe(ctx, userID, push.PlatformMobile, token); err != nil {
		t.Fatal(err)
	}
	items, err = repository.List(ctx, userID, push.PlatformMobile)
	if err != nil || len(items) != 0 {
		t.Fatalf("after unsubscribe=%+v err=%v", items, err)
	}
}
