package realtime

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type memoryBus struct {
	mu          sync.Mutex
	subscribers []func(tenantID, topic string, payload []byte)
}

func (m *memoryBus) Publish(ctx context.Context, tenantID, topic string, payload []byte) error {
	m.mu.Lock()
	subs := append([]func(tenantID, topic string, payload []byte){}, m.subscribers...)
	m.mu.Unlock()
	for _, fn := range subs {
		fn(tenantID, topic, payload)
	}
	return nil
}

type memoryBusCloser struct {
	bus *memoryBus
	fn  func(tenantID, topic string, payload []byte)
}

func (c *memoryBusCloser) Close() error {
	c.bus.mu.Lock()
	defer c.bus.mu.Unlock()
	for i, sub := range c.bus.subscribers {
		// Compare function pointers or simply clear
		if &sub == &c.fn {
			c.bus.subscribers = append(c.bus.subscribers[:i], c.bus.subscribers[i+1:]...)
			break
		}
	}
	return nil
}

func (m *memoryBus) Subscribe(ctx context.Context, handler func(tenantID, topic string, payload []byte)) (io.Closer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers = append(m.subscribers, handler)
	return &memoryBusCloser{bus: m, fn: handler}, nil
}

func TestHubDistributedBusBroadcast(t *testing.T) {
	bus := &memoryBus{}

	hub1 := &Hub{}
	if err := hub1.SetBus(bus); err != nil {
		t.Fatalf("hub1 SetBus: %v", err)
	}
	defer hub1.Close()

	hub2 := &Hub{}
	if err := hub2.SetBus(bus); err != nil {
		t.Fatalf("hub2 SetBus: %v", err)
	}
	defer hub2.Close()

	// Subscribe on hub2 for tenant "t1", topic "news:created"
	sub, err := hub2.subscribe("t1", []string{"news:created"})
	if err != nil {
		t.Fatalf("hub2 subscribe: %v", err)
	}
	defer hub2.unsubscribe(sub)

	// Publish on hub1 for tenant "t1", topic "news:created"
	scope1, _ := tenant.NewScope("t1", "s1")
	ctx1 := tenant.WithScope(context.Background(), scope1)
	evt := Event{
		ID:    "evt-1",
		Topic: "news:created",
		Data:  json.RawMessage(`{"title":"hello distributed"}`),
	}
	if err := hub1.Publish(ctx1, evt); err != nil {
		t.Fatalf("hub1 Publish: %v", err)
	}

	select {
	case received := <-sub.events:
		if received.ID != "evt-1" {
			t.Fatalf("expected evt-1, got %s", received.ID)
		}
		if received.Topic != "news:created" {
			t.Fatalf("expected topic news:created, got %s", received.Topic)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event on hub2")
	}
}

func TestHubDistributedBusTenantAndTopicIsolation(t *testing.T) {
	bus := &memoryBus{}

	hub1 := &Hub{}
	_ = hub1.SetBus(bus)
	defer hub1.Close()

	hub2 := &Hub{}
	_ = hub2.SetBus(bus)
	defer hub2.Close()

	// Sub A: tenant t1, topic news:created
	subA, err := hub2.subscribe("t1", []string{"news:created"})
	if err != nil {
		t.Fatal(err)
	}
	defer hub2.unsubscribe(subA)

	// Sub B: tenant t2, topic news:created
	subB, err := hub2.subscribe("t2", []string{"news:created"})
	if err != nil {
		t.Fatal(err)
	}
	defer hub2.unsubscribe(subB)

	// Sub C: tenant t1, topic pages:created
	subC, err := hub2.subscribe("t1", []string{"pages:created"})
	if err != nil {
		t.Fatal(err)
	}
	defer hub2.unsubscribe(subC)

	// Publish from hub1 for tenant t1, topic news:created
	scope1, _ := tenant.NewScope("t1", "s1")
	ctx := tenant.WithScope(context.Background(), scope1)
	_ = hub1.Publish(ctx, Event{ID: "t1-news", Topic: "news:created", Data: json.RawMessage(`{}`)})

	select {
	case <-subA.events:
		// Expected
	case <-time.After(1 * time.Second):
		t.Fatal("subA did not receive expected event")
	}

	// Verify subB and subC did NOT receive it
	select {
	case ev := <-subB.events:
		t.Fatalf("subB leaked cross-tenant event: %v", ev)
	case ev := <-subC.events:
		t.Fatalf("subC leaked wrong-topic event: %v", ev)
	default:
		// Success
	}
}

func TestRedisMessageBusChannelFormatting(t *testing.T) {
	bus := NewRedisMessageBus(nil, "app-cluster")
	expected := "app-cluster:hydro:realtime:tenant-123:records.created"
	if got := bus.channel("tenant-123", "records.created"); got != expected {
		t.Fatalf("expected channel %q, got %q", expected, got)
	}
}
