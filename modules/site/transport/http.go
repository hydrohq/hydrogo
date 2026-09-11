package transport

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type Route struct {
	Pattern string
	Public  bool
	Handler http.Handler
}

type Handler struct {
	DB                database.Executor
	Store             site.Store
	Service           *site.Service
	SubmissionLimiter security.RateLimiter
	AllowedForms      map[string]struct{}
	SitemapOrigin     string
	SitemapProvider   func(context.Context) ([]site.SitemapEntry, error)
}

func (h Handler) Routes() []Route {
	return []Route{
		{Pattern: "GET /site/settings", Public: true, Handler: http.HandlerFunc(h.publicSettings)},
		{Pattern: "GET /site/navigation/{locale}", Public: true, Handler: http.HandlerFunc(h.navigation)},
		{Pattern: "GET /sitemap.xml", Public: true, Handler: http.HandlerFunc(h.sitemap)},
		{Pattern: "POST /admin/site/settings", Handler: http.HandlerFunc(h.saveSetting)},
		{Pattern: "POST /admin/site/redirects", Handler: http.HandlerFunc(h.saveRedirect)},
		{Pattern: "POST /admin/site/navigation/{locale}", Handler: http.HandlerFunc(h.saveNavigation)},
		{Pattern: "GET /admin/site/inbox/{form}", Handler: http.HandlerFunc(h.inbox)},
		{Pattern: "POST /admin/site/inbox/{id}/status", Handler: http.HandlerFunc(h.inboxStatus)},
		{Pattern: "POST /forms/{form}/submit", Public: true, Handler: http.HandlerFunc(h.submitForm)},
	}
}

func (h Handler) inbox(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Store) {
		http.Error(w, "inbox unavailable", http.StatusServiceUnavailable)
		return
	}
	items, err := h.Store.ListSubmissions(r.Context(), h.DB, r.PathValue("form"), 100)
	if err != nil {
		http.Error(w, "inbox unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(items)
}

func (h Handler) inboxStatus(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil || site.ValidateInboxStatus(r.FormValue("status")) != nil {
		http.Error(w, "invalid inbox status", http.StatusBadRequest)
		return
	}
	if err := h.Service.SetSubmissionStatus(r.Context(), r.PathValue("id"), r.FormValue("status")); err != nil {
		http.Error(w, "inbox update failed", siteStatus(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) submitForm(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "form inbox unavailable", http.StatusServiceUnavailable)
		return
	}
	formKey := strings.TrimSpace(r.PathValue("form"))
	if _, ok := h.AllowedForms[formKey]; !ok {
		http.Error(w, "form not found", http.StatusNotFound)
		return
	}
	if h.SubmissionLimiter != nil && !h.SubmissionLimiter.Allow(formKey+":"+requestIP(r)) {
		security.WriteRateLimitedResponse(w, "too many form submissions", time.Minute)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	message := strings.TrimSpace(r.FormValue("message"))
	if email == "" || message == "" || len(message) > 32<<10 {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	payload, err := json.Marshal(map[string]string{"message": message, "name": strings.TrimSpace(r.FormValue("name"))})
	if err != nil {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	digest := sha256.Sum256([]byte(requestIP(r)))
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		http.Error(w, "form inbox unavailable", http.StatusServiceUnavailable)
		return
	}
	submission := site.Submission{ID: "form-" + hex.EncodeToString(idBytes), FormKey: formKey, Email: email, Payload: string(payload), SourceIPHash: hex.EncodeToString(digest[:])}
	if err := h.Service.Submit(r.Context(), submission); err != nil {
		http.Error(w, "form submission failed", siteStatus(err))
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func requestIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func (h Handler) saveSetting(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value site.Setting
	if !decodeJSON(w, r, &value) {
		return
	}
	if err := h.Service.SaveSetting(r.Context(), value); err != nil {
		http.Error(w, "invalid site setting", siteStatus(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) saveRedirect(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value site.Redirect
	if !decodeJSON(w, r, &value) {
		return
	}
	if err := h.Service.SaveRedirect(r.Context(), value); err != nil {
		http.Error(w, "invalid redirect", siteStatus(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) saveNavigation(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var items []site.NavigationItem
	if !decodeJSON(w, r, &items) {
		return
	}
	if err := h.Service.SaveNavigation(r.Context(), r.PathValue("locale"), items); err != nil {
		http.Error(w, "invalid navigation", siteStatus(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) admin(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return user != nil && user.Active && user.Role == auth.RoleAdmin
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := security.DecodeJSONStrict(w, r, target, 128<<10); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return false
	}
	return true
}

func siteStatus(err error) int {
	if errors.Is(err, site.ErrStoreRequired) || errors.Is(err, database.ErrDatabaseRequired) {
		return http.StatusServiceUnavailable
	}
	return http.StatusBadRequest
}

func (h Handler) sitemap(w http.ResponseWriter, r *http.Request) {
	if h.SitemapProvider == nil || h.SitemapOrigin == "" {
		http.Error(w, "sitemap unavailable", http.StatusServiceUnavailable)
		return
	}
	entries, err := h.SitemapProvider(r.Context())
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}
	data, err := site.RenderSitemap(entries)
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write(data)
}

func (h Handler) publicSettings(w http.ResponseWriter, r *http.Request) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Store) {
		http.Error(w, "site settings unavailable", http.StatusServiceUnavailable)
		return
	}
	settings, err := h.Store.ListPublicSettings(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "site settings unavailable", http.StatusInternalServerError)
		return
	}
	result := make(map[string]string, len(settings))
	for _, setting := range settings {
		result[setting.Key] = setting.Value
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(result)
}

func (h Handler) navigation(w http.ResponseWriter, r *http.Request) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Store) {
		http.Error(w, "navigation unavailable", http.StatusServiceUnavailable)
		return
	}
	items, err := h.Store.ListNavigation(r.Context(), h.DB, r.PathValue("locale"))
	if err != nil {
		http.Error(w, "navigation unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(items)
}
