package webhooks

import (
	"context"
	"errors"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/outbox"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

var ErrInvalidWorkerLoop = errors.New("invalid webhook worker loop configuration")

type OutboxStore interface {
	Claim(context.Context, database.Executor, string, time.Time, time.Duration) (outbox.Event, string, bool, error)
	Ack(context.Context, database.Executor, string, string, string) error
	Retry(context.Context, database.Executor, string, string, string, time.Time, string) error
}

type Worker struct {
	DB              database.Executor
	Store           OutboxStore
	Dispatcher      Dispatcher
	WorkerID        string
	Lease           time.Duration
	RetryAfter      time.Duration
	PollInterval    time.Duration
	MaxPollInterval time.Duration
	// TenantScopes makes shared-tenancy workers explicit. Each polling pass is
	// executed with one validated scope; an empty slice preserves single-tenant
	// behavior for deployments where tenancy is disabled.
	TenantScopes []tenant.Scope
	OnError      func(error)
	sleep        func(context.Context, time.Duration) error
}

// Run owns the bounded outbox polling lifecycle. Empty queues and transient
// SQLite/PostgreSQL errors back off exponentially up to MaxPollInterval; a
// successful claim immediately returns to the base interval. The loop exits
// only when ctx is cancelled, so a single lock/error cannot spin or kill the
// worker process.
func (w Worker) Run(ctx context.Context) error {
	base, max, err := w.pollBounds()
	if err != nil {
		return err
	}
	sleep := w.sleep
	if sleep == nil {
		sleep = sleepContext
	}
	interval := base
	for {
		processed, runErr := w.runScopedOnce(ctx, time.Now().UTC())
		if runErr != nil && w.OnError != nil {
			w.OnError(runErr)
		}
		if err := sleep(ctx, interval); err != nil {
			return err
		}
		if runErr == nil && processed {
			interval = base
		} else {
			interval = nextPollInterval(interval, max)
		}
	}
}

func (w Worker) runScopedOnce(ctx context.Context, now time.Time) (bool, error) {
	if len(w.TenantScopes) == 0 {
		return w.RunOnce(ctx, now)
	}
	var processed bool
	var firstErr error
	for _, scope := range w.TenantScopes {
		scopedCtx := tenant.WithScope(ctx, scope)
		claimed, err := w.RunOnce(scopedCtx, now)
		processed = processed || claimed
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return processed, firstErr
}

func (w Worker) pollBounds() (time.Duration, time.Duration, error) {
	base := w.PollInterval
	if base <= 0 {
		base = 250 * time.Millisecond
	}
	max := w.MaxPollInterval
	if max <= 0 {
		max = 5 * time.Second
	}
	if base > max || base < 10*time.Millisecond || max > 5*time.Minute {
		return 0, 0, ErrInvalidWorkerLoop
	}
	return base, max, nil
}

func nextPollInterval(current, max time.Duration) time.Duration {
	if current >= max || current > max/2 {
		return max
	}
	return current * 2
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
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
	ctx = security.WithJobID(ctx, event.ID)
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
