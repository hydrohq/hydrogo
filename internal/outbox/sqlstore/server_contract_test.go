package sqlstore

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/outbox"
	"github.com/ardaozyildirim/hydrogo/internal/testmysql"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestPostgresOutboxLeaseContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.Postgres, migrations.FS); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"outbox_events", "outbox_dead_letters"} {
		if _, err := db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			t.Fatal(err)
		}
	}
	runServerLeaseContract(t, db, database.Postgres)
}

func TestMySQLOutboxLeaseContract(t *testing.T) {
	db := testmysql.Open(t)
	testmysql.MigrateAll(t, db)
	runServerLeaseContract(t, db, database.MySQL)
}

// runServerLeaseContract is the lease contract the SQLite test checks, run on the
// real tables of a server database: claim, no double claim, retry later, stale
// ack refused, dead letter.
func runServerLeaseContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	store := Store{Dialect: dialect}
	ctx := context.Background()
	if err := store.Enqueue(ctx, db, "event-1", "content.created", "r1", 1); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	event, token, claimed, err := store.Claim(ctx, db, "worker-1", now, time.Minute)
	if err != nil || !claimed || event.ID != "event-1" || event.Attempts != 1 {
		t.Fatalf("claim = %#v %q %v %v", event, token, claimed, err)
	}
	if _, _, again, err := store.Claim(ctx, db, "worker-2", now, time.Minute); err != nil || again {
		t.Fatalf("a leased event was claimed twice: %v %v", again, err)
	}
	if err := store.Retry(ctx, db, event.ID, "worker-1", token, now.Add(time.Minute), "receiver down"); err != nil {
		t.Fatal(err)
	}
	if err := store.Ack(ctx, db, event.ID, "worker-1", token); !errors.Is(err, outbox.ErrLeaseInvalid) {
		t.Fatalf("stale ack = %v", err)
	}
	if _, _, early, err := store.Claim(ctx, db, "worker-2", now, time.Minute); err != nil || early {
		t.Fatalf("an event waiting for its retry time was claimed early: %v %v", early, err)
	}
	event, token, claimed, err = store.Claim(ctx, db, "worker-2", now.Add(2*time.Minute), time.Minute)
	if err != nil || !claimed || event.Attempts != 2 {
		t.Fatalf("retry claim = %#v %q %v %v", event, token, claimed, err)
	}
	if err := store.DeadLetter(ctx, db, event, "worker-2", token, "gave up"); err != nil {
		t.Fatal(err)
	}
	var left, dead int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM outbox_events WHERE id = 'event-1'").Scan(&left); err != nil || left != 0 {
		t.Fatalf("queue rows = %d %v", left, err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM outbox_dead_letters WHERE id = 'event-1'").Scan(&dead); err != nil || dead != 1 {
		t.Fatalf("dead letters = %d %v", dead, err)
	}
}
