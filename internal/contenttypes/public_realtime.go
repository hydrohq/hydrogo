package contenttypes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type RealtimePayload struct {
	Action string         `json:"action"` // "create", "update", "delete"
	Type   string         `json:"type"`
	Record map[string]any `json:"record"`
}

func (p PublicAPI) dispatchRealtime(ctx context.Context, typeID, action string, record collections.Record) {
	if p.Hub == nil {
		return
	}
	topic := "content:" + typeID
	payload := RealtimePayload{
		Action: action,
		Type:   typeID,
		Record: p.present(typeID, "", record),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = p.Hub.Publish(ctx, realtime.Event{
		Topic: topic,
		Time:  time.Now().UTC(),
		Data:  data,
	})
}

func (p PublicAPI) Subscribe(w http.ResponseWriter, r *http.Request) {
	typeID := r.PathValue("type")
	definition, ok := p.definition(typeID)
	if !ok {
		api.WriteError(w, r, http.StatusNotFound, "not found", nil)
		return
	}
	if !definition.Realtime || p.Hub == nil {
		api.WriteError(w, r, http.StatusBadRequest, "realtime subscription is not enabled for this content type", nil)
		return
	}

	principal := p.principal(r)
	readRule := ""
	if definition.AccessRules != nil {
		readRule = definition.AccessRules.Read
		if readRule == "" {
			readRule = definition.AccessRules.List
		}
	}

	// Fast pre-flight check if rule strictly requires auth
	if readRule == "admin" && !principal.IsAdmin {
		api.WriteError(w, r, http.StatusForbidden, "subscription forbidden", nil)
		return
	}
	if (readRule == "member" || (strings.Contains(readRule, "@auth.") && !strings.Contains(readRule, "empty(@auth."))) && principal.Anonymous {
		api.WriteError(w, r, http.StatusUnauthorized, "authentication required", nil)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unavailable", http.StatusInternalServerError)
		return
	}

	tenantID := ""
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	}

	topic := "content:" + typeID
	sub, err := p.Hub.SubscribeStream(tenantID, []string{topic})
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "realtime stream unavailable", nil)
		return
	}
	defer sub.Close()

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("X-Accel-Buffering", "no")

	controller := http.NewResponseController(w)
	write := func(data []byte) bool {
		_ = controller.SetWriteDeadline(time.Now().Add(30 * time.Second))
		_, err := w.Write(data)
		if err != nil {
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
		case event, ok := <-sub.C:
			if !ok {
				return
			}
			// Access rule check per record: only deliver if caller is allowed to read this record
			if readRule != "" {
				var payload RealtimePayload
				if err := json.Unmarshal(event.Data, &payload); err == nil && payload.Record != nil {
					allowed, err := EvaluateRule(readRule, principal, payload.Record, true)
					if err != nil || !allowed {
						continue // skip delivering this record to this subscriber!
					}
				}
			}
			sseMsg := fmt.Sprintf("event: message\ndata: %s\n\n", event.Data)
			if !write([]byte(sseMsg)) {
				return
			}
		case <-ticker.C:
			if !write([]byte(": heartbeat\n\n")) {
				return
			}
		}
	}
}
