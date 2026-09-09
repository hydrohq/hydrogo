package news

import (
	"context"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/jobs"
)

func TestRetryDelayIsBoundedExponential(t *testing.T) {
	base := 10 * time.Second
	if got := retryDelay(base, 1); got != 10*time.Second {
		t.Fatalf("first retry = %s", got)
	}
	if got := retryDelay(base, 3); got != 40*time.Second {
		t.Fatalf("third retry = %s", got)
	}
	if got := retryDelay(base, 20); got != time.Hour {
		t.Fatalf("capped retry = %s", got)
	}
}

func TestMalformedJobRecordsDeadLetterCallback(t *testing.T) {
	queue := jobs.NewQueue(1)
	created := time.Now().UTC()
	if err := queue.Enqueue(jobs.Job{ID: "bad", Type: "news.publish", Payload: "not-json", AvailableAt: created, CreatedAt: created}); err != nil {
		t.Fatal(err)
	}
	deadLetters := 0
	worker := Worker{Queue: queue, OnDeadLetter: func() { deadLetters++ }}
	processed, err := worker.RunOnce(context.Background(), "worker-1", created)
	if err != nil || !processed || deadLetters != 1 {
		t.Fatalf("malformed job result = processed:%v err:%v dead_letters:%d", processed, err, deadLetters)
	}
}
