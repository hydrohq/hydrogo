package news

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
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
}

type Worker struct {
	Queue   *jobs.Queue
	Durable DurableQueue
	Service Service
	Lease   time.Duration
	Retry   time.Duration
}

// EnqueueDue discovers scheduled items that are due and idempotently places
// their generation-guarded jobs on the configured queue.
func (w Worker) EnqueueDue(ctx context.Context, db database.Executor, now time.Time) error {
	if w.Service.Repository == nil || db == nil {
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
	if (w.Queue == nil && w.Durable == nil) || itemID == "" || generation <= 0 {
		return ErrScheduleStale
	}
	payload, err := json.Marshal(scheduleJob{ItemID: itemID, Generation: generation})
	if err != nil {
		return err
	}
	job := jobs.Job{ID: fmt.Sprintf("news.schedule.%s.%d", itemID, generation), Type: "news.publish", Payload: string(payload), AvailableAt: availableAt.UTC()}
	if w.Durable != nil {
		return w.Durable.Enqueue(ctx, job)
	}
	return w.Queue.Enqueue(job)
}

func (w Worker) RunOnce(ctx context.Context, workerID string, now time.Time) (bool, error) {
	if w.Queue == nil && w.Durable == nil {
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
	if w.Durable != nil {
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
		_ = w.ack(ctx, job.ID, workerID, token)
		return true, err
	}
	err = w.Service.RunScheduled(ctx, payload.ItemID, payload.Generation, now)
	if errors.Is(err, ErrScheduleStale) {
		return true, w.ack(ctx, job.ID, workerID, token)
	}
	if err != nil {
		return true, w.nack(ctx, job.ID, workerID, token, now.Add(retry))
	}
	return true, w.ack(ctx, job.ID, workerID, token)
}

func (w Worker) ack(ctx context.Context, id, worker, token string) error {
	if w.Durable != nil {
		return w.Durable.Ack(ctx, id, worker, token)
	}
	return w.Queue.Ack(id, worker, token)
}

func (w Worker) nack(ctx context.Context, id, worker, token string, retryAt time.Time) error {
	if w.Durable != nil {
		return w.Durable.Nack(ctx, id, worker, token, retryAt)
	}
	return w.Queue.Nack(id, worker, token, retryAt)
}
