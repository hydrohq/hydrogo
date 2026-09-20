package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	themev1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	blockrender "github.com/ardaozyildirim/hydrogo/modules/pages/render"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

type Handler struct {
	DB      database.Executor
	Content service.ContentService
	Theme   *themev1.ThemeBundle
	Media   interface {
		Get(context.Context, database.Executor, string) (media.Asset, error)
	}
	AdminCollection   adminstudio.Collection
	DefaultLocale     string
	AllowDemoFrontend bool
}

func (h Handler) defaultLocale() string {
	if h.DefaultLocale != "" {
		return h.DefaultLocale
	}
	return "tr"
}

func (h Handler) publicPageAllowed(page pages.Page) bool {
	return h.AllowDemoFrontend || !strings.HasPrefix(page.ID, "demo-")
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

// pageSummary is the intentionally small public discovery projection. Detail
// content remains behind the by-slug endpoint so list clients do not mistake
// empty draft fields for published content.
type pageSummary struct {
	ID                    string    `json:"id"`
	Locale                string    `json:"locale"`
	Slug                  string    `json:"slug"`
	Title                 string    `json:"title"`
	Revision              int       `json:"revision"`
	PublicationGeneration int64     `json:"publication_generation"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (h Handler) Routes() []Route {
	return []Route{
		{Pattern: "POST /api/pages", Handler: http.HandlerFunc(h.create), Protected: true},
		{Pattern: "GET /api/pages", Handler: http.HandlerFunc(h.listPublished)},
		{Pattern: "GET /api/pages/{id}", Handler: http.HandlerFunc(h.get), Protected: true},
		{Pattern: "GET /api/pages/{id}/preview", Handler: http.HandlerFunc(h.preview), Protected: true},
		{Pattern: "PUT /api/pages/{id}", Handler: http.HandlerFunc(h.update), Protected: true},
		{Pattern: "POST /api/pages/{id}/publish", Handler: http.HandlerFunc(h.publish), Protected: true},
		{Pattern: "GET /public/pages/{locale}/{slug}", Handler: http.HandlerFunc(h.getPublished)},
		{Pattern: "GET /api/pages/by-slug", Handler: http.HandlerFunc(h.getPublishedByQuery)},
		{Pattern: "GET /public/pages/{locale}/{slug}/html", Handler: http.HandlerFunc(h.getPublishedHTML)},
		{Pattern: "GET /admin/pages/new", Handler: http.HandlerFunc(h.adminNewPage), Protected: true},
		{Pattern: "GET /admin/pages/{id}", Handler: http.HandlerFunc(h.adminPage), Protected: true},
		{Pattern: "GET /admin/pages", Handler: http.HandlerFunc(h.adminList), Protected: true},
		{Pattern: "POST /admin/pages", Handler: http.HandlerFunc(h.adminCreateForm), Protected: true},
		{Pattern: "POST /admin/pages/{id}", Handler: http.HandlerFunc(h.adminUpdate), Protected: true},
		{Pattern: "POST /admin/pages/{id}/publish", Handler: http.HandlerFunc(h.adminPublish), Protected: true},
	}
}

type adminListRepository interface {
	ListAdmin(context.Context, database.Executor, int) ([]pages.Page, error)
}

func (h Handler) adminList(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Content.Pages) {
		http.Error(w, "page unavailable", http.StatusServiceUnavailable)
		return
	}
	repository, ok := h.Content.Pages.(adminListRepository)
	if !ok {
		http.Error(w, "page admin list unavailable", http.StatusServiceUnavailable)
		return
	}
	items, err := repository.ListAdmin(r.Context(), h.DB, 100)
	if err != nil {
		http.Error(w, "page unavailable", http.StatusServiceUnavailable)
		return
	}
	rows := make([]adminstudio.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, adminstudio.Row{Values: map[string]string{"locale": item.Locale, "slug": item.Slug, "title": item.Title, "status": string(item.Status)}, Actions: []adminstudio.Action{{Label: "Düzenle", Href: "/admin/pages/" + item.ID}}})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStart(w, "Sayfalar", "", security.NonceFromContext(r.Context()), "pages"); err != nil {
		return
	}
	_, _ = fmt.Fprint(w, `<section class="studio-card"><div class="table-wrap">`)
	if err := adminstudio.RenderList(w, adminstudio.ListData{Collection: h.AdminCollection, Rows: rows}); err != nil {
		http.Error(w, "page admin unavailable", http.StatusInternalServerError)
		return
	}
	_, _ = fmt.Fprint(w, `</div></section>`)
	_ = ui.AdminEnd(w)
}

func (h Handler) adminNewPage(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStart(w, "Yeni sayfa", "Sayfa yönetimi", security.NonceFromContext(r.Context()), "pages"); err != nil {
		return
	}
	_, _ = fmt.Fprint(w, `<section class="studio-card"><form class="studio-form" method="post" action="/admin/pages">`)
	if err := adminstudio.RenderCreateForm(w, adminstudio.FormData{
		Collection:  h.AdminCollection,
		Action:      "/admin/pages",
		CSRFToken:   security.CSRFTokenFromContext(r.Context()),
		Nonce:       security.NonceFromContext(r.Context()),
		Values:      map[string]string{"locale": "tr"},
		SubmitLabel: "Taslak oluştur",
	}); err != nil {
		return
	}
	_, _ = fmt.Fprint(w, `</form></section>`)
	_ = ui.AdminEnd(w)
}

func (h Handler) adminCreateForm(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	id, err := newID()
	if err != nil {
		http.Error(w, "page service unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := r.FormValue("locale")
	if locale == "" {
		locale = "tr"
	}
	page := pages.Page{
		ID:     id,
		Locale: locale,
		Slug:   r.FormValue("slug"),
		Title:  r.FormValue("title"),
		Body:   r.FormValue("body"),
		Status: pages.StatusDraft,
	}
	if err := h.Content.CreateDraft(r.Context(), page); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/pages/"+url.PathEscape(id), http.StatusSeeOther)
}

func (h Handler) listPublished(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			http.Error(w, "limit must be between 1 and 100", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	items, err := h.Content.ListPublished(r.Context(), h.DB, r.URL.Query().Get("locale"), limit)
	if err != nil {
		http.Error(w, "page lookup failed", pageReadStatus(err))
		return
	}
	visible := items[:0]
	for _, item := range items {
		if h.publicPageAllowed(item) {
			visible = append(visible, item)
		}
	}
	writeJSON(w, http.StatusOK, summarizePages(visible))
}

func summarizePages(items []pages.Page) []pageSummary {
	summaries := make([]pageSummary, 0, len(items))
	for _, page := range items {
		summaries = append(summaries, pageSummary{ID: page.ID, Locale: page.Locale, Slug: page.Slug, Title: page.Title, Revision: page.Revision, PublicationGeneration: page.PublicationGeneration, UpdatedAt: page.UpdatedAt})
	}
	return summaries
}

func (h Handler) getPublished(w http.ResponseWriter, r *http.Request) {
	page, err := h.Content.GetPublished(r.Context(), h.DB, r.PathValue("locale"), r.PathValue("slug"))
	if err != nil {
		http.Error(w, "page lookup failed", pageReadStatus(err))
		return
	}
	if !h.publicPageAllowed(page) {
		http.NotFound(w, r)
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
	if !h.publicPageAllowed(page) {
		http.NotFound(w, r)
		return
	}
	blocks, err := blockrender.Blocks(page.Blocks)
	if err != nil {
		http.Error(w, "page render failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if h.Theme != nil {
		var content strings.Builder
		fmt.Fprintf(&content, `<div data-testid="public-page"><h1>%s</h1><div class="content-body">%s</div></div>`, template.HTMLEscapeString(page.Title), ui.RichTextHTML(page.Body))
		for _, block := range page.Blocks {
			block, err = h.withPublicMedia(r.Context(), block)
			if err != nil {
				http.Error(w, "page media lookup failed", http.StatusInternalServerError)
				return
			}
			var themed strings.Builder
			if err := h.Theme.RenderBlock(&themed, block); err == nil {
				content.WriteString(themed.String())
			} else if errors.Is(err, themev1.ErrThemeBlockUnavailable) {
				fallback, fallbackErr := blockrender.Blocks([]pages.Block{block})
				if fallbackErr != nil {
					http.Error(w, "page render failed", http.StatusInternalServerError)
					return
				}
				content.WriteString(fallback)
			} else {
				http.Error(w, "page render failed", http.StatusInternalServerError)
				return
			}
		}
		if err := h.Theme.RenderPage(w, themev1.PageTemplateData{Title: page.Title, Slug: page.Slug, Locale: page.Locale, Direction: canonical.Direction(page.Locale), Content: template.HTML(content.String()), Slots: map[string]template.HTML{"content": template.HTML(content.String())}}); err != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
		}
		return
	}
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="%s" dir="%s"><head><meta charset="utf-8"><title>%s</title></head><body><main data-testid="public-page"><h1>%s</h1><div class="content-body">%s</div>%s</main></body></html>`, template.HTMLEscapeString(page.Locale), canonical.Direction(page.Locale), template.HTMLEscapeString(page.Title), template.HTMLEscapeString(page.Title), ui.RichTextHTML(page.Body), blocks)
}

