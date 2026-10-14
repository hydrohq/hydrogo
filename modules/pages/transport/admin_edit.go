package transport

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	uicomponents "github.com/ardaozyildirim/hydrogo/internal/ui/components"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	blockrender "github.com/ardaozyildirim/hydrogo/modules/pages/render"
)

func (h Handler) adminPage(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	page, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		http.Error(w, "page lookup failed", pageReadStatus(err))
		return
	}
	if !editorAllowed(r) {
		h.renderReadOnlyPage(w, r, page)
		return
	}
	if locale := canonical.Locale(r.URL.Query().Get("compare")); locale != "" && locale != page.Locale {
		repository, ok := h.Content.Pages.(adminListRepository)
		if !ok {
			http.Error(w, "page unavailable", http.StatusServiceUnavailable)
			return
		}
		items, listErr := repository.ListAdmin(r.Context(), h.DB, 100)
		if listErr != nil {
			http.Error(w, "page unavailable", http.StatusServiceUnavailable)
			return
		}
		var translated pages.Page
		for _, item := range items {
			if item.Locale == locale && item.Slug == page.Slug {
				translated, listErr = h.Content.Get(r.Context(), h.DB, item.ID)
				break
			}
		}
		data := ui.TranslationEditorData{SourceTitle: page.Title, SourceBody: page.Body, Locale: locale, Action: "/admin/pages/" + url.PathEscape(page.ID) + "/translate", CSRF: security.CSRFTokenFromContext(r.Context()), Exists: translated.ID != "", Outdated: translated.ID != "" && page.UpdatedAt.After(translated.UpdatedAt), Version: translated.Version}
		data.TargetTitle, data.TargetBody = translated.Title, translated.Body
		renderAdminScreen(w, r, "Sayfa çevirisi", "Yan yana çeviri", ui.TranslationEditor(data))
		return
	}
	renderAdminScreen(w, r, "Sayfa düzenle", "İçerik stüdyosu", pageEditScreen(h.pageEditView(r, page)))
}

// pageEditView collects everything the page editor shows for page.
func (h Handler) pageEditView(r *http.Request, page pages.Page) pageEditView {
	csrf := security.CSRFTokenFromContext(r.Context())
	base := "/admin/pages/" + url.PathEscape(page.ID)
	view := pageEditView{
		ID:            page.ID,
		Action:        base,
		PublishAction: base + "/publish",
		CSRF:          csrf,
		Version:       page.Version,
		BlockCount:    len(page.Blocks),
		Blocks:        newBlockEditorView(page.Blocks, csrf),
	}
	view.Fields = writerComponent(func(out io.Writer) error {
		if err := adminstudio.RenderCreateForm(out, adminstudio.FormData{
			Collection: h.AdminCollection,
			Action:     "/admin/pages/" + page.ID,
			CSRFToken:  csrf,
			HideSubmit: true,
			Values:     map[string]string{"locale": page.Locale, "slug": page.Slug, "title": page.Title, "body": page.Body},
			UILocale:   ui.AdminLocaleFromRequest(r),
		}); err != nil {
			return err
		}
		return uicomponents.Input(uicomponents.InputProps{ID: "page-tags", Name: "tags", Type: "text", Label: pageTagsLabel, Value: strings.Join(page.Tags, ", ")}).Render(r.Context(), out)
	})
	if h.Presence != nil {
		view.Before = append(view.Before, designsystem.EditPresence(page.ID, base+"/presence", csrf))
	}
	if h.PreviewSigner != nil {
		view.Before = append(view.Before,
			designsystem.DeviceViewportPreview(designsystem.DeviceViewportPreviewProps{Endpoint: base + "/preview-link", CSRF: csrf}),
			ui.DraftPreviewLinkForm(base+"/preview-link", csrf))
	}
	view.Before = append(view.Before, h.revisionPanels(r, page, base, csrf)...)
	if metrics := h.faqFeedbackMetrics(r, page.Blocks); len(metrics) > 0 {
		view.Before = append(view.Before, designsystem.FAQFeedbackReport(metrics))
	}
	if h.assistantEnabled() {
		view.After = append(view.After, designsystem.AIAssistantRequest(base+"/ai-suggest", csrf, h.SupportedLocales))
		if suggestion, ok := r.Context().Value(pageSuggestionKey{}).(pageSuggestion); ok {
			view.After = append(view.After, designsystem.AIAssistantSuggestion(suggestion.BeforeTitle, suggestion.BeforeBody, suggestion.Result, base+"/ai-applied", csrf))
		}
	}
	return view
}

// revisionPanels compares the current draft with the requested (or latest)
// stored revision, visually and field by field.
func (h Handler) revisionPanels(r *http.Request, page pages.Page, base, csrf string) []templ.Component {
	history, ok := h.Content.Pages.(interface {
		ListRevisions(context.Context, database.Executor, string, int) ([]pages.RevisionSnapshot, error)
	})
	if !ok {
		return nil
	}
	revisions, err := history.ListRevisions(r.Context(), h.DB, page.ID, 50)
	if err != nil || len(revisions) == 0 {
		return nil
	}
	selected := revisions[0]
	if requested, parseErr := strconv.Atoi(r.URL.Query().Get("revision")); parseErr == nil {
		for _, revision := range revisions {
			if revision.Revision == requested {
				selected = revision
				break
			}
		}
	}
	var panels []templ.Component
	beforePreview, beforeErr := h.revisionPreview(r.Context(), selected.Title, selected.Body, selected.Blocks)
	afterPreview, afterErr := h.revisionPreview(r.Context(), page.Title, page.Body, page.Blocks)
	if beforeErr == nil && afterErr == nil {
		panels = append(panels, pageRevisionCompare(beforePreview, afterPreview, "Sürüm "+strconv.Itoa(selected.Revision), "Geçerli taslak"))
	}
	changes := adminstudio.CompareFields(map[string]string{"Başlık": selected.Title, "Metin": selected.Body}, map[string]string{"Başlık": page.Title, "Metin": page.Body})
	editor := selected.ActorEmail
	if editor == "" {
		if selected.ActorID == "" {
			editor = "Önceki sürüm"
		} else {
			editor = "Yönetici"
		}
	}
	restore := base + "/revisions/" + strconv.Itoa(selected.Revision) + "/restore"
	return append(panels, ui.RevisionDiffPanel(changes, editor, selected.CreatedAt.Format("02.01.2006 15:04"), restore, csrf))
}

