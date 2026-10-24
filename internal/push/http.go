package push

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

type Handler struct {
	Dispatcher Dispatcher
	Limiter    security.RateLimiter
	Audit      auth.AuditRecorder
	// Logger is optional. When set, a failed audit write for a broadcast is
	// logged instead of silently discarded.
	Logger *slog.Logger
}

//go:embed assets/service-worker.js
var serviceWorkerSource []byte

//go:embed assets/push-client.js
var clientScriptSource []byte

func (h Handler) Routes() []router.Route {
	return []router.Route{
		{Pattern: "GET /push/service-worker.js", Access: router.Public, Handler: http.HandlerFunc(h.serviceWorker)},
		{Pattern: "GET /push/push-client.js", Access: router.Public, Handler: http.HandlerFunc(h.clientScript)},
		{Pattern: "GET /api/v1/push/config", Access: router.Public, Handler: http.HandlerFunc(h.config)},
		{Pattern: "POST /api/v1/push/web/subscription", Access: router.Protected, Handler: http.HandlerFunc(h.subscribeWeb)},
		{Pattern: "DELETE /api/v1/push/web/subscription", Access: router.Protected, Handler: http.HandlerFunc(h.unsubscribeWeb)},
		{Pattern: "POST /api/v1/push/mobile/subscription", Access: router.Protected, Handler: http.HandlerFunc(h.subscribeMobile)},
		{Pattern: "DELETE /api/v1/push/mobile/subscription", Access: router.Protected, Handler: http.HandlerFunc(h.unsubscribeMobile)},
		{Pattern: "POST " + AdminBroadcastPath, Access: router.Protected, Handler: http.HandlerFunc(h.broadcast)},
	}
}
func (h Handler) serviceWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Service-Worker-Allowed", "/")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(serviceWorkerSource))
}
func (h Handler) clientScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte(clientScriptSource))
}
func (h Handler) config(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"public_key": h.Dispatcher.PublicKey()})
}
func (h Handler) subscribeWeb(w http.ResponseWriter, r *http.Request) {
	user := activeUser(w, r)
	if user == nil {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	var raw json.RawMessage
	if !decodePushJSON(w, r, &raw) {
		http.Error(w, "invalid subscription", http.StatusBadRequest)
		return
	}
	var sub struct {
		Endpoint string `json:"endpoint"`
	}
	if json.Unmarshal(raw, &sub) != nil {
		http.Error(w, "invalid subscription", http.StatusBadRequest)
		return
	}
	if err := h.Dispatcher.Subscribe(r.Context(), Subscription{UserID: user.ID, Platform: PlatformWeb, Endpoint: sub.Endpoint, Keys: raw}); err != nil {
		http.Error(w, "subscription could not be saved", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h Handler) unsubscribeWeb(w http.ResponseWriter, r *http.Request) {
	user := activeUser(w, r)
	if user == nil {
		return
	}
	var sub struct {
		Endpoint string `json:"endpoint"`
	}
	if !decodePushJSON(w, r, &sub) || sub.Endpoint == "" {
		http.Error(w, "invalid subscription", http.StatusBadRequest)
		return
	}
	if err := h.Dispatcher.Unsubscribe(r.Context(), user.ID, PlatformWeb, sub.Endpoint); err != nil {
		http.Error(w, "subscription could not be removed", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h Handler) subscribeMobile(w http.ResponseWriter, r *http.Request) {
	user := activeUser(w, r)
	if user == nil {
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if !decodePushJSON(w, r, &req) {
		http.Error(w, "invalid device token", http.StatusBadRequest)
		return
	}
	if err := h.Dispatcher.Subscribe(r.Context(), Subscription{UserID: user.ID, Platform: PlatformMobile, Endpoint: req.Token}); err != nil {
		http.Error(w, "device could not be registered", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h Handler) unsubscribeMobile(w http.ResponseWriter, r *http.Request) {
	user := activeUser(w, r)
	if user == nil {
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if !decodePushJSON(w, r, &req) {
		http.Error(w, "invalid device token", http.StatusBadRequest)
		return
	}
	if err := h.Dispatcher.Unsubscribe(r.Context(), user.ID, PlatformMobile, req.Token); err != nil {
		http.Error(w, "device could not be removed", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AdminBroadcastPath sends a notification to every web and/or mobile
// subscriber; it is restricted to administrators and rate limited per user.
const AdminBroadcastPath = "/api/v1/admin/push/broadcast"

type broadcastRequest struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	URL      string `json:"url"`
	Audience string `json:"audience"`
}

func (h Handler) broadcast(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(w, r) {
		return
	}
	if h.Limiter != nil {
		user := auth.UserFromContext(r.Context())
		if !security.AllowForRequest(h.Limiter, r.Context(), user.ID) {
			security.WriteRateLimitedResponse(w, "push delivery is temporarily unavailable", time.Minute)
			return
		}
	}
	var input broadcastRequest
	if !api.DecodeJSON(w, r, &input, 8192) {
		return
	}
	if input.Audience == "" {
		input.Audience = "all"
	}
	message := Message{Title: strings.TrimSpace(input.Title), Body: strings.TrimSpace(input.Body), URL: strings.TrimSpace(input.URL)}
	result, err := h.Dispatcher.Broadcast(r.Context(), input.Audience, message)
	if err != nil {
		if errors.Is(err, ErrInvalidMessage) {
			fields := map[string]string{}
			if message.Title == "" {
				fields["title"] = "required"
			}
			if message.Body == "" {
				fields["body"] = "required"
			}
			if input.Audience != "all" && input.Audience != "web" && input.Audience != "mobile" {
				fields["audience"] = "invalid"
			}
			if len(fields) == 0 {
				fields["title"] = "invalid"
			}
			api.WriteValidation(w, r, fields)
			return
		}
		api.WriteError(w, r, http.StatusServiceUnavailable, "notification could not be sent", nil)
		return
	}
	if h.Audit != nil {
		if err := h.Audit.Record(r.Context(), "push.broadcast", strings.Join([]string{input.Audience, itoa(result.Sent), itoa(result.Delivered), itoa(result.Failed)}, ":")); err != nil && h.Logger != nil {
			h.Logger.ErrorContext(r.Context(), "audit write failed", "action", "push.broadcast", "error", err)
		}
	}
	api.WriteData(w, http.StatusOK, result, nil)
}
func activeUser(w http.ResponseWriter, r *http.Request) *auth.User {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		api.WriteError(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return nil
	}
	return user
}
func isAdmin(w http.ResponseWriter, r *http.Request) bool {
	user := activeUser(w, r)
	if user == nil {
		return false
	}
	if user.Role != auth.RoleAdmin {
		api.WriteError(w, r, http.StatusForbidden, "administrator role required", nil)
		return false
	}
	return true
}
func itoa(n int) string { return strconv.Itoa(n) }

func decodePushJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		return false
	}
	return decoder.Decode(&struct{}{}) == io.EOF
}
