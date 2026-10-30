package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

const (
	bufferSize            = 16
	defaultMaxSubscribers = 4096
	maxTopics             = 16
)

type Event struct {
	ID    string          `json:"id"`
	Topic string          `json:"topic"`
	Time  time.Time       `json:"time"`
	Data  json.RawMessage `json:"data,omitempty"`
}
type subscriber struct {
	tenant string
	topics map[string]struct{}
	events chan Event
}
type Hub struct {
	mu                 sync.Mutex
	subscribers        map[*subscriber]struct{}
	RequireTenantScope bool
	MaxSubscribers     int
	Bus                MessageBus
	busCloser          io.Closer
}

// SetBus configures a distributed message bus adapter (e.g. Redis).
func (h *Hub) SetBus(bus MessageBus) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.busCloser != nil {
		_ = h.busCloser.Close()
		h.busCloser = nil
	}
	h.Bus = bus
	if bus != nil {
		closer, err := bus.Subscribe(context.Background(), func(tenantID, topic string, payload []byte) {
			var event Event
			if err := json.Unmarshal(payload, &event); err == nil {
				h.broadcastLocal(tenantID, event)
			}
		})
		if err != nil {
			return err
		}
		h.busCloser = closer
	}
	return nil
}

// Close shuts down the hub and any attached message bus subscriptions.
func (h *Hub) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.busCloser != nil {
		err := h.busCloser.Close()
		h.busCloser = nil
		return err
	}
	return nil
}

func (h *Hub) Publish(ctx context.Context, event Event) error {
	if h == nil {
		return errors.New("realtime hub unavailable")
	}
	if !validTopic(event.Topic) {
		return errors.New("invalid realtime topic")
	}
	if h.RequireTenantScope {
		if _, err := tenant.Require(ctx); err != nil {
			return err
		}
	}
	if len(event.Data) > 16<<10 || !json.Valid(event.Data) {
		return errors.New("invalid realtime event data")
	}
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}

	tid := tenantID(ctx)

	if h.Bus != nil {
		raw, err := json.Marshal(event)
		if err != nil {
			return err
		}
		return h.Bus.Publish(ctx, tid, event.Topic, raw)
	}

	h.broadcastLocal(tid, event)
	return nil
}

func (h *Hub) broadcastLocal(tid string, event Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for sub := range h.subscribers {
		if sub.tenant != tid {
			continue
		}
		if _, ok := sub.topics[event.Topic]; !ok {
			continue
		}
		select {
		case sub.events <- event:
		default:
			close(sub.events)
			delete(h.subscribers, sub)
		}
	}
}

func (h *Hub) subscribe(tenantID string, topics []string) (*subscriber, error) {
	if len(topics) == 0 || len(topics) > maxTopics {
		return nil, errors.New("invalid topic count")
	}
	sub := &subscriber{tenant: tenantID, topics: make(map[string]struct{}, len(topics)), events: make(chan Event, bufferSize)}
	for _, topic := range topics {
		if !validTopic(topic) {
			return nil, errors.New("invalid topic")
		}
		sub.topics[topic] = struct{}{}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subscribers == nil {
		h.subscribers = make(map[*subscriber]struct{})
	}
	limit := h.MaxSubscribers
	if limit <= 0 {
		limit = defaultMaxSubscribers
	}
	if len(h.subscribers) >= limit {
		return nil, errors.New("subscriber capacity reached")
	}
	h.subscribers[sub] = struct{}{}
	return sub, nil
}
func (h *Hub) unsubscribe(sub *subscriber) { h.mu.Lock(); delete(h.subscribers, sub); h.mu.Unlock() }

// SubscriberCount returns the number of active subscribers on the hub.
func (h *Hub) SubscriberCount() int {
	if h == nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers)
}

// Subscription represents an active event subscription. Call Close when done.
type Subscription struct {
	hub *Hub
	sub *subscriber
	C   <-chan Event
}

// Close removes the subscriber from the hub and stops receiving events.
func (s *Subscription) Close() {
	if s != nil && s.hub != nil && s.sub != nil {
		s.hub.unsubscribe(s.sub)
	}
}

// SubscribeStream opens a realtime subscription channel for tenantID and the given topics.
func (h *Hub) SubscribeStream(tenantID string, topics []string) (*Subscription, error) {
	sub, err := h.subscribe(tenantID, topics)
	if err != nil {
		return nil, err
	}
	return &Subscription{
		hub: h,
		sub: sub,
		C:   sub.events,
	}, nil
}
func validTopic(topic string) bool {
	if len(topic) < 3 || len(topic) > 96 || strings.Count(topic, ":") != 1 {
		return false
	}
	for _, r := range topic {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == ':' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}
func tenantID(ctx context.Context) string { scope, _ := tenant.FromContext(ctx); return scope.TenantID }

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	tenantName := tenantID(r.Context())
	if h.RequireTenantScope {
		scope, err := tenant.Require(r.Context())
		if err != nil {
			http.Error(w, "subscription unavailable", http.StatusForbidden)
			return
		}
		tenantName = scope.TenantID
	}
	topics := strings.Split(r.URL.Query().Get("topics"), ",")
	for _, topic := range topics {
		if !validTopic(topic) || !canSubscribe(user.Role, topic) {
			http.Error(w, "invalid subscription", http.StatusForbidden)
			return
		}
	}
	sub, err := h.subscribe(tenantName, topics)
	if err != nil {
		http.Error(w, "subscription unavailable", http.StatusServiceUnavailable)
		return
	}
	defer h.unsubscribe(sub)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	controller := http.NewResponseController(w)
	write := func(data []byte) bool {
		_ = controller.SetWriteDeadline(time.Now().Add(30 * time.Second))
		_, e := w.Write(data)
		if e != nil {
			return false
		}
		flusher.Flush()
		return true
	}
	if !write([]byte(": connected\n\n")) {
		return
	}
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-sub.events:
			if !ok {
				return
			}
			payload, e := json.Marshal(event)
			if e != nil || !write([]byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event.Topic, payload))) {
				return
			}
		case <-ticker.C:
			if !write([]byte(": heartbeat\n\n")) {
				return
			}
		}
	}
}

func canSubscribe(role auth.Role, topic string) bool {
	if role == auth.RoleAdmin {
		return true
	}
	if role == auth.RoleEditor {
		return strings.HasPrefix(topic, "pages:") || strings.HasPrefix(topic, "news:") || strings.HasPrefix(topic, "forms:")
	}
	return role == auth.RoleViewer && (topic == "pages:published" || topic == "news:published")
}
