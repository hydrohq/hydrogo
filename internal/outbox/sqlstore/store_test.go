package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/outbox"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func TestSQLiteOutboxLeaseContract(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, "file:outbox-contract?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, topic TEXT NOT NULL, aggregate_id TEXT NOT NULL, payload_version INTEGER NOT NULL, available_at TEXT NOT NULL DEFAULT '1970-01-01T00:00:00Z', attempts INTEGER NOT NULL DEFAULT 0, lease_owner TEXT, lease_token TEXT, lease_until TEXT, last_error TEXT)`,
		`CREATE INDEX idx_outbox_claimable ON outbox_events(available_at, lease_until, id)`,
		`CREATE TABLE outbox_dead_letters (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL DEFAULT '', topic TEXT NOT NULL, aggregate_id TEXT NOT NULL, payload_version INTEGER NOT NULL, attempts INTEGER NOT NULL, last_error TEXT NOT NULL, failed_at TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	store := Store{Dialect: database.SQLite}
	ctx := context.Background()
	if err := store.Enqueue(ctx, db, "event-1", "order.created", "order-1", 1); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	event, token, claimed, err := store.Claim(ctx, db, "worker-1", now, time.Minute)
	if err != nil || !claimed || event.ID != "event-1" || token == "" || event.Attempts != 1 {
		t.Fatalf("claim = %#v %q %v %v", event, token, claimed, err)
	}
	if _, _, claimed, err := store.Claim(ctx, db, "worker-2", now, time.Minute); err != nil || claimed {
		t.Fatalf("leased event claimed twice: %v %v", claimed, err)
	}
	if err := store.Retry(ctx, db, event.ID, "worker-1", token, now.Add(time.Minute), "erp unavailable"); err != nil {
		t.Fatal(err)
	}
	if err := store.Ack(ctx, db, event.ID, "worker-1", token); !errors.Is(err, outbox.ErrLeaseInvalid) {
		t.Fatalf("stale ack = %v", err)
	}
	event, token, claimed, err = store.Claim(ctx, db, "worker-2", now.Add(2*time.Minute), time.Minute)
	if err != nil || !claimed || event.Attempts != 2 {
		t.Fatalf("retry claim = %#v %q %v %v", event, token, claimed, err)
	}
	if err := store.Ack(ctx, db, event.ID, "worker-2", token); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM outbox_events`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("acknowledged events = %d", count)
	}
}

func TestSQLiteOutboxDeadLetterRequiresLeaseAndPersistsFailure(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, "file:outbox-dead-letter-contract?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, topic TEXT NOT NULL, aggregate_id TEXT NOT NULL, payload_version INTEGER NOT NULL, available_at TEXT NOT NULL DEFAULT '1970-01-01T00:00:00Z', attempts INTEGER NOT NULL DEFAULT 0, lease_owner TEXT, lease_token TEXT, lease_until TEXT, last_error TEXT)`,
		`CREATE TABLE outbox_dead_letters (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL DEFAULT '', topic TEXT NOT NULL, aggregate_id TEXT NOT NULL, payload_version INTEGER NOT NULL, attempts INTEGER NOT NULL, last_error TEXT NOT NULL, failed_at TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	store := Store{Dialect: database.SQLite}
	ctx := context.Background()
	if err := store.Enqueue(ctx, db, "event-dl", "order.created", "order-1", 1); err != nil {
		t.Fatal(err)
	}
	event, token, ok, err := store.Claim(ctx, db, "worker", time.Now().UTC(), time.Minute)
	if err != nil || !ok {
		t.Fatalf("claim: %#v %q %v", event, token, err)
	}
	if err := store.DeadLetter(ctx, db, event, "worker", token, "endpoint permanently unavailable"); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM outbox_events`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("queue count=%d err=%v", count, err)
	}
	var topic, reason string
	if err := db.QueryRow(`SELECT topic,last_error FROM outbox_dead_letters WHERE id = 'event-dl'`).Scan(&topic, &reason); err != nil {
		t.Fatal(err)
	}
	if topic != "order.created" || reason == "" {
		t.Fatalf("dead letter = %q/%q", topic, reason)
	}
}

