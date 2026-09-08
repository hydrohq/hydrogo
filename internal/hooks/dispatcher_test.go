package hooks

import (
	"context"
	"errors"
	"testing"
)

func TestDispatcherIsBoundedAndIdempotent(t *testing.T) {
	count := 0
	d := New(2, func(_ context.Context, _ Event) error { count++; return nil })
	event := Event{ID: "event-1", Type: "page.published", AggregateID: "page-1", Version: 1}
	if err := d.Dispatch(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("duplicate event handler calls = %d, want 1", count)
	}
	if err := d.Dispatch(context.Background(), Event{ID: "event-2", Type: "page.updated", AggregateID: "page-1", Version: 2}); err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch(context.Background(), Event{ID: "event-3", Type: "page.updated", AggregateID: "page-1", Version: 3}); err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("evicted event was not allowed to run again: %d", count)
	}
}

func TestDispatcherRetriesFailedEvent(t *testing.T) {
	fail := errors.New("temporary hook failure")
	attempts := 0
	d := New(4, func(_ context.Context, _ Event) error {
		attempts++
		if attempts == 1 {
			return fail
		}
		return nil
	})
	event := Event{ID: "event-1", Type: "news.published", AggregateID: "news-1", Version: 1}
	if err := d.Dispatch(context.Background(), event); !errors.Is(err, fail) {
		t.Fatalf("first dispatch = %v", err)
	}
	if err := d.Dispatch(context.Background(), event); err != nil {
		t.Fatalf("retry dispatch = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("handler attempts = %d, want 2", attempts)
	}
}
