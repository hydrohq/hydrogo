package webhooks

import (
	"context"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/outbox"
)

type OutboxStore interface {
	Claim(context.Context, database.Executor, string, time.Time, time.Duration) (outbox.Event, string, bool, error)
	Ack(context.Context, database.Executor, string, string, string) error
	Retry(context.Context, database.Executor, string, string, string, time.Time, string) error
}

type Worker struct {
	DB         database.Executor
	Store      OutboxStore
	Dispatcher Dispatcher
	WorkerID   string
	Lease      time.Duration
	RetryAfter time.Duration
}

// RunOnce claims at most one event. Delivery is acknowledged only after every
// matching endpoint succeeds; failures release the lease for a bounded retry.
func (w Worker) RunOnce(ctx context.Context, now time.Time) (bool, error) {
	if w.Store == nil || w.DB == nil || w.WorkerID == "" || w.Lease <= 0 {
		return false, nil
	}
	event, token, claimed, err := w.Store.Claim(ctx, w.DB, w.WorkerID, now, w.Lease)
	if err != nil || !claimed {
		return claimed, err
	}
	err = w.Dispatcher.Dispatch(ctx, Event{ID: event.ID, Topic: event.Topic, AggregateID: event.AggregateID, PayloadVersion: event.PayloadVersion})
	if err == nil {
		return true, w.Store.Ack(ctx, w.DB, event.ID, w.WorkerID, token)
	}
	delay := w.RetryAfter
	if delay <= 0 {
		delay = time.Minute
	}
	if retryErr := w.Store.Retry(ctx, w.DB, event.ID, w.WorkerID, token, now.Add(delay), err.Error()); retryErr != nil {
		return true, retryErr
	}
	return true, err
}
