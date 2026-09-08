package news

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/jobs"
)

type scheduleJob struct {
	ItemID     string `json:"item_id"`
	Generation int64  `json:"generation"`
}

type Worker struct {
	Queue   *jobs.Queue
	Service Service
	Lease   time.Duration
	Retry   time.Duration
}

func (w Worker) EnqueueSchedule(itemID string, generation int64, availableAt time.Time) error {
	if w.Queue == nil || itemID == "" || generation <= 0 {
		return ErrScheduleStale
	}
	payload, err := json.Marshal(scheduleJob{ItemID: itemID, Generation: generation})
	if err != nil {
		return err
	}
	return w.Queue.Enqueue(jobs.Job{ID: fmt.Sprintf("news.schedule.%s.%d", itemID, generation), Type: "news.publish", Payload: string(payload), AvailableAt: availableAt.UTC()})
}

func (w Worker) RunOnce(ctx context.Context, workerID string, now time.Time) (bool, error) {
	if w.Queue == nil {
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
	job, token, ok := w.Queue.Claim(workerID, now, lease)
	if !ok {
		return false, nil
	}
	var payload scheduleJob
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
		_ = w.Queue.Ack(job.ID, workerID, token)
		return true, err
	}
	err := w.Service.RunScheduled(ctx, payload.ItemID, payload.Generation, now)
	if errors.Is(err, ErrScheduleStale) {
		return true, w.Queue.Ack(job.ID, workerID, token)
	}
	if err != nil {
		return true, w.Queue.Nack(job.ID, workerID, token, now.Add(retry))
	}
	return true, w.Queue.Ack(job.ID, workerID, token)
}
