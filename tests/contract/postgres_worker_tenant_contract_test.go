package contract_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/internal/notify"
	outboxstore "github.com/hydrohq/hydrogo/internal/outbox/sqlstore"
	"github.com/hydrohq/hydrogo/internal/tenant"
	"github.com/hydrohq/hydrogo/internal/webhooks"
	"github.com/hydrohq/hydrogo/migrations"
)

type recordingSender struct {
	mu     sync.Mutex
	titles []string
}

func (r *recordingSender) Send(_ context.Context, msg notify.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.titles = append(r.titles, msg.Title)
	return nil
}

// Background workers hold a raw *sql.DB. Under the fail-closed RLS policies
// that only sees tenant_id = ” unless each store call binds its tenant, so
// a shared database delivered nothing for named tenants. Both outbox workers
// must deliver every tenant's events.
func TestPostgresWorkersDeliverEveryTenant(t *testing.T) {
	ctx := context.Background()
	db := openIsolatedPostgres(t, ctx)
	var bypassesRLS bool
	if err := db.QueryRowContext(ctx, `SELECT rolsuper OR rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&bypassesRLS); err != nil {
		t.Fatal(err)
	}
	if bypassesRLS {
		t.Skip("current PostgreSQL role bypasses row-level security")
	}
	if err := database.ApplyMigrations(ctx, db, database.Postgres, migrations.FS); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	alpha, err := tenant.NewScope("tenant-alpha", "org-alpha")
	if err != nil {
		t.Fatal(err)
	}
	beta, err := tenant.NewScope("tenant-beta", "org-beta")
	if err != nil {
		t.Fatal(err)
	}
	scopes := []tenant.Scope{alpha, beta}

	store := outboxstore.Store{Dialect: database.Postgres, RequireTenantScope: true}
	notes := notify.OutboxStore{Dialect: database.Postgres, RequireTenantScope: true}
	bound := database.NewTenantTransactor(db, database.Postgres, true)
	for _, scope := range scopes {
		scopedCtx := tenant.WithScope(ctx, scope)
		// Enqueue the way request handlers do: inside their bound transaction.
		if err := bound.Within(scopedCtx, func(tx database.Executor) error {
			if err := store.Enqueue(scopedCtx, tx, "event-"+scope.TenantID, "order.created", "order-"+scope.TenantID, 1); err != nil {
				return err
			}
			return notes.Enqueue(scopedCtx, tx, "telegram", notify.Message{Title: scope.TenantID})
		}); err != nil {
			t.Fatalf("enqueue for %s: %v", scope.TenantID, err)
		}
	}
	pending := func() (total int) {
		for _, scope := range scopes {
			var n int
			if err := bound.Within(tenant.WithScope(ctx, scope), func(exec database.Executor) error {
				return exec.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_events`).Scan(&n)
			}); err != nil {
				t.Fatal(err)
			}
			total += n
		}
		return total
	}

	var mu sync.Mutex
	var delivered []string
	runCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event webhooks.Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		delivered = append(delivered, event.ID)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	worker := webhooks.Worker{
		DB:           db,
		Store:        store,
		TenantScopes: scopes,
		Dispatcher: webhooks.Dispatcher{Endpoints: []webhooks.Endpoint{{
			ID: "sink", URL: server.URL, Secret: "contract-test-secret", AllowInternal: true,
			Topics: map[string]struct{}{"order.created": {}},
		}}},
		WorkerID:     "outbox-worker-contract",
		Lease:        time.Minute,
		PollInterval: 10 * time.Millisecond,
		OnError: func(err error) {
			if runCtx.Err() == nil {
				t.Errorf("outbox worker: %v", err)
			}
		},
	}
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		_ = worker.Run(runCtx)
	}()
	for pending() > 0 && runCtx.Err() == nil {
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	<-stopped
	if left := pending(); left != 0 {
		t.Fatalf("outbox still holds %d events; want every tenant's event acknowledged", left)
	}
	mu.Lock()
	defer mu.Unlock()
	sort.Strings(delivered)
	if len(delivered) != 2 || delivered[0] != "event-tenant-alpha" || delivered[1] != "event-tenant-beta" {
		t.Fatalf("webhook worker delivered %v; want both tenants' events", delivered)
	}
	sender := &recordingSender{}
	notifyWorker := notify.Worker{DB: db, Store: notes, Sender: sender, TenantScopes: scopes}
	if n, err := notifyWorker.RunOnce(ctx, time.Now().UTC().Add(time.Second)); err != nil || n != 2 {
		t.Fatalf("notification worker processed %d (err %v); want 2", n, err)
	}
	sort.Strings(sender.titles)
	if len(sender.titles) != 2 || sender.titles[0] != "tenant-alpha" || sender.titles[1] != "tenant-beta" {
		t.Fatalf("notification worker sent %v; want both tenants", sender.titles)
	}
	if n, err := notifyWorker.RunOnce(ctx, time.Now().UTC().Add(time.Second)); err != nil || n != 0 {
		t.Fatalf("second notification pass processed %d (err %v); want 0 after MarkSent", n, err)
	}
}