func (h Handler) renderReadOnlyPage(w http.ResponseWriter, r *http.Request, page pages.Page) {
	preview, err := h.revisionPreview(r.Context(), page.Title, page.Body, page.Blocks)
	if err != nil {
		http.Error(w, "page preview unavailable", http.StatusServiceUnavailable)
		return
	}
	view := pageReadOnlyView{Title: page.Title, Preview: preview, Fields: []uicomponents.InputProps{
		{ID: "page-title-readonly", Name: "title", Label: "Başlık", Value: page.Title, ReadOnly: true},
		{ID: "page-slug-readonly", Name: "slug", Label: "Adres", Value: page.Slug, ReadOnly: true},
		{ID: "page-locale-readonly", Name: "locale", Label: "Dil", Value: page.Locale, ReadOnly: true},
		{ID: "page-status-readonly", Name: "status", Label: "Durum", Value: string(page.Status), ReadOnly: true},
	}}
	renderAdminScreen(w, r, "Sayfa görüntüle", "Salt okunur içerik", uicomponents.Card(uicomponents.CardProps{Title: "Sayfa içeriği", Class: "studio-card", Body: pageReadOnlyBody(view)}))
}

func (h Handler) faqFeedbackMetrics(r *http.Request, blocks []pages.Block) []designsystem.FAQFeedbackMetric {
	if !h.AnalyticsEnabled || h.Feedback == nil {
		return nil
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	} else if h.RequireTenantScope {
		return nil
	}
	var result []designsystem.FAQFeedbackMetric
	for _, block := range blocks {
		if block.Type != "faq" {
			continue
		}
		var payload struct {
			Items []designsystem.FAQItem `json:"items"`
		}
		if json.Unmarshal(block.Payload, &payload) != nil || len(payload.Items) == 0 {
			continue
		}
		ids := make([]string, 0, len(payload.Items))
		for index, item := range payload.Items {
			ids = append(ids, designsystem.FAQFeedbackID(item, index))
		}
		counts, err := h.Feedback.FeedbackCounts(r.Context(), tenantID, "faq", ids)
		if err != nil {
			continue
		}
		for index, item := range payload.Items {
			value := counts[ids[index]]
			result = append(result, designsystem.FAQFeedbackMetric{Question: item.Question, Up: value.Up, Down: value.Down})
		}
	}
	return result
}

// revisionPreview renders a standalone preview document for a sandboxed frame.
func (h Handler) revisionPreview(ctx context.Context, title, body string, blocks []pages.Block) (string, error) {
	prepared := make([]pages.Block, len(blocks))
	for i, block := range blocks {
		resolved, err := h.withPublicMedia(ctx, block)
		if err != nil {
			return "", err
		}
		prepared[i] = resolved
	}
	rendered, err := blockrender.Blocks(prepared)
	if err != nil {
		return "", err
	}
	return renderString(ctx, pagePreviewDocument(title, ui.RichTextHTML(body), rendered))
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
	page := pages.Page{ID: r.PathValue("id"), Locale: locale, Slug: r.FormValue("slug"), Title: r.FormValue("title"), Body: r.FormValue("body"), Blocks: blocks, Tags: strings.Split(r.FormValue("tags"), ",")}
	if err := h.validatePageMedia(r.Context(), page); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Content.UpdateDraft(r.Context(), page, version); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/pages/"+url.PathEscape(page.ID)+"?saved=1", http.StatusSeeOther)
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
	oldLocale, oldSlug, oldErr := h.Content.PublishedSlug(r.Context(), h.DB, r.PathValue("id"))
	if oldErr != nil {
		http.Error(w, "Yayımlanan eski bağlantı alınamadı.", http.StatusServiceUnavailable)
		return
	}
	page, pageErr := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if pageErr != nil {
		http.Error(w, "Sayfa bilgisi alınamadı.", http.StatusServiceUnavailable)
		return
	}
	if err := h.Content.Publish(r.Context(), r.PathValue("id"), version); err != nil {
		http.Error(w, "Sayfa yayımlanamadı. İçeriği kontrol edip yeniden deneyin.", pageStatus(err))
		return
	}
	if oldSlug != "" {
		if err := h.recordPreviousPublishedSlug(r.Context(), oldLocale, oldSlug, page); err != nil {
			http.Error(w, "Sayfa yayımlandı ancak eski bağlantısı yönlendirilemedi.", http.StatusServiceUnavailable)
			return
		}
	}
	if user := auth.UserFromContext(r.Context()); user != nil && h.Notifications != nil {
		_ = h.Notifications.Add(r.Context(), h.DB, user.ID, "Sayfa yayına alındı", "Sayfanız ziyaretçilere açıldı.", "/admin/pages")
	}
	http.Redirect(w, r, "/admin/pages/"+url.PathEscape(r.PathValue("id")), http.StatusSeeOther)
}
