package transport

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	blockrender "github.com/ardaozyildirim/hydrogo/modules/pages/render"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

type Handler struct {
	DB      database.Executor
	Content service.ContentService
}

type pageInput struct {
	Locale string        `json:"locale"`
	Slug   string        `json:"slug"`
	Title  string        `json:"title"`
	Body   string        `json:"body"`
	Blocks []pages.Block `json:"blocks"`
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
		{Pattern: "GET /api/pages/by-slug", Handler: http.HandlerFunc(h.getPublishedByQuery)},
		{Pattern: "GET /public/pages/{locale}/{slug}/html", Handler: http.HandlerFunc(h.getPublishedHTML)},
		{Pattern: "GET /admin/pages/{id}", Handler: http.HandlerFunc(h.adminPage), Protected: true},
		{Pattern: "POST /admin/pages/{id}", Handler: http.HandlerFunc(h.adminUpdate), Protected: true},
		{Pattern: "POST /admin/pages/{id}/publish", Handler: http.HandlerFunc(h.adminPublish), Protected: true},
	}
}

func (h Handler) getPublished(w http.ResponseWriter, r *http.Request) {
	page, err := h.Content.GetPublished(r.Context(), h.DB, r.PathValue("locale"), r.PathValue("slug"))
	if err != nil {
		http.Error(w, "page lookup failed", pageReadStatus(err))
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h Handler) getPublishedHTML(w http.ResponseWriter, r *http.Request) {
	page, err := h.Content.GetPublished(r.Context(), h.DB, r.PathValue("locale"), r.PathValue("slug"))
	if err != nil {
		http.Error(w, "page lookup failed", pageReadStatus(err))
		return
	}
	blocks, err := blockrender.Blocks(page.Blocks)
	if err != nil {
		http.Error(w, "page render failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="%s"><head><meta charset="utf-8"><title>%s</title></head><body><main><h1>%s</h1><div class="content-body">%s</div>%s</main></body></html>`, template.HTMLEscapeString(page.Locale), template.HTMLEscapeString(page.Title), template.HTMLEscapeString(page.Title), template.HTMLEscapeString(page.Body), blocks)
}

func (h Handler) getPublishedByQuery(w http.ResponseWriter, r *http.Request) {
	h.getPublishedWithRoute(w, r, r.URL.Query().Get("locale"), r.URL.Query().Get("slug"))
}

func (h Handler) getPublishedWithRoute(w http.ResponseWriter, r *http.Request, locale, slug string) {
	page, err := h.Content.GetPublished(r.Context(), h.DB, locale, slug)
	if err != nil {
		http.Error(w, "page lookup failed", pageReadStatus(err))
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
	id, err := newID()
	if err != nil {
		http.Error(w, "page service unavailable", http.StatusServiceUnavailable)
		return
	}
	page := pages.Page{ID: id, Locale: input.Locale, Slug: input.Slug, Title: input.Title, Body: input.Body, Blocks: input.Blocks}
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
		http.Error(w, "page lookup failed", pageReadStatus(err))
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
	page := pages.Page{ID: r.PathValue("id"), Locale: input.Locale, Slug: input.Slug, Title: input.Title, Body: input.Body, Blocks: input.Blocks}
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
		http.Error(w, "page lookup failed", pageReadStatus(err))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	blockEditor := renderBlockEditor(page.Blocks, security.CSRFTokenFromContext(r.Context()))
	_, _ = w.Write([]byte(`<form method="post" action="/admin/pages/` + template.HTMLEscapeString(page.ID) + `">
<label>Locale <input name="locale" value="` + template.HTMLEscapeString(page.Locale) + `"></label>
<label>Slug <input name="slug" value="` + template.HTMLEscapeString(page.Slug) + `"></label>
<label>Title <input name="title" value="` + template.HTMLEscapeString(page.Title) + `"></label>
<label>Body <textarea name="body">` + template.HTMLEscapeString(page.Body) + `</textarea></label>
<fieldset><legend>Content blocks</legend>` + blockEditor + `</fieldset>
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
	var blocks []pages.Block
	if r.FormValue("block_count") != "" {
		blocks, err = parseBlockForm(r)
		if err != nil {
			http.Error(w, "invalid block fields", http.StatusBadRequest)
			return
		}
	} else if raw := r.FormValue("blocks"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &blocks); err != nil {
			http.Error(w, "invalid blocks JSON", http.StatusBadRequest)
			return
		}
	}
	page := pages.Page{ID: r.PathValue("id"), Locale: r.FormValue("locale"), Slug: r.FormValue("slug"), Title: r.FormValue("title"), Body: r.FormValue("body"), Blocks: blocks}
	if err := h.Content.UpdateDraft(r.Context(), page, version); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/pages/"+template.HTMLEscapeString(page.ID), http.StatusSeeOther)
}

func renderBlockEditor(blocks []pages.Block, csrf string) string {
	count := len(blocks)
	if count < 1 {
		count = 1
	}
	if count > 8 {
		count = 8
	}
	var out strings.Builder
	out.WriteString(`<input type="hidden" name="block_count" value="` + strconv.Itoa(count) + `">`)
	for i := 0; i < count; i++ {
		blockType, title, subtitle, buttonText, buttonURL, items := "", "", "", "", "", "[]"
		if i < len(blocks) {
			blockType, title, subtitle, buttonText, buttonURL, items = blockEditorValues(blocks[i])
		}
		fmt.Fprintf(&out, `<div class="block-editor-row"><strong>Block %d</strong><select name="block_type_%d"><option value="">Remove</option><option value="hero" %s>Hero</option><option value="faq" %s>FAQ</option><option value="cta" %s>CTA</option></select><input name="block_title_%d" value="%s" placeholder="Title"><input name="block_subtitle_%d" value="%s" placeholder="Subtitle"><input name="block_button_text_%d" value="%s" placeholder="Button text"><input name="block_button_url_%d" value="%s" placeholder="Button URL"><textarea name="block_items_%d" placeholder='[{"question":"Q","answer":"A"}]'>%s</textarea><input type="hidden" name="block_csrf_%d" value="%s"></div>`, i+1, i, selected(blockType, "hero"), selected(blockType, "faq"), selected(blockType, "cta"), i, template.HTMLEscapeString(title), i, template.HTMLEscapeString(subtitle), i, template.HTMLEscapeString(buttonText), i, template.HTMLEscapeString(buttonURL), i, template.HTMLEscapeString(items), i, template.HTMLEscapeString(csrf))
	}
	return out.String()
}

func selected(value, wanted string) string {
	if value == wanted {
		return `selected`
	}
	return ""
}

func blockEditorValues(block pages.Block) (string, string, string, string, string, string) {
	var data map[string]json.RawMessage
	if json.Unmarshal(block.Payload, &data) != nil {
		return block.Type, "", "", "", "", "[]"
	}
	get := func(key string) string { var value string; _ = json.Unmarshal(data[key], &value); return value }
	items := "[]"
	if raw, ok := data["items"]; ok {
		items = string(raw)
	}
	return block.Type, get("title"), get("subtitle"), get("button_text"), get("button_url"), items
}

func parseBlockForm(r *http.Request) ([]pages.Block, error) {
	count, err := strconv.Atoi(r.FormValue("block_count"))
	if err != nil || count < 0 || count > 8 {
		return nil, pages.ErrInvalidBlocks
	}
	blocks := make([]pages.Block, 0, count)
	for i := 0; i < count; i++ {
		typ := strings.TrimSpace(r.FormValue("block_type_" + strconv.Itoa(i)))
		if typ == "" {
			continue
		}
		payload := map[string]any{"title": r.FormValue("block_title_" + strconv.Itoa(i))}
		if typ == "faq" {
			var items []map[string]string
			if json.Unmarshal([]byte(r.FormValue("block_items_"+strconv.Itoa(i))), &items) != nil {
				return nil, pages.ErrInvalidBlocks
			}
			payload = map[string]any{"items": items}
		} else {
			payload["subtitle"] = r.FormValue("block_subtitle_" + strconv.Itoa(i))
			payload["button_text"] = r.FormValue("block_button_text_" + strconv.Itoa(i))
			payload["button_url"] = r.FormValue("block_button_url_" + strconv.Itoa(i))
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, pages.Block{Type: typ, Payload: raw})
	}
	if err := pages.ValidateBlocks(blocks); err != nil {
		return nil, err
	}
	return blocks, nil
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
	if errors.Is(err, service.ErrDependencyRequired) {
		return http.StatusServiceUnavailable
	}
	return http.StatusInternalServerError
}

func pageReadStatus(err error) int {
	if errors.Is(err, service.ErrDependencyRequired) {
		return http.StatusServiceUnavailable
	}
	return http.StatusNotFound
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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
