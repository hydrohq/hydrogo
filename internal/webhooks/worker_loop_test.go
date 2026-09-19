package webhooks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/outbox"
)

type loopStore struct{ calls int }

func (s *loopStore) Claim(context.Context, database.Executor, string, time.Time, time.Duration) (outbox.Event, string, bool, error) {
	s.calls++
	return outbox.Event{}, "", false, errors.New("database is locked")
}
func (*loopStore) Ack(context.Context, database.Executor, string, string, string) error { return nil }
func (*loopStore) Retry(context.Context, database.Executor, string, string, string, time.Time, string) error {
	return nil
}

func TestWorkerRunBacksOffTransientDatabaseErrorsAndStopsOnContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := &loopStore{}
	var delays []time.Duration
	worker := Worker{
		DB: structExecutor{}, Store: store, WorkerID: "loop", Lease: time.Minute,
		PollInterval: 10 * time.Millisecond, MaxPollInterval: 40 * time.Millisecond,
		sleep: func(ctx context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			if len(delays) >= 3 {
				cancel()
			}
			return ctx.Err()
		},
	}
	if err := worker.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("worker run error = %v", err)
	}
	if len(delays) != 3 || delays[0] != 10*time.Millisecond || delays[1] != 20*time.Millisecond || delays[2] != 40*time.Millisecond {
		t.Fatalf("poll delays = %v", delays)
	}
}

func TestWorkerRunRejectsUnboundedPollingConfiguration(t *testing.T) {
	worker := Worker{PollInterval: time.Nanosecond, MaxPollInterval: time.Second}
	if err := worker.Run(context.Background()); !errors.Is(err, ErrInvalidWorkerLoop) {
		t.Fatalf("invalid loop error = %v", err)
	}
}
