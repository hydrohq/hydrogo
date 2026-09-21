package analytics

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
)

type testSink struct {
	mu    sync.Mutex
	count int
	fails bool
}

func (s *testSink) RecordContentView(context.Context, string, string, v1.ContentViewEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.count++
	if s.fails {
		return context.Canceled
	}
	return nil
}

func TestContentViewQueueIsBoundedAndAsynchronous(t *testing.T) {
	sink := &testSink{}
	q := NewContentViewQueue(sink, 1, 1)
	defer q.Close()
	event := ContentViewJob{ConsentID: "opaque", PurposeVersion: "v1", Event: v1.ContentViewEvent{Day: "2026-09-22", TenantID: "t1", ContentType: "article", ContentID: "a1"}}
	accepted := 0
	for i := 0; i < 1000; i++ {
		if q.Enqueue(event) {
			accepted++
		}
	}
	if accepted == 0 || q.Dropped() == 0 {
		t.Fatalf("queue was not bounded: accepted=%d dropped=%d", accepted, q.Dropped())
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		sink.mu.Lock()
		count := sink.count
		sink.mu.Unlock()
		if count > 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("queued view was not delivered asynchronously")
}
