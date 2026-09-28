package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

const (
	bufferSize     = 16
	maxSubscribers = 256
	maxTopics      = 16
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
	h.mu.Lock()
	defer h.mu.Unlock()
	for sub := range h.subscribers {
		if sub.tenant != tenantID(ctx) {
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
	return nil
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
	if len(h.subscribers) >= maxSubscribers {
		return nil, errors.New("subscriber capacity reached")
	}
	h.subscribers[sub] = struct{}{}
	return sub, nil
}
func (h *Hub) unsubscribe(sub *subscriber) { h.mu.Lock(); delete(h.subscribers, sub); h.mu.Unlock() }
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
