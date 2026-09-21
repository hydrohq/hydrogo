package jobs

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"
)

func TestQueueIsBoundedIdempotentAndReclaimsExpiredLease(t *testing.T) {
	q := NewQueue(1)
	created := time.Unix(100, 0).UTC()
	job := Job{ID: "news-1", Type: "news.publish", Payload: "generation=1", CreatedAt: created, AvailableAt: created}
	if err := q.Enqueue(job); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(job); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(Job{ID: "news-2", Type: "news.publish", Payload: "generation=2", CreatedAt: created}); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("queue full = %v", err)
	}
	claimed, token, ok := q.Claim("worker-a", created, time.Minute)
	if !ok || claimed.Attempts != 1 {
		t.Fatalf("claim = %#v/%q/%v", claimed, token, ok)
	}
	if _, _, ok := q.Claim("worker-b", created.Add(30*time.Second), time.Minute); ok {
		t.Fatal("active lease was claimed twice")
	}
	reclaimed, retryToken, ok := q.Claim("worker-b", created.Add(2*time.Minute), time.Minute)
	if !ok || reclaimed.Attempts != 2 || retryToken == token {
		t.Fatalf("expired lease reclaim = %#v/%q/%v", reclaimed, retryToken, ok)
	}
	if err := q.Ack(claimed.ID, "worker-a", token); !errors.Is(err, ErrLeaseInvalid) {
		t.Fatalf("old lease ack = %v", err)
	}
	if err := q.Ack(reclaimed.ID, "worker-b", retryToken); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(Job{ID: "dead", Type: "news.publish", Payload: "bad", CreatedAt: created, AvailableAt: created}); err != nil {
		t.Fatal(err)
	}
	dead, deadToken, ok := q.Claim("worker-c", created, time.Minute)
	if !ok {
		t.Fatal("dead-letter test job was not claimed")
	}
	if err := q.DeadLetter(dead.ID, "worker-c", deadToken, "permanent failure"); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := q.Claim("worker-d", created.Add(time.Hour), time.Minute); ok {
		t.Fatal("dead-lettered job remained claimable")
	}
}

func TestBoundErrorTruncatesByRune(t *testing.T) {
	reason := strings.Repeat("🚀", MaxErrorBytes+10)
	got := BoundError(reason)
	if len(got) > MaxErrorBytes {
		t.Fatalf("bounded error bytes = %d", len(got))
	}
	if !utf8.ValidString(got) {
		t.Fatal("bounded error is not valid UTF-8")
	}
}

func TestQueueFailsClosedWhenNilOrZeroValue(t *testing.T) {
	var nilQueue *Queue
	if err := nilQueue.Enqueue(Job{ID: "job", Type: "work"}); !errors.Is(err, ErrQueueUnavailable) {
		t.Fatalf("nil queue enqueue = %v, want %v", err, ErrQueueUnavailable)
	}
	if _, _, ok := nilQueue.Claim("worker", time.Now(), time.Minute); ok {
		t.Fatal("nil queue unexpectedly claimed a job")
	}
	zeroQueue := Queue{}
	if err := zeroQueue.Enqueue(Job{ID: "job", Type: "work"}); !errors.Is(err, ErrQueueUnavailable) {
		t.Fatalf("zero-value queue enqueue = %v, want %v", err, ErrQueueUnavailable)
	}
}

func TestQueueConcurrentWorkersClaimAndAcknowledgeEachJobOnce(t *testing.T) {
	const jobsToProcess = 128
	q := NewQueue(jobsToProcess)
	now := time.Unix(100, 0).UTC()
	for i := 0; i < jobsToProcess; i++ {
		if err := q.Enqueue(Job{ID: fmt.Sprintf("job-%03d", i), Type: "news.publish", Payload: "generation=1", CreatedAt: now, AvailableAt: now}); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}

	var claimed sync.Map
	var acknowledged atomic.Int32
	var workers sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			for {
				job, token, ok := q.Claim(fmt.Sprintf("worker-%d", worker), now, time.Minute)
				if !ok {
					return
				}
				if _, loaded := claimed.LoadOrStore(job.ID, struct{}{}); loaded {
					t.Errorf("job %q was claimed more than once", job.ID)
					return
				}
				if err := q.Ack(job.ID, fmt.Sprintf("worker-%d", worker), token); err != nil {
					t.Errorf("ack %q: %v", job.ID, err)
					return
				}
				acknowledged.Add(1)
			}
		}(worker)
	}
	workers.Wait()
	if got := int(acknowledged.Load()); got != jobsToProcess {
		t.Fatalf("acknowledged %d jobs, want %d", got, jobsToProcess)
	}
	if len(q.jobs) != 0 {
		t.Fatalf("queue retained %d jobs after all acknowledgements", len(q.jobs))
	}
}
