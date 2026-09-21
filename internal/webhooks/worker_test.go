package webhooks

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/outbox"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type fakeOutbox struct {
	event                                 outbox.Event
	claimed, acked, retried, deadLettered bool
	scopeIDs                              []string
}

func (f *fakeOutbox) Claim(ctx context.Context, _ database.Executor, _ string, _ time.Time, _ time.Duration) (outbox.Event, string, bool, error) {
	f.claimed = true
	if scope, err := tenant.Require(ctx); err == nil {
		f.scopeIDs = append(f.scopeIDs, scope.TenantID)
	}
	return f.event, "token", true, nil
}
func (f *fakeOutbox) Ack(context.Context, database.Executor, string, string, string) error {
	f.acked = true
	return nil
}
func (f *fakeOutbox) Retry(context.Context, database.Executor, string, string, string, time.Time, string) error {
	f.retried = true
	return nil
}
func (f *fakeOutbox) DeadLetter(context.Context, database.Executor, outbox.Event, string, string, string) error {
	f.deadLettered = true
	return nil
}

func TestWorkerDeadLettersAfterBoundedDeliveryAttempts(t *testing.T) {
	store := &fakeOutbox{event: outbox.Event{ID: "event-dead", Topic: "order.created", AggregateID: "order-1", PayloadVersion: 1, Attempts: 3}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "permanent", http.StatusBadGateway) }))
	defer server.Close()
	worker := Worker{DB: structExecutor{}, Store: store, WorkerID: "w1", Lease: time.Minute, MaxDeliveryAttempts: 3, Dispatcher: Dispatcher{Endpoints: []Endpoint{{ID: "erp", URL: server.URL, Secret: "secret", Topics: map[string]struct{}{"order.created": {}}}}, MaxAttempts: 1}}
	processed, err := worker.RunOnce(context.Background(), time.Now())
	if err == nil || !processed || !store.deadLettered || store.retried {
		t.Fatalf("dead letter state processed=%v err=%v store=%+v", processed, err, store)
	}
}

func TestWorkerAcknowledgesOnlyAfterDelivery(t *testing.T) {
	store := &fakeOutbox{event: outbox.Event{ID: "event-1", Topic: "order.created", AggregateID: "order-1", PayloadVersion: 1}}
	worker := Worker{DB: structExecutor{}, Store: store, WorkerID: "w1", Lease: time.Minute, Dispatcher: Dispatcher{Endpoints: nil}}
	processed, err := worker.RunOnce(context.Background(), time.Now())
	if err != nil || !processed || !store.claimed || !store.acked || store.retried {
		t.Fatalf("worker state processed=%v err=%v store=%+v", processed, err, store)
	}
}

func TestWorkerRunsEachTenantScope(t *testing.T) {
	store := &fakeOutbox{event: outbox.Event{ID: "event-tenant", Topic: "order.created", AggregateID: "order-1", PayloadVersion: 1}}
	alpha, _ := tenant.NewScope("tenant-a", "alpha")
	beta, _ := tenant.NewScope("tenant-b", "beta")
	worker := Worker{DB: structExecutor{}, Store: store, WorkerID: "w1", Lease: time.Minute, TenantScopes: []tenant.Scope{alpha, beta}, Dispatcher: Dispatcher{Endpoints: nil}}
	processed, err := worker.runScopedOnce(context.Background(), time.Now())
	if err != nil || !processed || !store.claimed || !store.acked || len(store.scopeIDs) != 2 || store.scopeIDs[0] != "tenant-a" || store.scopeIDs[1] != "tenant-b" {
		t.Fatalf("scoped worker state processed=%v err=%v store=%+v", processed, err, store)
	}
}

type structExecutor struct{}

func (structExecutor) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, nil
}
func (structExecutor) QueryRowContext(context.Context, string, ...any) *sql.Row { return nil }
