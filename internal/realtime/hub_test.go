package realtime

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func TestHubScopesEventsByTenantAndTopic(t *testing.T) {
	hub := &Hub{}
	sub, err := hub.subscribe("tenant-a", []string{"pages:updated"})
	if err != nil {
		t.Fatal(err)
	}
	ctxA := tenant.WithScope(context.Background(), tenant.Scope{TenantID: "tenant-a", Slug: "a"})
	ctxB := tenant.WithScope(context.Background(), tenant.Scope{TenantID: "tenant-b", Slug: "b"})
	if err := hub.Publish(ctxB, Event{Topic: "pages:updated", Data: []byte(`{"id":"x"}`)}); err != nil {
		t.Fatal(err)
	}
	if err := hub.Publish(ctxA, Event{Topic: "pages:created", Data: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if len(sub.events) != 0 {
		t.Fatal("unrelated tenant or topic received an event")
	}
	if err := hub.Publish(ctxA, Event{Topic: "pages:updated", Data: []byte(`{"id":"x"}`)}); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-sub.events:
		if event.Topic != "pages:updated" {
			t.Fatalf("event = %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("event not delivered")
	}
}

func TestSSERequiresAuthenticationAndStreamsEvents(t *testing.T) {
	hub := &Hub{}
	unauthorized := httptest.NewRecorder()
	hub.ServeHTTP(unauthorized, httptest.NewRequest("GET", "/?topics=pages:created", nil))
	if unauthorized.Code != 401 {
		t.Fatalf("unauthenticated response = %d", unauthorized.Code)
	}
	ctx := auth.WithUser(context.Background(), &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true})
	streamCtx, cancel := context.WithCancel(ctx)
	req := httptest.NewRequest("GET", "/?topics=pages:created", nil).WithContext(streamCtx)
	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { hub.ServeHTTP(recorder, req); close(done) }()
	deadline := time.Now().Add(time.Second)
	for {
		hub.mu.Lock()
		ready := len(hub.subscribers) == 1
		hub.mu.Unlock()
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("subscriber did not connect")
		}
		time.Sleep(time.Millisecond)
	}
	_ = hub.Publish(ctx, Event{Topic: "pages:created", Data: []byte(`{"id":"a"}`)})
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stream did not close")
	}
	if !strings.Contains(recorder.Body.String(), "event: pages:created") || !strings.Contains(recorder.Body.String(), `"id":"a"`) {
		t.Fatalf("SSE body = %q", recorder.Body.String())
	}
}

func TestSSERejectsEditorSystemTopic(t *testing.T) {
	hub := &Hub{}
	ctx := auth.WithUser(context.Background(), &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true})
	rec := httptest.NewRecorder()
	hub.ServeHTTP(rec, httptest.NewRequest("GET", "/?topics=system:changed", nil).WithContext(ctx))
	if rec.Code != 403 {
		t.Fatalf("system topic response = %d", rec.Code)
	}
}

func TestSubscribeStreamAndClose(t *testing.T) {
	hub := &Hub{}
	sub, err := hub.SubscribeStream("tenant-1", []string{"content:messages"})
	if err != nil {
		t.Fatalf("SubscribeStream error: %v", err)
	}
	defer sub.Close()

	ctx := tenant.WithScope(context.Background(), tenant.Scope{TenantID: "tenant-1", Slug: "t1"})
	if err := hub.Publish(ctx, Event{Topic: "content:messages", Data: []byte(`{"id":"m1"}`)}); err != nil {
		t.Fatalf("Publish error: %v", err)
	}

	select {
	case ev := <-sub.C:
		if ev.Topic != "content:messages" {
			t.Fatalf("unexpected topic: %s", ev.Topic)
		}
	case <-time.After(time.Second):
		t.Fatal("event not received in stream")
	}

	sub.Close()
	hub.mu.Lock()
	count := len(hub.subscribers)
	hub.mu.Unlock()
	if count != 0 {
		t.Fatalf("expected 0 subscribers after close, got %d", count)
	}
}
