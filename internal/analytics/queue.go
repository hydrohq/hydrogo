package analytics

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
)

type ContentViewSink interface {
	RecordContentView(context.Context, string, string, v1.ContentViewEvent) error
}

type ContentViewJob struct {
	ConsentID      string
	PurposeVersion string
	Event          v1.ContentViewEvent
}

// ContentViewQueue keeps analytics off the request critical path. Enqueue is
// non-blocking and drops only when the explicit bounded capacity is exhausted.
// Dropped events are observable and never cause a public request to fail.
type ContentViewQueue struct {
	sink    ContentViewSink
	jobs    chan ContentViewJob
	dropped atomic.Uint64
	failed  atomic.Uint64
	close   chan struct{}
	wg      sync.WaitGroup
}

func NewContentViewQueue(sink ContentViewSink, capacity, workers int) *ContentViewQueue {
	if capacity < 1 {
		capacity = 128
	}
	if workers < 1 {
		workers = 1
	}
	q := &ContentViewQueue{sink: sink, jobs: make(chan ContentViewJob, capacity), close: make(chan struct{})}
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker()
	}
	return q
}

func (q *ContentViewQueue) Enqueue(job ContentViewJob) bool {
	if q == nil || q.sink == nil {
		return false
	}
	select {
	case <-q.close:
		return false
	default:
	}
	select {
	case q.jobs <- job:
		return true
	default:
		q.dropped.Add(1)
		return false
	}
}

func (q *ContentViewQueue) Dropped() uint64 {
	if q == nil {
		return 0
	}
	return q.dropped.Load()
}

func (q *ContentViewQueue) Failed() uint64 {
	if q == nil {
		return 0
	}
	return q.failed.Load()
}

func (q *ContentViewQueue) Close() {
	if q == nil {
		return
	}
	select {
	case <-q.close:
		return
	default:
		close(q.close)
	}
	q.wg.Wait()
}

func (q *ContentViewQueue) worker() {
	defer q.wg.Done()
	for {
		select {
		case <-q.close:
			return
		case job := <-q.jobs:
			if err := q.sink.RecordContentView(context.Background(), job.ConsentID, job.PurposeVersion, job.Event); err != nil {
				q.failed.Add(1)
			}
		}
	}
}
