package sqlstore

import (
	"context"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/jobs"
)

// Queue adapts Store to the context-aware durable queue contract used by
// modules/news.Worker.
type Queue struct {
	Store Store
	DB    database.Executor
}

func (q Queue) Enqueue(ctx context.Context, job jobs.Job) error {
	return q.Store.Enqueue(ctx, q.DB, job)
}

func (q Queue) Claim(ctx context.Context, worker string, now time.Time, duration time.Duration) (jobs.Job, string, bool, error) {
	return q.Store.Claim(ctx, q.DB, worker, now, duration)
}

func (q Queue) Ack(ctx context.Context, id, worker, token string) error {
	return q.Store.Ack(ctx, q.DB, id, worker, token)
}

func (q Queue) Nack(ctx context.Context, id, worker, token string, retryAt time.Time) error {
	return q.Store.Nack(ctx, q.DB, id, worker, token, retryAt)
}
