// Package jobs contains bounded, idempotent in-process job lease primitives.
package jobs

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var (
	ErrQueueFull        = errors.New("job queue is full")
	ErrQueueUnavailable = errors.New("job queue is unavailable")
	ErrJobConflict      = errors.New("job id already has different payload")
	ErrLeaseInvalid     = errors.New("job lease is invalid")
)

const MaxErrorBytes = 1024

// BoundError keeps retry/dead-letter diagnostics useful without allowing an
// untrusted downstream error to grow the durable jobs table without bound.
func BoundError(reason string) string {
	reason = strings.TrimSpace(reason)
	if len(reason) <= MaxErrorBytes {
		return reason
	}
	bounded := reason[:MaxErrorBytes]
	for !utf8.ValidString(bounded) {
		bounded = bounded[:len(bounded)-1]
	}
	return bounded
}

type Job struct {
	ID           string
	Type         string
	Payload      string
	AvailableAt  time.Time
	CreatedAt    time.Time
	Attempts     int
	DeadLettered bool
	LastError    string
}

type lease struct {
	Job
	Worker     string
	Token      string
	LeaseUntil time.Time
}

type Queue struct {
	mu         sync.Mutex
	jobs       map[string]*lease
	maxEntries int
}

func NewQueue(maxEntries int) *Queue {
	if maxEntries < 1 {
		maxEntries = 1
	}
	return &Queue{jobs: make(map[string]*lease), maxEntries: maxEntries}
}

func (q *Queue) Enqueue(job Job) error {
	if q == nil || q.jobs == nil || q.maxEntries < 1 {
		return ErrQueueUnavailable
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if job.ID == "" || job.Type == "" {
		return ErrJobConflict
	}
	if existing, ok := q.jobs[job.ID]; ok {
		if existing.Type != job.Type || existing.Payload != job.Payload {
			return ErrJobConflict
		}
		return nil
	}
	if len(q.jobs) >= q.maxEntries {
		return ErrQueueFull
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	if job.AvailableAt.IsZero() {
		job.AvailableAt = job.CreatedAt
	}
	q.jobs[job.ID] = &lease{Job: job}
	return nil
}

func (q *Queue) Claim(worker string, now time.Time, duration time.Duration) (Job, string, bool) {
	if q == nil || q.jobs == nil || q.maxEntries < 1 {
		return Job{}, "", false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if worker == "" || duration <= 0 {
		return Job{}, "", false
	}
	candidates := make([]*lease, 0)
	for _, candidate := range q.jobs {
		if candidate.DeadLettered {
			continue
		}
		if candidate.Worker != "" && now.Before(candidate.LeaseUntil) {
			continue
		}
		if now.Before(candidate.AvailableAt) {
			continue
		}
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].AvailableAt.Equal(candidates[j].AvailableAt) {
			return candidates[i].ID < candidates[j].ID
		}
		return candidates[i].AvailableAt.Before(candidates[j].AvailableAt)
	})
	if len(candidates) == 0 {
		return Job{}, "", false
	}
	selected := candidates[0]
	selected.Worker = worker
	selected.LeaseUntil = now.Add(duration)
	selected.Attempts++
	selected.Token = worker + ":" + selected.ID + ":" + now.UTC().Format(time.RFC3339Nano)
	return selected.Job, selected.Token, true
}

func (q *Queue) Ack(id, worker, token string) error {
	if q == nil || q.jobs == nil || q.maxEntries < 1 {
		return ErrQueueUnavailable
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	selected, ok := q.jobs[id]
	if !ok || selected.Worker != worker || selected.Token != token {
		return ErrLeaseInvalid
	}
	delete(q.jobs, id)
	return nil
}

func (q *Queue) Nack(id, worker, token string, retryAt time.Time) error {
	return q.NackWithError(id, worker, token, retryAt, "")
}

func (q *Queue) NackWithError(id, worker, token string, retryAt time.Time, reason string) error {
	if q == nil || q.jobs == nil || q.maxEntries < 1 {
		return ErrQueueUnavailable
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	selected, ok := q.jobs[id]
	if !ok || selected.Worker != worker || selected.Token != token {
		return ErrLeaseInvalid
	}
	selected.Worker = ""
	selected.Token = ""
	selected.LeaseUntil = time.Time{}
	selected.AvailableAt = retryAt
	selected.LastError = BoundError(reason)
	return nil
}

func (q *Queue) DeadLetter(id, worker, token, reason string) error {
	if q == nil || q.jobs == nil || q.maxEntries < 1 {
		return ErrQueueUnavailable
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	selected, ok := q.jobs[id]
	if !ok || selected.Worker != worker || selected.Token != token {
		return ErrLeaseInvalid
	}
	selected.Worker = ""
	selected.Token = ""
	selected.LeaseUntil = time.Time{}
	selected.DeadLettered = true
	selected.LastError = BoundError(reason)
	return nil
}
