package webhooks

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"strings"
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
	retryNext                             time.Time
	retryReason                           string
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
func (f *fakeOutbox) Retry(_ context.Context, _ database.Executor, _, _, _ string, retryNext time.Time, retryReason string) error {
	f.retried = true
	f.retryNext = retryNext
	f.retryReason = retryReason
	return nil
}
func (f *fakeOutbox) DeadLetter(context.Context, database.Executor, outbox.Event, string, string, string) error {
	f.deadLettered = true
	return nil
}

func TestWorkerDeadLettersAfterBoundedDeliveryAttempts(t *testing.T) {
	store := &fakeOutbox{event: outbox.Event{ID: "event-dead", Topic: "order.created", AggregateID: "order-1", PayloadVersion: 1, Attempts: 3}}
	client := &http.Client{Transport: webhookRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("permanent")), Request: r}, nil
	})}
	worker := Worker{DB: structExecutor{}, Store: store, WorkerID: "w1", Lease: time.Minute, MaxDeliveryAttempts: 3, Dispatcher: Dispatcher{Client: client, Endpoints: []Endpoint{{ID: "erp", URL: "https://erp.example.test/hook", Secret: "secret", Topics: map[string]struct{}{"order.created": {}}}}, MaxAttempts: 1}}
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

func TestWorkerRetriesWithExponentialBackoff(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	// Attempts = 3 -> delay should be 1m * 2^(3-1) = 4m
	store := &fakeOutbox{event: outbox.Event{ID: "event-retry", Topic: "order.created", AggregateID: "order-1", PayloadVersion: 1, Attempts: 3}}
	client := &http.Client{Transport: webhookRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("server down")), Request: r}, nil
	})}
	worker := Worker{
		DB:                  structExecutor{},
		Store:               store,
		WorkerID:            "w1",
		Lease:               time.Minute,
		RetryAfter:          time.Minute,
		MaxDeliveryAttempts: 5,
		Dispatcher:          Dispatcher{Client: client, Endpoints: []Endpoint{{ID: "erp", URL: "https://erp.example.test/hook", Secret: "secret", Topics: map[string]struct{}{"order.created": {}}}}, MaxAttempts: 1},
	}
	processed, err := worker.RunOnce(context.Background(), now)
	if err == nil || !processed || !store.retried {
		t.Fatalf("expected retried status, got processed=%v err=%v store=%+v", processed, err, store)
	}
	expectedNext := now.Add(4 * time.Minute)
	if !store.retryNext.Equal(expectedNext) {
		t.Fatalf("expected retryNext %v, got %v", expectedNext, store.retryNext)
	}
	if !strings.Contains(store.retryReason, "unexpected status 502") {
		t.Fatalf("expected failure reason logged in retryReason, got %q", store.retryReason)
	}
}

type structExecutor struct{}

func (structExecutor) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, nil
}
func (structExecutor) QueryRowContext(context.Context, string, ...any) *sql.Row { return nil }
