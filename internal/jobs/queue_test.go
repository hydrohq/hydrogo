package jobs

import (
	"errors"
	"testing"
	"time"
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
}
