package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func newSpool(t *testing.T, scoped bool) (Spool, database.Transactor) {
	t.Helper()
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "spool.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	tx := database.NewTenantTransactor(db, database.SQLite, scoped)
	return Spool{DB: db, Tx: tx, Dialect: database.SQLite, RequireTenantScope: scoped, MaxPending: 4}, tx
}

func TestSpoolSQLiteEnqueueClaimDeliverAndList(t *testing.T) {
	store, tx := newSpool(t, false)
	ctx := context.Background()
	entry := site.EmailNotification{EventKey: "submission-1:alert:team@example.com", To: "team@example.com", Subject: "Neue Anfrage", Body: "Hello"}
	if err := tx.Within(ctx, func(executor database.Executor) error { return store.Enqueue(ctx, executor, entry) }); err != nil {
		t.Fatal(err)
	}
	if err := tx.Within(ctx, func(executor database.Executor) error { return store.Enqueue(ctx, executor, entry) }); err != nil {
		t.Fatalf("idempotent enqueue: %v", err)
	}
	items, err := store.List(ctx, mailer.SpoolPending, 10)
	if err != nil || len(items) != 1 || items[0].Message.To != entry.To {
		t.Fatalf("pending entries=%#v err=%v", items, err)
	}
	now := time.Now().UTC()
	claimed, token, ok, err := store.Claim(ctx, "worker-1", now, time.Minute)
	if err != nil || !ok || claimed.Attempts != 1 || token == "" {
		t.Fatalf("claim=%#v token=%q ok=%v err=%v", claimed, token, ok, err)
	}
	if _, _, ok, err := store.Claim(ctx, "worker-2", now, time.Minute); err != nil || ok {
		t.Fatalf("leased mail was claimed twice: ok=%v err=%v", ok, err)
	}
	if err := store.Delivered(ctx, claimed.ID, "worker-1", token, now); err != nil {
		t.Fatal(err)
	}
	items, err = store.List(ctx, mailer.SpoolDelivered, 10)
	if err != nil || len(items) != 1 || !items[0].DeliveredAt.Equal(now) {
		t.Fatalf("delivered entries=%#v err=%v", items, err)
	}
}

func TestSpoolSQLiteRetriesAndFencesExpiredWorkers(t *testing.T) {
	store, tx := newSpool(t, false)
	ctx := context.Background()
	entry := site.EmailNotification{EventKey: "submission-2:alert", To: "team@example.com", Subject: "Alert", Body: "Hello"}
	if err := tx.Within(ctx, func(executor database.Executor) error { return store.Enqueue(ctx, executor, entry) }); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	first, token1, ok, err := store.Claim(ctx, "worker-1", now, time.Minute)
	if err != nil || !ok {
		t.Fatalf("first claim: %v/%v", ok, err)
	}
	if err := store.Retry(ctx, first.ID, "worker-1", token1, now.Add(5*time.Minute), "offline"); err != nil {
		t.Fatal(err)
	}
	if _, _, ok, err := store.Claim(ctx, "worker-2", now.Add(4*time.Minute), time.Minute); err != nil || ok {
		t.Fatalf("early retry claim: %v/%v", ok, err)
	}
	second, token2, ok, err := store.Claim(ctx, "worker-2", now.Add(5*time.Minute), time.Minute)
	if err != nil || !ok || second.Attempts != 2 {
		t.Fatalf("second claim=%#v ok=%v err=%v", second, ok, err)
	}
	if err := store.FailPermanent(ctx, second.ID, "worker-2", token2, "rejected"); err != nil {
		t.Fatal(err)
	}
	if err := store.Delivered(ctx, second.ID, "worker-1", token1, now); !errors.Is(err, mailer.ErrSpoolLeaseInvalid) {
		t.Fatalf("stale worker update = %v", err)
	}
	if err := store.RetryNow(ctx, second.ID); err != nil {
		t.Fatal(err)
	}
	items, err := store.List(ctx, mailer.SpoolPending, 10)
	if err != nil || len(items) != 1 || items[0].Attempts != 0 {
		t.Fatalf("manual retry entries=%#v err=%v", items, err)
	}
}

func TestSpoolTenantScopesSameEventKeys(t *testing.T) {
	store, tx := newSpool(t, true)
	ctx := context.Background()
	alpha, _ := tenant.NewScope("tenant-a", "alpha")
	beta, _ := tenant.NewScope("tenant-b", "beta")
	for _, item := range []struct {
		scope tenant.Scope
		body  string
	}{{alpha, "A"}, {beta, "B"}} {
		jobCtx := tenant.WithScope(ctx, item.scope)
		message := site.EmailNotification{EventKey: "same-event", To: "team@example.com", Subject: "Alert", Body: item.body}
		if err := tx.Within(jobCtx, func(executor database.Executor) error { return store.Enqueue(jobCtx, executor, message) }); err != nil {
			t.Fatal(err)
		}
	}
	items, err := store.List(tenant.WithScope(ctx, alpha), "", 10)
	if err != nil || len(items) != 1 || items[0].Message.Body != "A" {
		t.Fatalf("alpha spool entries=%#v err=%v", items, err)
	}
	items, err = store.List(tenant.WithScope(ctx, beta), "", 10)
	if err != nil || len(items) != 1 || items[0].Message.Body != "B" {
		t.Fatalf("beta spool entries=%#v err=%v", items, err)
	}
}
