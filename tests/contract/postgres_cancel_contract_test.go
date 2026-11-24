package contract_test

import (
	"context"
	"testing"
	"time"
)

// A cancelled request context must stop the query on the server, not only on
// the client socket, so a dropped client does not leave Postgres working.
func TestPostgresContextCancelStopsServerQuery(t *testing.T) {
	ctx := context.Background()
	db := openIsolatedPostgres(t, ctx)
	const marker = "hydro_cancel_probe_7f3a"

	queryCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	_, err := db.ExecContext(queryCtx, "SELECT pg_sleep(30) /* "+marker+" */")
	if err == nil {
		t.Fatal("pg_sleep(30) returned without error under a 300ms context")
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		var running int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity
			WHERE state = 'active' AND query LIKE '%' || $1 || '%' AND pid <> pg_backend_pid()`, marker).Scan(&running); err != nil {
			t.Fatal(err)
		}
		if running == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("cancelled query still running on the server after 5s (%d backends)", running)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
