package webhooks

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/outbox"
)

type fakeOutbox struct {
	event                   outbox.Event
	claimed, acked, retried bool
}

func (f *fakeOutbox) Claim(context.Context, database.Executor, string, time.Time, time.Duration) (outbox.Event, string, bool, error) {
	f.claimed = true
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

func TestWorkerAcknowledgesOnlyAfterDelivery(t *testing.T) {
	store := &fakeOutbox{event: outbox.Event{ID: "event-1", Topic: "order.created", AggregateID: "order-1", PayloadVersion: 1}}
	worker := Worker{DB: structExecutor{}, Store: store, WorkerID: "w1", Lease: time.Minute, Dispatcher: Dispatcher{Endpoints: nil}}
	processed, err := worker.RunOnce(context.Background(), time.Now())
	if err != nil || !processed || !store.claimed || !store.acked || store.retried {
		t.Fatalf("worker state processed=%v err=%v store=%+v", processed, err, store)
	}
}

type structExecutor struct{}

func (structExecutor) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, nil
}
func (structExecutor) QueryRowContext(context.Context, string, ...any) *sql.Row { return nil }
