package hooks

import (
	"context"
	"errors"
	"testing"
	"time"
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

func TestDispatcherRunsHandlersOutsideLockAndDeduplicatesInFlight(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	count := 0
	d := New(4, func(_ context.Context, event Event) error {
		count++
		if event.ID == "outer" {
			close(started)
			<-release
		}
		return nil
	})
	outer := Event{ID: "outer", Type: "page.updated", AggregateID: "page-1", Version: 1}
	done := make(chan error, 1)
	go func() { done <- d.Dispatch(context.Background(), outer) }()
	<-started

	duplicate := make(chan error, 1)
	go func() { duplicate <- d.Dispatch(context.Background(), outer) }()
	select {
	case err := <-duplicate:
		t.Fatalf("in-flight duplicate returned before original completed: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := <-duplicate; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("in-flight duplicate handler calls = %d, want 1", count)
	}
}

func TestDispatcherCanDispatchAnotherEventFromHandler(t *testing.T) {
	var d *Dispatcher
	inner := Event{ID: "inner", Type: "page.updated", AggregateID: "page-1", Version: 2}
	outer := Event{ID: "outer", Type: "page.updated", AggregateID: "page-1", Version: 1}
	count := 0
	d = New(4, func(ctx context.Context, event Event) error {
		count++
		if event.ID == outer.ID {
			return d.Dispatch(ctx, inner)
		}
		return nil
	})
	done := make(chan error, 1)
	go func() { done <- d.Dispatch(context.Background(), outer) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("nested dispatch deadlocked")
	}
	if count != 2 {
		t.Fatalf("nested dispatch handler calls = %d, want 2", count)
	}
}
