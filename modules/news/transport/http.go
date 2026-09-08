package transport

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

type Route struct {
	Pattern   string
	Handler   http.Handler
	Protected bool
}

type Handler struct {
	Service news.Service
	DB      database.Executor
}

type input struct {
	Locale string `json:"locale"`
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type updateInput struct {
	input
	Version int `json:"version"`
}

func (h Handler) Routes() []Route {
	return []Route{
		{Pattern: "POST /api/news", Protected: true, Handler: http.HandlerFunc(h.create)},
		{Pattern: "GET /api/news/{id}", Protected: true, Handler: http.HandlerFunc(h.get)},
		{Pattern: "PUT /api/news/{id}", Protected: true, Handler: http.HandlerFunc(h.update)},
		{Pattern: "POST /api/news/{id}/schedule", Protected: true, Handler: http.HandlerFunc(h.schedule)},
		{Pattern: "POST /api/news/{id}/publish", Protected: true, Handler: http.HandlerFunc(h.publish)},
		{Pattern: "POST /api/news/{id}/retract", Protected: true, Handler: http.HandlerFunc(h.retract)},
		{Pattern: "GET /public/news/{locale}/{slug}", Handler: http.HandlerFunc(h.publicGet)},
	}
}

func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value input
	if !decodeJSON(w, r, &value) {
		return
	}
	item := news.Item{ID: newID(), Locale: value.Locale, Slug: value.Slug, Title: value.Title, Body: value.Body, Status: news.StatusDraft}
	if err := h.Service.CreateDraft(r.Context(), item); err != nil {
		writeError(w, err)
		return
	}
	h.write(w, r, item.ID)
}

func (h Handler) get(w http.ResponseWriter, r *http.Request) { h.write(w, r, r.PathValue("id")) }

func (h Handler) update(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value updateInput
	if !decodeJSON(w, r, &value) {
		return
	}
	item := news.Item{ID: r.PathValue("id"), Locale: value.Locale, Slug: value.Slug, Title: value.Title, Body: value.Body, Status: news.StatusDraft}
	if err := h.Service.UpdateDraft(r.Context(), item, value.Version); err != nil {
		writeError(w, err)
		return
	}
	h.write(w, r, item.ID)
}

func (h Handler) schedule(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value struct {
		Version     int    `json:"version"`
		ScheduledAt string `json:"scheduled_at"`
	}
	if !decodeJSON(w, r, &value) {
		return
	}
	at, err := time.Parse(time.RFC3339, value.ScheduledAt)
	if err != nil {
		http.Error(w, "invalid schedule", http.StatusBadRequest)
		return
	}
	if err := h.Service.Schedule(r.Context(), r.PathValue("id"), value.Version, at); err != nil {
		writeError(w, err)
		return
	}
	h.write(w, r, r.PathValue("id"))
}

func (h Handler) publish(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	version, err := strconv.Atoi(r.URL.Query().Get("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	if err := h.Service.Publish(r.Context(), r.PathValue("id"), version); err != nil {
		writeError(w, err)
		return
	}
	h.write(w, r, r.PathValue("id"))
}

func (h Handler) retract(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	version, err := strconv.Atoi(r.URL.Query().Get("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	if err := h.Service.Retract(r.Context(), r.PathValue("id"), version); err != nil {
		writeError(w, err)
		return
	}
	h.write(w, r, r.PathValue("id"))
}

func (h Handler) publicGet(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	item, err := h.Service.Repository.GetPublishedByRoute(r.Context(), h.DB, r.PathValue("locale"), r.PathValue("slug"))
	if err != nil {
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h Handler) write(w http.ResponseWriter, r *http.Request, id string) {
	if h.DB == nil {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	item, err := h.Service.Repository.Get(r.Context(), h.DB, id)
	if err != nil {
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func editorAllowed(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return user != nil && user.Active && (user.Role == auth.RoleAdmin || user.Role == auth.RoleEditor)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, news.ErrInvalidItem) {
		status = http.StatusBadRequest
	}
	if errors.Is(err, news.ErrConflict) || errors.Is(err, news.ErrScheduleStale) {
		status = http.StatusConflict
	}
	http.Error(w, "news operation failed", status)
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "news-generated"
	}
	return hex.EncodeToString(bytes)
}