func TestSQLiteScopedOutboxLeaseCannotCrossTenant(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, "file:scoped-outbox-contract?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, topic TEXT NOT NULL, aggregate_id TEXT NOT NULL, payload_version INTEGER NOT NULL, available_at TEXT NOT NULL DEFAULT '1970-01-01T00:00:00Z', attempts INTEGER NOT NULL DEFAULT 0, lease_owner TEXT, lease_token TEXT, lease_until TEXT, last_error TEXT)`); err != nil {
		t.Fatal(err)
	}
	store := Store{Dialect: database.SQLite}
	alpha, err := tenant.NewScope("tenant-a", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	beta, err := tenant.NewScope("tenant-b", "beta")
	if err != nil {
		t.Fatal(err)
	}
	alphaCtx := tenant.WithScope(context.Background(), alpha)
	if err := store.EnqueueScoped(alphaCtx, db, "event-a", "order.created", "order-a", 1); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, _, claimed, err := store.ClaimScoped(tenant.WithScope(context.Background(), beta), db, "worker-b", now, time.Minute); err != nil || claimed {
		t.Fatalf("cross-tenant claim = claimed:%v err:%v", claimed, err)
	}
	event, token, claimed, err := store.ClaimScoped(alphaCtx, db, "worker-a", now, time.Minute)
	if err != nil || !claimed || event.TenantID != "tenant-a" || token == "" {
		t.Fatalf("tenant claim = %#v %q %v %v", event, token, claimed, err)
	}
	if err := store.AckScoped(tenant.WithScope(context.Background(), beta), db, event.ID, "worker-a", token); !errors.Is(err, outbox.ErrLeaseInvalid) {
		t.Fatalf("cross-tenant ack = %v", err)
	}
	if err := store.AckScoped(alphaCtx, db, event.ID, "worker-a", token); err != nil {
		t.Fatal(err)
	}
}

func TestScopedOutboxRequiresTenantScope(t *testing.T) {
	store := Store{Dialect: database.SQLite}
	if err := store.EnqueueScoped(context.Background(), nil, "id", "topic", "aggregate", 1); !errors.Is(err, tenant.ErrMissingScope) {
		t.Fatalf("missing scope enqueue = %v", err)
	}
}

func TestOutboxStoreAutomaticallyScopesWhenContextHasTenant(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, "file:auto-scoped-outbox?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, topic TEXT NOT NULL, aggregate_id TEXT NOT NULL, payload_version INTEGER NOT NULL, available_at TEXT NOT NULL DEFAULT '1970-01-01T00:00:00Z', attempts INTEGER NOT NULL DEFAULT 0, lease_owner TEXT, lease_token TEXT, lease_until TEXT, last_error TEXT)`); err != nil {
		t.Fatal(err)
	}
	scope, err := tenant.NewScope("tenant-a", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithScope(context.Background(), scope)
	store := Store{Dialect: database.SQLite, RequireTenantScope: true}
	if err := store.Enqueue(ctx, db, "event-a", "order.created", "order-a", 1); err != nil {
		t.Fatal(err)
	}
	event, token, claimed, err := store.Claim(ctx, db, "worker-a", time.Now().UTC(), time.Minute)
	if err != nil || !claimed || event.TenantID != "tenant-a" || token == "" {
		t.Fatalf("auto-scoped claim = %#v %q %v %v", event, token, claimed, err)
	}
	if err := store.Ack(ctx, db, event.ID, "worker-a", token); err != nil {
		t.Fatal(err)
	}
}

func TestOutboxStoreFailClosedModeRejectsMissingScope(t *testing.T) {
	store := Store{Dialect: database.SQLite, RequireTenantScope: true}
	if err := store.Enqueue(context.Background(), nil, "id", "topic", "aggregate", 1); !errors.Is(err, tenant.ErrMissingScope) {
		t.Fatalf("required scope enqueue = %v", err)
	}
}

var _ database.Executor = (*sql.DB)(nil)
