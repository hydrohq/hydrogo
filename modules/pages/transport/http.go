package transport

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strconv"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

type Handler struct {
	DB      database.Executor
	Content service.ContentService
}

type pageInput struct {
	Locale string `json:"locale"`
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type updateInput struct {
	pageInput
	Version int `json:"version"`
}

func (h Handler) Routes() []Route {
	return []Route{
		{Pattern: "POST /api/pages", Handler: http.HandlerFunc(h.create), Protected: true},
		{Pattern: "GET /api/pages/{id}", Handler: http.HandlerFunc(h.get), Protected: true},
		{Pattern: "GET /api/pages/{id}/preview", Handler: http.HandlerFunc(h.preview), Protected: true},
		{Pattern: "PUT /api/pages/{id}", Handler: http.HandlerFunc(h.update), Protected: true},
		{Pattern: "POST /api/pages/{id}/publish", Handler: http.HandlerFunc(h.publish), Protected: true},
		{Pattern: "GET /public/pages/{locale}/{slug}", Handler: http.HandlerFunc(h.getPublished)},
		{Pattern: "GET /admin/pages/{id}", Handler: http.HandlerFunc(h.adminPage), Protected: true},
		{Pattern: "POST /admin/pages/{id}", Handler: http.HandlerFunc(h.adminUpdate), Protected: true},
		{Pattern: "POST /admin/pages/{id}/publish", Handler: http.HandlerFunc(h.adminPublish), Protected: true},
	}
}

func (h Handler) getPublished(w http.ResponseWriter, r *http.Request) {
	page, err := h.Content.GetPublished(r.Context(), h.DB, r.PathValue("locale"), r.PathValue("slug"))
	if err != nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

type Route struct {
	Pattern   string
	Handler   http.Handler
	Protected bool
}

func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var input pageInput
	if err := decodeJSON(w, r, &input); err != nil {
		return
	}
	page := pages.Page{ID: newID(), Locale: input.Locale, Slug: input.Slug, Title: input.Title, Body: input.Body}
	if err := h.Content.CreateDraft(r.Context(), page); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	h.writePage(w, r, page.ID)
}

func (h Handler) get(w http.ResponseWriter, r *http.Request) {
	h.writePage(w, r, r.PathValue("id"))
}

func (h Handler) preview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	h.writePage(w, r, r.PathValue("id"))
}

func (h Handler) writePage(w http.ResponseWriter, r *http.Request, id string) {
	page, err := h.Content.Get(r.Context(), h.DB, id)
	if err != nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h Handler) update(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var input updateInput
	if err := decodeJSON(w, r, &input); err != nil {
		return
	}
	page := pages.Page{ID: r.PathValue("id"), Locale: input.Locale, Slug: input.Slug, Title: input.Title, Body: input.Body}
	if err := h.Content.UpdateDraft(r.Context(), page, input.Version); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	h.writePage(w, r, page.ID)
}

func (h Handler) publish(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	expectedVersion, err := strconv.Atoi(r.URL.Query().Get("version"))
	if err != nil || expectedVersion <= 0 {
		http.Error(w, "version query parameter is required", http.StatusBadRequest)
		return
	}
	if err := h.Content.Publish(r.Context(), r.PathValue("id"), expectedVersion); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	h.writePage(w, r, r.PathValue("id"))
}

func (h Handler) adminPage(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	page, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<form method="post" action="/admin/pages/` + template.HTMLEscapeString(page.ID) + `">
<label>Locale <input name="locale" value="` + template.HTMLEscapeString(page.Locale) + `"></label>
<label>Slug <input name="slug" value="` + template.HTMLEscapeString(page.Slug) + `"></label>
<label>Title <input name="title" value="` + template.HTMLEscapeString(page.Title) + `"></label>
<label>Body <textarea name="body">` + template.HTMLEscapeString(page.Body) + `</textarea></label>
<input type="hidden" name="_csrf" value="` + template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())) + `">
<input type="hidden" name="version" value="` + strconv.Itoa(page.Version) + `">
<button type="submit">Save draft</button>
</form>`))
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
	page := pages.Page{ID: r.PathValue("id"), Locale: r.FormValue("locale"), Slug: r.FormValue("slug"), Title: r.FormValue("title"), Body: r.FormValue("body")}
	if err := h.Content.UpdateDraft(r.Context(), page, version); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/pages/"+template.HTMLEscapeString(page.ID), http.StatusSeeOther)
}

func (h Handler) adminPublish(w http.ResponseWriter, r *http.Request) {
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
	if err := h.Content.Publish(r.Context(), r.PathValue("id"), version); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/pages/"+template.HTMLEscapeString(r.PathValue("id")), http.StatusSeeOther)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	if err := security.DecodeJSONStrict(w, r, target, 1<<20); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return err
	}
	return nil
}

func pageError(err error) string {
	if errors.Is(err, service.ErrInvalidPage) {
		return "invalid page"
	}
	if errors.Is(err, service.ErrConflict) {
		return "page was changed by another editor"
	}
	if errors.Is(err, pages.ErrRouteConflict) {
		return "published route is already in use"
	}
	return "page operation failed"
}

func pageStatus(err error) int {
	if errors.Is(err, service.ErrInvalidPage) {
		return http.StatusBadRequest
	}
	if errors.Is(err, service.ErrConflict) {
		return http.StatusConflict
	}
	if errors.Is(err, pages.ErrRouteConflict) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "page-generated"
	}
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func editorAllowed(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return user != nil && user.Active && (user.Role == auth.RoleAdmin || user.Role == auth.RoleEditor)
}
