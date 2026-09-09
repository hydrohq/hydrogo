package transport

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
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
		{Pattern: "GET /admin/news/{id}", Protected: true, Handler: http.HandlerFunc(h.adminPage)},
		{Pattern: "POST /admin/news/{id}", Protected: true, Handler: http.HandlerFunc(h.adminUpdate)},
		{Pattern: "POST /admin/news/{id}/schedule", Protected: true, Handler: http.HandlerFunc(h.adminSchedule)},
		{Pattern: "POST /admin/news/{id}/publish", Protected: true, Handler: http.HandlerFunc(h.adminPublish)},
		{Pattern: "POST /admin/news/{id}/retract", Protected: true, Handler: http.HandlerFunc(h.adminRetract)},
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
	id, err := newID()
	if err != nil {
		http.Error(w, "news service unavailable", http.StatusServiceUnavailable)
		return
	}
	item := news.Item{ID: id, Locale: value.Locale, Slug: value.Slug, Title: value.Title, Body: value.Body, Status: news.StatusDraft}
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
	if database.RequireExecutor(h.DB) != nil || h.Service.Repository == nil {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	item, err := h.Service.Repository.GetPublishedByRoute(r.Context(), h.DB, r.PathValue("locale"), r.PathValue("slug"))
	if err != nil {
		if errors.Is(err, database.ErrDatabaseRequired) {
			http.Error(w, "news unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h Handler) adminPage(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if database.RequireExecutor(h.DB) != nil || h.Service.Repository == nil {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	item, err := h.Service.Repository.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		if errors.Is(err, database.ErrDatabaseRequired) {
			http.Error(w, "news unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<form method="post" action="/admin/news/` + template.HTMLEscapeString(item.ID) + `">
<label>Locale <input name="locale" value="` + template.HTMLEscapeString(item.Locale) + `"></label>
<label>Slug <input name="slug" value="` + template.HTMLEscapeString(item.Slug) + `"></label>
<label>Title <input name="title" value="` + template.HTMLEscapeString(item.Title) + `"></label>
<label>Body <textarea name="body">` + template.HTMLEscapeString(item.Body) + `</textarea></label>
<input type="hidden" name="_csrf" value="` + template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())) + `">
<input type="hidden" name="version" value="` + strconv.Itoa(item.Version) + `">
<button type="submit">Save draft</button>
</form>
<form method="post" action="/admin/news/` + template.HTMLEscapeString(item.ID) + `/schedule"><input type="hidden" name="_csrf" value="` + template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())) + `"><input type="hidden" name="version" value="` + strconv.Itoa(item.Version) + `"><input name="scheduled_at" type="datetime-local"><button type="submit">Schedule</button></form>
<form method="post" action="/admin/news/` + template.HTMLEscapeString(item.ID) + `/publish"><input type="hidden" name="_csrf" value="` + template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())) + `"><input type="hidden" name="version" value="` + strconv.Itoa(item.Version) + `"><button type="submit">Publish</button></form>
<form method="post" action="/admin/news/` + template.HTMLEscapeString(item.ID) + `/retract"><input type="hidden" name="_csrf" value="` + template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())) + `"><input type="hidden" name="version" value="` + strconv.Itoa(item.Version) + `"><button type="submit">Retract</button></form>`))
}

func (h Handler) adminUpdate(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	version, err := strconv.Atoi(r.FormValue("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	item := news.Item{ID: r.PathValue("id"), Locale: r.FormValue("locale"), Slug: r.FormValue("slug"), Title: r.FormValue("title"), Body: r.FormValue("body"), Status: news.StatusDraft}
	if err := h.Service.UpdateDraft(r.Context(), item, version); err != nil {
		writeError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/news/"+template.HTMLEscapeString(item.ID), http.StatusSeeOther)
}

func (h Handler) adminSchedule(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	version, err := strconv.Atoi(r.FormValue("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	at, err := time.Parse("2006-01-02T15:04", r.FormValue("scheduled_at"))
	if err != nil {
		http.Error(w, "invalid schedule", http.StatusBadRequest)
		return
	}
	if err := h.Service.Schedule(r.Context(), r.PathValue("id"), version, at); err != nil {
		writeError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/news/"+template.HTMLEscapeString(r.PathValue("id")), http.StatusSeeOther)
}

func (h Handler) adminPublish(w http.ResponseWriter, r *http.Request) {
	h.adminTransition(w, r, "publish")
}

func (h Handler) adminRetract(w http.ResponseWriter, r *http.Request) {
	h.adminTransition(w, r, "retract")
}

func (h Handler) adminTransition(w http.ResponseWriter, r *http.Request, action string) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	version, err := strconv.Atoi(r.FormValue("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	if action == "publish" {
		err = h.Service.Publish(r.Context(), r.PathValue("id"), version)
	} else {
		err = h.Service.Retract(r.Context(), r.PathValue("id"), version)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/news/"+template.HTMLEscapeString(r.PathValue("id")), http.StatusSeeOther)
}

func (h Handler) write(w http.ResponseWriter, r *http.Request, id string) {
	if database.RequireExecutor(h.DB) != nil || h.Service.Repository == nil {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	item, err := h.Service.Repository.Get(r.Context(), h.DB, id)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseRequired) {
			http.Error(w, "news unavailable", http.StatusServiceUnavailable)
			return
		}
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
	if err := security.DecodeJSONStrict(w, r, target, 1<<20); err != nil {
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
	if errors.Is(err, news.ErrRepositoryRequired) || errors.Is(err, database.ErrDatabaseRequired) {
		http.Error(w, "news service unavailable", http.StatusServiceUnavailable)
		return
	}
	status := http.StatusInternalServerError
	if errors.Is(err, news.ErrInvalidItem) {
		status = http.StatusBadRequest
	}
	if errors.Is(err, news.ErrConflict) || errors.Is(err, news.ErrScheduleStale) {
		status = http.StatusConflict
	}
	http.Error(w, "news operation failed", status)
}

func newID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
