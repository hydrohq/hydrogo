package analytics

import (
	"context"
	"sync"
	"testing"
	"time"
)

type searchSink struct {
	mu     sync.Mutex
	events []SearchQueryEvent
}

func (s *searchSink) RecordSearchQuery(_ context.Context, event SearchQueryEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func TestNormalizeSearchTerm(t *testing.T) {
	for input, want := range map[string]string{"  HYDRO   GO ": "hydro go", "": "", "user@example.com": "", "call 1234567": ""} {
		if got := NormalizeSearchTerm(input); got != want {
			t.Errorf("NormalizeSearchTerm(%q)=%q want %q", input, got, want)
		}
	}
}

func TestSearchQueryQueuePersistsEvents(t *testing.T) {
	sink := &searchSink{}
	queue := NewSearchQueryQueue(sink, 2, 1)
	if !queue.Enqueue(SearchQueryEvent{Day: SearchQueryDay(time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)), TenantID: "tenant-a", Term: "hydro", ResultCount: 0}) {
		t.Fatal("enqueue failed")
	}
	queue.Close()
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if len(sink.events) != 1 || sink.events[0].ResultCount != 0 {
		t.Fatalf("events=%+v", sink.events)
	}
}