func (h Handler) withPublicMedia(ctx context.Context, block pages.Block) (pages.Block, error) {
	if h.Media == nil || block.Type != "hero" {
		return block, nil
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(block.Payload, &payload); err != nil {
		return block, err
	}
	var assetID string
	if err := json.Unmarshal(payload["media_asset_id"], &assetID); err != nil || assetID == "" {
		return block, nil
	}
	asset, err := h.Media.Get(ctx, h.DB, assetID)
	if err != nil {
		return block, nil
	}
	if asset.Status != "ready" || asset.Visibility != media.VisibilityPublic {
		return block, nil
	}
	mediaURL := "/media/" + asset.ID
	if provider, ok := h.Media.(interface{ PublicURL(string) (string, error) }); ok {
		if publicURL, urlErr := provider.PublicURL(asset.StorageKey); urlErr == nil && publicURL != "" {
			mediaURL = publicURL
		}
	}
	encodedURL, err := json.Marshal(mediaURL)
	if err != nil {
		return block, err
	}
	payload["media_url"] = encodedURL
	encoded, err := json.Marshal(payload)
	if err != nil {
		return block, err
	}
	block.Payload = encoded
	return block, nil
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
	if !h.publicPageAllowed(page) {
		http.NotFound(w, r)
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
	var form bytes.Buffer
	if err := adminstudio.RenderCreateForm(&form, adminstudio.FormData{Collection: h.AdminCollection, Action: "/admin/pages/" + page.ID, CSRFToken: security.CSRFTokenFromContext(r.Context()), Nonce: security.NonceFromContext(r.Context()), HideSubmit: true, Values: map[string]string{"locale": page.Locale, "slug": page.Slug, "title": page.Title, "body": page.Body}}); err != nil {
		http.Error(w, "page admin unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStart(w, "Sayfa düzenle", "İçerik stüdyosu", security.NonceFromContext(r.Context()), "pages"); err != nil {
		return
	}
	blockEditor := renderBlockEditor(page.Blocks, security.CSRFTokenFromContext(r.Context()))
	_, _ = fmt.Fprintf(w, `<section class="studio-card"><form class="studio-form" method="post" action="/admin/pages/%s">%s<div class="block-editor"><div class="block-editor-heading"><div><h3>İçerik blokları</h3><p>Sayfanın görsel bölümlerini ve sıralamasını yönetin.</p></div><span>%d blok</span></div><fieldset>%s</fieldset></div><input type="hidden" name="version" value="%d"><button type="submit">Taslağı kaydet</button></form></section><section class="studio-card"><h2>Yayınlama</h2><p class="muted">Taslağı kaydedin veya doğrudan yayına alın.</p><div class="studio-actions"><form method="post" action="/admin/pages/%s/publish"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="version" value="%d"><button type="submit" class="button button-success">Yayına Al</button></form></div></section>`, template.HTMLEscapeString(page.ID), form.String(), len(page.Blocks), blockEditor, page.Version, template.HTMLEscapeString(page.ID), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), page.Version)
	_ = ui.AdminEnd(w)
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
	locale := r.FormValue("locale")
	if locale == "" {
		locale = h.defaultLocale()
	}
	page := pages.Page{ID: r.PathValue("id"), Locale: locale, Slug: r.FormValue("slug"), Title: r.FormValue("title"), Body: r.FormValue("body"), Blocks: blocks}
	if err := h.Content.UpdateDraft(r.Context(), page, version); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/pages/"+url.PathEscape(page.ID), http.StatusSeeOther)
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
		blockType, title, subtitle, buttonText, buttonURL, mediaAssetID, items := "", "", "", "", "", "", "[]"
		if i < len(blocks) {
			blockType, title, subtitle, buttonText, buttonURL, mediaAssetID, items = blockEditorValues(blocks[i])
		}
		fmt.Fprintf(&out, `<div class="block-editor-row" data-block-row data-block-type="%s"><strong>Block %d</strong><label class="block-field block-field-type" data-block-field="type">Blok türü<select name="block_type_%d"><option value="">Remove</option><option value="hero" %s>Hero</option><option value="faq" %s>FAQ</option><option value="cta" %s>CTA</option><option value="pricing_table" %s>Pricing table</option><option value="testimonials" %s>Testimonials</option></select></label><label class="block-field" data-block-field="title">Başlık<input name="block_title_%d" value="%s" placeholder="Başlık"></label><label class="block-field" data-block-field="subtitle">Alt başlık<input name="block_subtitle_%d" value="%s" placeholder="Alt başlık"></label><div class="block-field media-field" data-block-field="media"><label>Medya asset ID<input id="block_media_asset_id_%d" name="block_media_asset_id_%d" value="%s" placeholder="Medyadan seçin"></label><button class="button button-quiet media-picker-open" type="button" data-media-target="block_media_asset_id_%d">Medyadan seç</button></div><label class="block-field" data-block-field="buttonText">Buton metni<input name="block_button_text_%d" value="%s" placeholder="Buton metni"></label><label class="block-field" data-block-field="buttonURL">Buton URL<input name="block_button_url_%d" value="%s" placeholder="/shop"></label><label class="block-field" data-block-field="items">Items JSON<textarea name="block_items_%d" placeholder='[{"question":"Soru","answer":"Cevap"}]'>%s</textarea></label><input type="hidden" name="block_csrf_%d" value="%s"></div>`, template.HTMLEscapeString(blockType), i+1, i, selected(blockType, "hero"), selected(blockType, "faq"), selected(blockType, "cta"), selected(blockType, "pricing_table"), selected(blockType, "testimonials"), i, template.HTMLEscapeString(title), i, template.HTMLEscapeString(subtitle), i, i, template.HTMLEscapeString(mediaAssetID), i, i, template.HTMLEscapeString(buttonText), i, template.HTMLEscapeString(buttonURL), i, template.HTMLEscapeString(items), i, template.HTMLEscapeString(csrf))
	}
	return out.String()
}

func selected(value, wanted string) string {
	if value == wanted {
		return `selected`
	}
	return ""
}

func renderMediaPicker(nonce string) string {
	escapedNonce := template.HTMLEscapeString(nonce)
	return `<style nonce="` + escapedNonce + `">.media-field{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:8px;align-items:end}.media-field label{min-width:0}.media-field .button{padding:10px 12px;white-space:nowrap}@media(max-width:620px){.media-field{grid-template-columns:1fr}.media-field .button{width:100%}}</style><script nonce="` + escapedNonce + `">(()=>{if(window.__hydroPagesMediaBound)return;window.__hydroPagesMediaBound=true;document.addEventListener('click',event=>{const opener=event.target.closest('.media-picker-open');if(opener){const target=document.getElementById(opener.dataset.mediaTarget);if(!target)return;if(typeof window.HydroOpenMediaPicker==='function'){window.HydroOpenMediaPicker(assetId=>{target.value=assetId;target.dispatchEvent(new Event('input',{bubbles:true}));target.dispatchEvent(new Event('change',{bubbles:true}));});}}});})()</script>`
}

func blockEditorValues(block pages.Block) (string, string, string, string, string, string, string) {
	var data map[string]json.RawMessage
	if json.Unmarshal(block.Payload, &data) != nil {
		return block.Type, "", "", "", "", "", "[]"
	}
	get := func(key string) string { var value string; _ = json.Unmarshal(data[key], &value); return value }
	items := "[]"
	if raw, ok := data["items"]; ok {
		items = string(raw)
	}
	return block.Type, get("title"), get("subtitle"), get("button_text"), get("button_url"), get("media_asset_id"), items
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
		if mediaAssetID := strings.TrimSpace(r.FormValue("block_media_asset_id_" + strconv.Itoa(i))); mediaAssetID != "" {
			payload["media_asset_id"] = mediaAssetID
		}
		if typ == "faq" {
			var items []map[string]string
			if json.Unmarshal([]byte(r.FormValue("block_items_"+strconv.Itoa(i))), &items) != nil {
				return nil, pages.ErrInvalidBlocks
			}
			payload = map[string]any{"items": items}
		} else if typ == "pricing_table" || typ == "testimonials" {
			var items []json.RawMessage
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
	http.Redirect(w, r, "/admin/pages/"+url.PathEscape(r.PathValue("id")), http.StatusSeeOther)
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
	return auth.Can(user, auth.PermissionContentWrite)
}
