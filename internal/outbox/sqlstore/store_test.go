package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/outbox"
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

var _ database.Executor = (*sql.DB)(nil)
