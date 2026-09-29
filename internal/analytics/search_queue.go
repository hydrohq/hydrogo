package analytics

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/observability"
)

type SearchQueryEvent struct {
	Day         string
	TenantID    string
	Term        string
	Locale      string
	ResultCount int
}

type SearchInsight struct {
	Term, Locale                 string
	SearchCount, ZeroResultCount int64
}

type SearchQuerySink interface {
	RecordSearchQuery(context.Context, SearchQueryEvent) error
}

type SearchQueryQueue struct {
	sink    SearchQuerySink
	jobs    chan SearchQueryEvent
	dropped atomic.Uint64
	failed  atomic.Uint64
	close   chan struct{}
	wg      sync.WaitGroup
}

func NewSearchQueryQueue(sink SearchQuerySink, capacity, workers int) *SearchQueryQueue {
	if capacity < 1 {
		capacity = 128
	}
	if workers < 1 {
		workers = 1
	}
	q := &SearchQueryQueue{sink: sink, jobs: make(chan SearchQueryEvent, capacity), close: make(chan struct{})}
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker()
	}
	return q
}

func (q *SearchQueryQueue) Enqueue(event SearchQueryEvent) bool {
	if q == nil || q.sink == nil || event.ResultCount < 0 || event.Term == "" {
		return false
	}
	select {
	case <-q.close:
		return false
	default:
	}
	select {
	case q.jobs <- event:
		return true
	default:
		q.dropped.Add(1)
		return false
	}
}

func (q *SearchQueryQueue) Dropped() uint64 {
	if q == nil {
		return 0
	}
	return q.dropped.Load()
}
func (q *SearchQueryQueue) Failed() uint64 {
	if q == nil {
		return 0
	}
	return q.failed.Load()
}
func (q *SearchQueryQueue) Close() {
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
func (q *SearchQueryQueue) worker() {
	defer q.wg.Done()
	for {
		select {
		case <-q.close:
			for {
				select {
				case event := <-q.jobs:
					q.persist(event)
				default:
					return
				}
			}
		case event := <-q.jobs:
			q.persist(event)
		}
	}
}

func (q *SearchQueryQueue) persist(event SearchQueryEvent) {
	ctx, span := observability.StartComponentSpan(context.Background(), "analytics", "search_query.persist")
	if err := q.sink.RecordSearchQuery(ctx, event); err != nil {
		q.failed.Add(1)
	}
	span.End()
}

// NormalizeSearchTerm folds case and whitespace, rejects identifying contact
// strings, and bounds the aggregate dimension before persistence.
func NormalizeSearchTerm(value string) string {
	term := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
	if term == "" || len(term) > 120 || strings.ContainsAny(term, "\x00\r\n") || strings.Contains(term, "@") {
		return ""
	}
	digits := 0
	for _, r := range term {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	if digits >= 7 {
		return ""
	}
	return term
}

func SearchQueryDay(now time.Time) string { return now.UTC().Format("2006-01-02") }
