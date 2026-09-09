package news

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/jobs"
)

type scheduleJob struct {
	ItemID     string `json:"item_id"`
	Generation int64  `json:"generation"`
}

// DurableQueue is the persistent lease contract implemented by the SQL job
// store. Queue remains supported for compact in-process operation.
type DurableQueue interface {
	Enqueue(context.Context, jobs.Job) error
	Claim(context.Context, string, time.Time, time.Duration) (jobs.Job, string, bool, error)
	Ack(context.Context, string, string, string) error
	Nack(context.Context, string, string, string, time.Time) error
	DeadLetter(context.Context, string, string, string, string) error
}

type Worker struct {
	Queue        *jobs.Queue
	Durable      DurableQueue
	Service      Service
	Lease        time.Duration
	Retry        time.Duration
	MaxAttempts  int
	OnRetry      func()
	OnDeadLetter func()
}

// EnqueueDue discovers scheduled items that are due and idempotently places
// their generation-guarded jobs on the configured queue.
func (w Worker) EnqueueDue(ctx context.Context, db database.Executor, now time.Time) error {
	if guards.IsNil(w.Service.Repository) || database.RequireExecutor(db) != nil {
		return errors.New("news scheduler repository is unavailable")
	}
	items, err := w.Service.Repository.ListScheduled(ctx, db, now)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := w.EnqueueScheduleContext(ctx, item.ID, item.ScheduleGeneration, now); err != nil {
			return err
		}
	}
	return nil
}

func (w Worker) EnqueueSchedule(itemID string, generation int64, availableAt time.Time) error {
	return w.EnqueueScheduleContext(context.Background(), itemID, generation, availableAt)
}

func (w Worker) EnqueueScheduleContext(ctx context.Context, itemID string, generation int64, availableAt time.Time) error {
	if (w.Queue == nil && guards.IsNil(w.Durable)) || itemID == "" || generation <= 0 {
		return ErrScheduleStale
	}
	payload, err := json.Marshal(scheduleJob{ItemID: itemID, Generation: generation})
	if err != nil {
		return err
	}
	job := jobs.Job{ID: fmt.Sprintf("news.schedule.%s.%d", itemID, generation), Type: "news.publish", Payload: string(payload), AvailableAt: availableAt.UTC()}
	if !guards.IsNil(w.Durable) {
		return w.Durable.Enqueue(ctx, job)
	}
	return w.Queue.Enqueue(job)
}

func (w Worker) RunOnce(ctx context.Context, workerID string, now time.Time) (bool, error) {
	if w.Queue == nil && guards.IsNil(w.Durable) {
		return false, errors.New("news worker queue is unavailable")
	}
	lease := w.Lease
	if lease <= 0 {
		lease = time.Minute
	}
	retry := w.Retry
	if retry <= 0 {
		retry = time.Minute
	}
	var (
		job   jobs.Job
		token string
		ok    bool
		err   error
	)
	if !guards.IsNil(w.Durable) {
		job, token, ok, err = w.Durable.Claim(ctx, workerID, now, lease)
	} else {
		job, token, ok = w.Queue.Claim(workerID, now, lease)
	}
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	var payload scheduleJob
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
		if err := w.deadLetter(ctx, job.ID, workerID, token, err.Error()); err != nil {
			return true, err
		}
		w.recordDeadLetter()
		return true, nil
	}
	err = w.Service.RunScheduled(ctx, payload.ItemID, payload.Generation, now)
	if errors.Is(err, ErrScheduleStale) {
		return true, w.ack(ctx, job.ID, workerID, token)
	}
	if err != nil {
		if job.Attempts >= w.maxAttempts() {
			if deadLetterErr := w.deadLetter(ctx, job.ID, workerID, token, err.Error()); deadLetterErr != nil {
				return true, deadLetterErr
			}
			w.recordDeadLetter()
			return true, nil
		}
		if nackErr := w.nack(ctx, job.ID, workerID, token, now.Add(retryDelay(retry, job.Attempts)), err.Error()); nackErr != nil {
			return true, nackErr
		}
		w.recordRetry()
		return true, nil
	}
	return true, w.ack(ctx, job.ID, workerID, token)
}

func (w Worker) recordRetry() {
	if w.OnRetry != nil {
		w.OnRetry()
	}
}

func (w Worker) recordDeadLetter() {
	if w.OnDeadLetter != nil {
		w.OnDeadLetter()
	}
}

func (w Worker) ack(ctx context.Context, id, worker, token string) error {
	if !guards.IsNil(w.Durable) {
		return w.Durable.Ack(ctx, id, worker, token)
	}
	return w.Queue.Ack(id, worker, token)
}

func (w Worker) nack(ctx context.Context, id, worker, token string, retryAt time.Time, reason string) error {
	if !guards.IsNil(w.Durable) {
		if queue, ok := w.Durable.(interface {
			NackWithError(context.Context, string, string, string, time.Time, string) error
		}); ok {
			return queue.NackWithError(ctx, id, worker, token, retryAt, reason)
		}
		return w.Durable.Nack(ctx, id, worker, token, retryAt)
	}
	if queue, ok := any(w.Queue).(interface {
		NackWithError(string, string, string, time.Time, string) error
	}); ok {
		return queue.NackWithError(id, worker, token, retryAt, reason)
	}
	return w.Queue.Nack(id, worker, token, retryAt)
}

func (w Worker) deadLetter(ctx context.Context, id, worker, token, reason string) error {
	if !guards.IsNil(w.Durable) {
		return w.Durable.DeadLetter(ctx, id, worker, token, reason)
	}
	return w.Queue.DeadLetter(id, worker, token, reason)
}

func (w Worker) maxAttempts() int {
	if w.MaxAttempts <= 0 {
		return 5
	}
	return w.MaxAttempts
}

func retryDelay(base time.Duration, attempts int) time.Duration {
	if base <= 0 {
		base = time.Minute
	}
	if attempts < 1 {
		attempts = 1
	}
	delay := base
	for i := 1; i < attempts && delay < time.Hour; i++ {
		if delay > time.Hour/2 {
			return time.Hour
		}
		delay *= 2
	}
	if delay > time.Hour {
		return time.Hour
	}
	return delay
}
