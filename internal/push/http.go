package push

import (
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
)

type Handler struct {
	Dispatcher Dispatcher
	Limiter    security.RateLimiter
	Audit      auth.AuditRecorder
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
		{Pattern: "GET /auth/me/push", Access: router.Protected, Handler: http.HandlerFunc(h.settings)},
		{Pattern: "POST /api/v1/push/web/subscription", Access: router.Protected, Handler: http.HandlerFunc(h.subscribeWeb)},
		{Pattern: "DELETE /api/v1/push/web/subscription", Access: router.Protected, Handler: http.HandlerFunc(h.unsubscribeWeb)},
		{Pattern: "POST /api/v1/push/mobile/subscription", Access: router.Protected, Handler: http.HandlerFunc(h.subscribeMobile)},
		{Pattern: "DELETE /api/v1/push/mobile/subscription", Access: router.Protected, Handler: http.HandlerFunc(h.unsubscribeMobile)},
		{Pattern: "GET /admin/push", Access: router.Protected, Handler: http.HandlerFunc(h.adminPage)},
		{Pattern: "POST /admin/push/broadcast", Access: router.Protected, Handler: http.HandlerFunc(h.broadcast)},
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
func (h Handler) settings(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if ui.AdminStartForRequest(w, r, "Notifications", "Push settings", "settings") != nil {
		return
	}
	if err := ui.PushSettings(security.CSRFTokenFromContext(r.Context())).Render(r.Context(), w); err != nil {
		http.Error(w, "push settings unavailable", http.StatusInternalServerError)
		return
	}
	_ = ui.AdminEnd(w)
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
func (h Handler) adminPage(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(w, r) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if ui.AdminStartForRequest(w, r, "Push announcements", "Broadcast", "settings") != nil {
		return
	}
	sent, delivered, failed := 0, 0, 0
	result := false
	if values := strings.SplitN(r.URL.Query().Get("sent"), ",", 3); len(values) == 3 {
		sent, _ = strconv.Atoi(values[0])
		delivered, _ = strconv.Atoi(values[1])
		failed, _ = strconv.Atoi(values[2])
		result = true
	}
	_ = ui.PushBroadcast(security.CSRFTokenFromContext(r.Context()), sent, delivered, failed, result).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
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
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if r.ParseForm() != nil {
		http.Error(w, "invalid notification", http.StatusBadRequest)
		return
	}
	message := Message{Title: strings.TrimSpace(r.FormValue("title")), Body: strings.TrimSpace(r.FormValue("body")), URL: strings.TrimSpace(r.FormValue("url"))}
	result, err := h.Dispatcher.Broadcast(r.Context(), r.FormValue("audience"), message)
	if err != nil {
		http.Error(w, "notification could not be sent", http.StatusBadRequest)
		return
	}
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), "push.broadcast", strings.Join([]string{r.FormValue("audience"), itoa(result.Sent), itoa(result.Delivered), itoa(result.Failed)}, ":"))
	}
	http.Redirect(w, r, "/admin/push?sent="+url.QueryEscape(strings.TrimSpace(strings.Join([]string{itoa(result.Sent), itoa(result.Delivered), itoa(result.Failed)}, ","))), http.StatusSeeOther)
}
func activeUser(w http.ResponseWriter, r *http.Request) *auth.User {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
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
		http.Error(w, "forbidden", http.StatusForbidden)
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
