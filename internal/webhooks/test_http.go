package webhooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

type TestHandler struct {
	Dispatcher Dispatcher
	Endpoints  []Endpoint
}

func (h TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid test request", http.StatusBadRequest)
		return
	}
	endpointID := strings.TrimSpace(r.FormValue("endpoint_id"))
	topic := strings.TrimSpace(r.FormValue("topic"))
	var endpoint *Endpoint
	for i := range h.Endpoints {
		if h.Endpoints[i].ID == endpointID {
			endpoint = &h.Endpoints[i]
			break
		}
	}
	if endpoint == nil {
		http.Error(w, "webhook endpoint not found", http.StatusNotFound)
		return
	}
	if !strings.Contains(topic, ".") || len(topic) > 96 {
		http.Error(w, "invalid event type", http.StatusBadRequest)
		return
	}
	event := Event{ID: fmt.Sprintf("test-%s-%d", time.Now().UTC().Format("20060102T150405"), time.Now().UnixNano()), Topic: topic, PayloadVersion: 1, Data: map[string]string{"message": "HydroGo webhook delivery test", "initiated_by": user.ID}}
	result, err := h.Dispatcher.Test(r.Context(), *endpoint, event)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
	}
	_ = json.NewEncoder(w).Encode(struct {
		Result TestResult `json:"result"`
		Error  string     `json:"error,omitempty"`
	}{result, func() string {
		if err != nil {
			return "Webhook test failed"
		}
		return ""
	}()})
}

var _ http.Handler = TestHandler{}
