package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	stdsql "database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	uicomponents "github.com/ardaozyildirim/hydrogo/internal/ui/components"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/news"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type Route struct {
	Pattern   string
	Handler   http.Handler
	Protected bool
}

type Handler struct {
	Service     news.Service
	DB          database.Executor
	Media       publicMediaURLProvider
	MediaLookup interface {
		Get(context.Context, database.Executor, string) (media.Asset, error)
	}
	AdminCollection    adminstudio.Collection
	DefaultLocale      string
	PublicOrigin       string
	NavigationProvider func(context.Context, string) ([]v1.ThemeNavigationItem, error)
	ContentViews       interface {
		ContentViewCounts(context.Context, string, string, []string) (map[string]int64, error)
	}
	Feedback interface {
		FeedbackTotals(context.Context, string, string, string) (analytics.FeedbackTotals, error)
		FeedbackCounts(context.Context, string, string, []string) (map[string]analytics.FeedbackTotals, error)
	}
	RequireTenantScope    bool
	AnalyticsEnabled      bool
	PreviewSigner         *preview.Signer
	PublicSupport         adminstudio.SupportConfig
	Presence              *realtime.Presence
	PublicContactSettings func(context.Context) ([]site.Setting, error)
	// Theme renders the public news pages; nil uses the built-in views.
	Theme *v1.ThemeBundle
}

type publicMediaURLProvider interface {
	PublicURL(string) (string, error)
}

const (
	maxNewsTaxonomyFormBytes = 16 << 10
	maxNewsItemFormBytes     = 256 << 10
	maxNewsWorkflowFormBytes = 4 << 10
)

func (h Handler) defaultLocale() string {
	if h.DefaultLocale != "" {
		return h.DefaultLocale
	}
	return "tr"
}

type publishedListRepository interface {
	ListPublished(context.Context, database.Executor, string, int) ([]news.Item, error)
}

type relatedListRepository interface {
	ListRelatedPublished(context.Context, database.Executor, string, string, int) ([]news.Item, error)
}

type adminListRepository interface {
	ListAdmin(context.Context, database.Executor, int) ([]news.Item, error)
}

type taxonomyAdminRepository interface {
	ListTaxonomy(context.Context, database.Executor, string) ([]v1.TaxonomyTerm, error)
	ListTaxonomyTerms(context.Context, database.Executor) ([]v1.TaxonomyTerm, error)
	ReplaceTaxonomy(context.Context, database.Executor, string, []v1.TaxonomyTerm) error
	UpsertTaxonomyTerm(context.Context, database.Executor, v1.TaxonomyTerm) error
	DeleteTaxonomyTerm(context.Context, database.Executor, string) error
}

type transactionStarter interface {
	BeginTx(context.Context, *stdsql.TxOptions) (*stdsql.Tx, error)
}

type input struct {
	Locale         string     `json:"locale"`
	Slug           string     `json:"slug"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	CoverMediaID   string     `json:"cover_media_id"`
	Excerpt        string     `json:"excerpt"`
	KeyTakeaways   string     `json:"key_takeaways"`
	Category       string     `json:"category"`
	AuthorName     string     `json:"author_name"`
	Featured       bool       `json:"featured"`
	Tags           string     `json:"tags"`
	SEOTitle       string     `json:"seo_title"`
	SEODescription string     `json:"seo_description"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

type updateInput struct {
	input
	Version int `json:"version"`
}

func (h Handler) Routes() []Route {
	routes := []Route{
		{Pattern: "POST /api/news", Protected: true, Handler: http.HandlerFunc(h.create)},
		{Pattern: "GET /api/news/{id}", Protected: true, Handler: http.HandlerFunc(h.get)},
		{Pattern: "PUT /api/news/{id}", Protected: true, Handler: http.HandlerFunc(h.update)},
		{Pattern: "POST /api/news/{id}/schedule", Protected: true, Handler: http.HandlerFunc(h.schedule)},
		{Pattern: "POST /api/news/{id}/publish", Protected: true, Handler: http.HandlerFunc(h.publish)},
		{Pattern: "POST /api/news/{id}/retract", Protected: true, Handler: http.HandlerFunc(h.retract)},
		{Pattern: "GET /api/news", Handler: http.HandlerFunc(h.publicList)},
		{Pattern: "GET /api/news/by-slug", Handler: http.HandlerFunc(h.publicGetByQuery)},
		{Pattern: "GET /admin/news", Protected: true, Handler: http.HandlerFunc(h.adminList)},
		{Pattern: "GET /admin/news/new", Protected: true, Handler: http.HandlerFunc(h.adminNewPage)},
		{Pattern: "GET /admin/news/taxonomy", Protected: true, Handler: http.HandlerFunc(h.adminTaxonomyCatalog)},
		{Pattern: "POST /admin/news", Protected: true, Handler: http.HandlerFunc(h.adminCreate)},
		{Pattern: "POST /admin/news/taxonomy", Protected: true, Handler: http.HandlerFunc(h.adminTaxonomyTerm)},
		{Pattern: "POST /admin/news/taxonomy/{id}/delete", Protected: true, Handler: http.HandlerFunc(h.adminTaxonomyTermDelete)},
		{Pattern: "GET /admin/news/{id}", Protected: true, Handler: http.HandlerFunc(h.adminPage)},
		{Pattern: "POST /admin/news/{id}", Protected: true, Handler: http.HandlerFunc(h.adminUpdate)},
		{Pattern: "POST /admin/news/{id}/taxonomy", Protected: true, Handler: http.HandlerFunc(h.adminTaxonomy)},
		{Pattern: "POST /admin/news/{id}/schedule", Protected: true, Handler: http.HandlerFunc(h.adminSchedule)},
		{Pattern: "POST /admin/news/{id}/publish", Protected: true, Handler: http.HandlerFunc(h.adminPublish)},
		{Pattern: "POST /admin/news/{id}/retract", Protected: true, Handler: http.HandlerFunc(h.adminRetract)},
		{Pattern: "GET /public/news/{locale}/{slug}/html", Handler: http.HandlerFunc(h.publicGetHTML)},
		{Pattern: "GET /public/news/{locale}/html", Handler: http.HandlerFunc(h.publicListHTML)},
		{Pattern: "GET /public/news/{locale}/{slug}", Handler: http.HandlerFunc(h.publicGet)},
		{Pattern: "POST /admin/news/{id}/preview", Protected: true, Handler: http.HandlerFunc(h.adminPreview)},
	}
	if h.Presence != nil {
		routes = append(routes,
			Route{Pattern: "POST /admin/news/{id}/presence", Protected: true, Handler: http.HandlerFunc(h.editPresenceHeartbeat)},
			Route{Pattern: "GET /admin/news/{id}/presence", Protected: true, Handler: http.HandlerFunc(h.editPresenceList)},
		)
	}
	return routes
}

func (h Handler) adminList(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	repository, ok := h.Service.Repository.(adminListRepository)
	if !ok {
		http.Error(w, "news admin list unavailable", http.StatusServiceUnavailable)
		return
	}
	items, err := repository.ListAdmin(r.Context(), h.DB, 100)
	if err != nil {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	viewCounts := map[string]int64{}
	viewsAvailable := h.ContentViews != nil && h.AnalyticsEnabled
	if viewsAvailable && len(items) > 0 {
		ids := make([]string, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		tenantID := "default"
		if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
			tenantID = scope.TenantID
		}
		var countsErr error
		viewCounts, countsErr = h.ContentViews.ContentViewCounts(r.Context(), tenantID, "news", ids)
		viewsAvailable = countsErr == nil
	}
	if viewsAvailable {
		sortNewsByViews(items, viewCounts, r.URL.Query().Get("sort"))
	}
	feedbackCounts := map[string]analytics.FeedbackTotals{}
	feedbackAvailable := h.Feedback != nil && len(items) > 0
	if feedbackAvailable {
		ids := make([]string, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		tenantID := "default"
		if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
			tenantID = scope.TenantID
		} else if h.RequireTenantScope {
			feedbackAvailable = false
		}
		if feedbackAvailable {
			var countsErr error
			feedbackCounts, countsErr = h.Feedback.FeedbackCounts(r.Context(), tenantID, "news", ids)
			feedbackAvailable = countsErr == nil
		}
	}
	uiLocale := ui.AdminLocaleFromRequest(r)
	rows := make([]adminstudio.Row, 0, len(items))
	for _, item := range items {
		title := item.Title
		if h.MediaLookup != nil {
			if content, contentErr := h.Service.Repository.Get(r.Context(), h.DB, item.ID); contentErr == nil {
				references := media.MissingReferences(content.Body+content.CoverMediaID, nil)
				missing := 0
				for _, id := range references {
					if _, getErr := h.MediaLookup.Get(r.Context(), h.DB, id); getErr != nil {
						missing++
					}
				}
				if missing > 0 {
					title += fmt.Sprintf(" ⚠ "+i18n.T(uiLocale, "admin.content.missing_images"), missing)
				}
			}
		}
		label := i18n.T(uiLocale, "admin.generated.list.view")
		if editorAllowed(r) {
			label = i18n.T(uiLocale, "admin.generated.list.edit")
		}
		actions := []adminstudio.Action{{Label: label, Href: "/admin/news/" + item.ID}}
		counts := feedbackCounts[item.ID]
		values := map[string]string{"locale": item.Locale, "slug": item.Slug, "title": title, "status": string(item.Status), "views": strconv.FormatInt(viewCounts[item.ID], 10), "helpful": strconv.FormatInt(counts.Up, 10), "not_helpful": strconv.FormatInt(counts.Down, 10)}
		rows = append(rows, adminstudio.Row{Values: values, ValueHTML: map[string]template.HTML{"views": renderViewCountBadge(r.Context(), viewCounts[item.ID])}, Status: adminstudio.StatusInfo{Value: string(item.Status), ScheduledAt: item.ScheduledAt}, Actions: actions})
	}
	collection := h.AdminCollection
	if viewsAvailable {
		collection.Fields = append(append([]adminstudio.Field(nil), collection.Fields...), adminstudio.Field{ID: "views", Label: "Views", Kind: adminstudio.FieldText, VisibleInList: true})
	}
	if feedbackAvailable {
		collection.Fields = append(collection.Fields, adminstudio.Field{ID: "helpful", Label: "Helpful", Kind: adminstudio.FieldText, VisibleInList: true}, adminstudio.Field{ID: "not_helpful", Label: "Not helpful", Kind: adminstudio.FieldText, VisibleInList: true})
	}
	if r.URL.Query().Get("fragment") == "table" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if viewsAvailable {
			_ = designsystem.ContentViewSort(designsystem.ContentViewSortProps{Path: "/admin/news", Status: r.URL.Query().Get("status"), Selected: r.URL.Query().Get("sort")}).Render(r.Context(), w)
		}
		if err := adminstudio.RenderList(w, adminstudio.ListData{
			Collection:        collection,
			Rows:              rows,
			StatusFilter:      r.URL.Query().Get("status"),
			UILocale:          uiLocale,
			ListCaption:       fmt.Sprintf(i18n.T(uiLocale, "admin.generated.list.caption"), collection.Label),
			ActionColumnLabel: i18n.T(uiLocale, "admin.generated.list.actions"),
		}); err != nil {
			http.Error(w, "news admin unavailable", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Haberler", "", "news"); err != nil {
		return
	}
	if err := ui.ListViewSwitcher("admin-content-list", "hydro-view-content").Render(r.Context(), w); err != nil {
		return
	}
	_, _ = fmt.Fprint(w, `<section class="studio-card"><div class="table-wrap" id="admin-content-list" data-view-mode="table">`)
	if viewsAvailable {
		_ = designsystem.ContentViewSort(designsystem.ContentViewSortProps{Path: "/admin/news", Status: r.URL.Query().Get("status"), Selected: r.URL.Query().Get("sort")}).Render(r.Context(), w)
	}
	if err := adminstudio.RenderList(w, adminstudio.ListData{
		Collection:        collection,
		Rows:              rows,
		StatusFilter:      r.URL.Query().Get("status"),
		UILocale:          uiLocale,
		ListCaption:       fmt.Sprintf(i18n.T(uiLocale, "admin.generated.list.caption"), collection.Label),
		ActionColumnLabel: i18n.T(uiLocale, "admin.generated.list.actions"),
	}); err != nil {
		http.Error(w, "news admin unavailable", http.StatusInternalServerError)
		return
	}
	_, _ = fmt.Fprintf(w, `</div></section>`)
	_ = ui.AdminEnd(w)
}

func sortNewsByViews(items []news.Item, counts map[string]int64, direction string) {
	if direction != "views" && direction != "-views" {
		return
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := counts[items[i].ID], counts[items[j].ID]
		if a == b {
			return items[i].ID < items[j].ID
		}
		if direction == "views" {
			return a < b
		}
		return a > b
	})
}

func renderViewCountBadge(ctx context.Context, count int64) template.HTML {
	var out strings.Builder
	if err := designsystem.ViewCountBadge(designsystem.ViewCountBadgeProps{Count: count}).Render(ctx, &out); err != nil {
		return ""
	}
	return template.HTML(out.String())
}

func (h Handler) adminNewPage(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Yeni haber", "Yayın merkezi", "news"); err != nil {
		return
	}
	fieldMap := make(map[string]adminstudio.Field)
	for _, f := range h.AdminCollection.Fields {
		fieldMap[f.ID] = f
	}
	var leftFields, rightMetaFields, rightSEOFields []adminstudio.Field
	for _, id := range []string{"title", "slug", "cover_media_id", "excerpt", "key_takeaways", "body"} {
		if f, ok := fieldMap[id]; ok {
			leftFields = append(leftFields, f)
		}
	}
	for _, id := range []string{"featured", "author_name", "category", "tags", "expires_at"} {
		if f, ok := fieldMap[id]; ok {
			rightMetaFields = append(rightMetaFields, f)
		}
	}
	for _, id := range []string{"seo_title", "seo_description"} {
		if f, ok := fieldMap[id]; ok {
			rightSEOFields = append(rightSEOFields, f)
		}
	}
	if !hasAdminField(rightMetaFields, "expires_at") {
		rightMetaFields = append(rightMetaFields, adminstudio.Field{ID: "expires_at", Label: "Otomatik yayından kalkma", Kind: adminstudio.FieldDate, Widget: "datetime-local"})
	}
	values := map[string]string{"locale": h.defaultLocale(), "expires_at": ""}
	csrf := security.CSRFTokenFromContext(r.Context())
	uiLocale := ui.AdminLocaleFromRequest(r)

	var leftBuf bytes.Buffer
	leftCol := adminstudio.Collection{ID: "news_left", Label: "Haber İçeriği", Fields: leftFields, Actions: []adminstudio.Action{{Label: "Kaydet", Href: "/admin/news"}}}
	if err := adminstudio.RenderCreateForm(&leftBuf, adminstudio.FormData{Collection: leftCol, Action: "/admin/news", CSRFToken: csrf, Values: values, HideSubmit: true, UILocale: uiLocale}); err != nil {
		return
	}

	var rightMetaBuf bytes.Buffer
	rightMetaCol := adminstudio.Collection{ID: "news_meta", Label: "Genel Bilgiler", Fields: rightMetaFields, Actions: []adminstudio.Action{{Label: "Kaydet", Href: "/admin/news"}}}
	if err := adminstudio.RenderCreateForm(&rightMetaBuf, adminstudio.FormData{Collection: rightMetaCol, Action: "/admin/news", CSRFToken: csrf, Values: values, HideSubmit: true, UILocale: uiLocale}); err != nil {
		return
	}

	var rightSEOBuf bytes.Buffer
	rightSEOCol := adminstudio.Collection{ID: "news_seo", Label: "SEO & Arama Motoru", Fields: rightSEOFields, Actions: []adminstudio.Action{{Label: "Kaydet", Href: "/admin/news"}}}
	if err := adminstudio.RenderCreateForm(&rightSEOBuf, adminstudio.FormData{Collection: rightSEOCol, Action: "/admin/news", CSRFToken: csrf, Values: values, HideSubmit: true, UILocale: uiLocale}); err != nil {
		return
	}

	_, _ = fmt.Fprintf(w, `<div class="editor-top-nav"><div class="editor-top-info"><strong class="editor-live-title">[Başlıksız]</strong><span class="editor-live-sub">Yeni bir Haber oluşturuluyor</span></div><div class="editor-top-actions"><button type="submit" form="news-form" class="button">Taslağı kaydet</button></div></div><form id="news-form" class="studio-form" method="post" action="/admin/news"><div class="editor-split-container"><div class="editor-main-col"><section class="editor-card"><div class="editor-card-title"><span>Haber İçeriği</span></div>%s</section></div><div class="editor-side-col"><section class="editor-card"><div class="editor-card-title"><span>Yayın Bilgileri</span><span class="card-tag">YENİ</span></div><div class="editor-side-fields"><div class="editor-side-status"><span>Durum</span><strong>Taslak</strong></div><div class="editor-side-status"><span>Dil</span><strong>%s</strong></div><p class="muted" style="margin:8px 0 0;font-size:.78rem">Kaydedildikten sonra zamanlama ve yayına alma butonları aktifleşir.</p></div></section><section class="editor-card"><div class="editor-card-title"><span>Genel Bilgiler</span></div><div class="editor-side-fields">%s</div></section><section class="editor-card"><div class="editor-card-title"><span>SEO &amp; Arama Motoru</span></div><div class="editor-side-fields">%s</div></section></div></div><input type="hidden" name="locale" value="%s"><input type="hidden" name="_csrf" value="%s"></form><script>(()=>{const titleInp=document.querySelector('#news-form input[name="title"]'),liveTitle=document.querySelector('.editor-live-title');if(titleInp&&liveTitle){titleInp.addEventListener('input',()=>{liveTitle.textContent=titleInp.value.trim()||'[Başlıksız]';});}})();</script>`, template.HTML(leftBuf.String()), template.HTMLEscapeString(h.defaultLocale()), template.HTML(rightMetaBuf.String()), template.HTML(rightSEOBuf.String()), template.HTMLEscapeString(h.defaultLocale()), template.HTMLEscapeString(csrf))
	_ = ui.AdminEnd(w)
}

func (h Handler) adminTaxonomyCatalog(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	repository, ok := h.Service.Repository.(taxonomyAdminRepository)
	if !ok || database.RequireExecutor(h.DB) != nil {
		http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
		return
	}
	terms, err := repository.ListTaxonomyTerms(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Taxonomy", "Haber sınıflandırması", "news"); err != nil {
		return
	}
	_, _ = fmt.Fprint(w, `<section class="studio-card"><h2>Yeni terim</h2><p class="muted">Etiket ve kategorilerde çoklu dil desteği için etiketleri JSON olarak girin.</p><form class="studio-form" method="post" action="/admin/news/taxonomy"><label>ID<input name="id" required maxlength="128"></label><label>Slug<input name="slug" required maxlength="128" pattern="[a-z0-9][a-z0-9-]*"></label><label>Tür<select name="kind"><option value="category">Kategori</option><option value="tag">Etiket</option></select></label><label>Üst terim ID<input name="parent" maxlength="128"></label><label>Etiketler JSON<textarea name="labels_json" required placeholder='{"tr":"Haber","en":"News"}'></textarea></label><input type="hidden" name="_csrf" value="`+template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context()))+`"><button type="submit">Terimi kaydet</button></form></section>`)
	_, _ = fmt.Fprint(w, `<section class="studio-card"><h2>Terim kataloğu</h2><div class="table-wrap"><table><thead><tr><th>ID</th><th>Slug</th><th>Tür</th><th>Üst terim</th><th>Etiketler</th><th>İşlem</th></tr></thead><tbody>`)
	for _, term := range terms {
		labels, _ := json.Marshal(term.Labels)
		_, _ = fmt.Fprintf(w, `<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td><code>%s</code></td><td><form method="post" action="/admin/news/taxonomy/%s/delete"><input type="hidden" name="_csrf" value="%s"><button type="submit" class="button button-quiet button-danger">Sil</button></form></td></tr>`, template.HTMLEscapeString(term.ID), template.HTMLEscapeString(term.Slug), template.HTMLEscapeString(term.Kind), template.HTMLEscapeString(term.Parent), template.HTMLEscapeString(string(labels)), template.URLQueryEscaper(term.ID), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())))
	}
	_, _ = fmt.Fprint(w, `</tbody></table></div></section>`)
	_ = ui.AdminEnd(w)
}

func (h Handler) adminTaxonomyTerm(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxNewsTaxonomyFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	repository, ok := h.Service.Repository.(taxonomyAdminRepository)
	if !ok || database.RequireExecutor(h.DB) != nil {
		http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
		return
	}
	var labels map[string]string
	if err := json.Unmarshal([]byte(r.FormValue("labels_json")), &labels); err != nil {
		http.Error(w, "invalid taxonomy labels", http.StatusBadRequest)
		return
	}
	term := v1.TaxonomyTerm{ID: r.FormValue("id"), Slug: r.FormValue("slug"), Kind: r.FormValue("kind"), Parent: r.FormValue("parent"), Labels: labels}
	if err := upsertTaxonomyTermAtomically(r.Context(), h.DB, repository, term); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	http.Redirect(w, r, "/admin/news/taxonomy", http.StatusSeeOther)
}

func (h Handler) adminTaxonomyTermDelete(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	repository, ok := h.Service.Repository.(taxonomyAdminRepository)
	if !ok || database.RequireExecutor(h.DB) != nil {
		http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := deleteTaxonomyTermAtomically(r.Context(), h.DB, repository, r.PathValue("id")); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	http.Redirect(w, r, "/admin/news/taxonomy", http.StatusSeeOther)
}

func (h Handler) adminCreate(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxNewsItemFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	id, err := newID()
	if err != nil {
		http.Error(w, "news service unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := r.FormValue("locale")
	if locale == "" {
		locale = h.defaultLocale()
	}
	expiresAt, err := parseExpiresAt(r.FormValue("expires_at"))
	if err != nil {
		http.Error(w, "invalid expiry date", http.StatusBadRequest)
		return
	}
	item := news.Item{ID: id, Locale: locale, Slug: r.FormValue("slug"), Title: r.FormValue("title"), Body: r.FormValue("body"), CoverMediaID: r.FormValue("cover_media_id"), Excerpt: r.FormValue("excerpt"), KeyTakeaways: r.FormValue("key_takeaways"), Category: r.FormValue("category"), AuthorName: r.FormValue("author_name"), Featured: r.FormValue("featured") == "on", Tags: r.FormValue("tags"), SEOTitle: r.FormValue("seo_title"), SEODescription: r.FormValue("seo_description"), ExpiresAt: expiresAt, Status: news.StatusDraft}
	if err := h.validateNewsMedia(r.Context(), item); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Service.CreateDraft(r.Context(), item); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	http.Redirect(w, r, "/admin/news/"+url.PathEscape(id), http.StatusSeeOther)
}

func (h Handler) publicList(w http.ResponseWriter, r *http.Request) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	repository, ok := h.Service.Repository.(publishedListRepository)
	if !ok {
		http.Error(w, "news list unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := canonical.Locale(r.URL.Query().Get("locale"))
	if locale == "" {
		http.Error(w, "locale is required", http.StatusBadRequest)
		return
	}
	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			http.Error(w, "limit must be between 1 and 100", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	items, err := repository.ListPublished(r.Context(), h.DB, locale, limit)
	if err != nil {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	api.WriteJSON(w, http.StatusOK, items)
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
	item := news.Item{ID: id, Locale: value.Locale, Slug: value.Slug, Title: value.Title, Body: value.Body, CoverMediaID: value.CoverMediaID, Excerpt: value.Excerpt, KeyTakeaways: value.KeyTakeaways, Category: value.Category, AuthorName: value.AuthorName, Featured: value.Featured, Tags: value.Tags, SEOTitle: value.SEOTitle, SEODescription: value.SEODescription, ExpiresAt: value.ExpiresAt, Status: news.StatusDraft}
	if err := h.validateNewsMedia(r.Context(), item); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Service.CreateDraft(r.Context(), item); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	h.write(w, r, item.ID)
}

func (h Handler) get(w http.ResponseWriter, r *http.Request) {
	if !auth.Can(auth.UserFromContext(r.Context()), auth.PermissionContentRead) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	h.write(w, r, r.PathValue("id"))
}

func (h Handler) update(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value updateInput
	if !decodeJSON(w, r, &value) {
		return
	}
	item := news.Item{ID: r.PathValue("id"), Locale: value.Locale, Slug: value.Slug, Title: value.Title, Body: value.Body, CoverMediaID: value.CoverMediaID, Excerpt: value.Excerpt, KeyTakeaways: value.KeyTakeaways, Category: value.Category, AuthorName: value.AuthorName, Featured: value.Featured, Tags: value.Tags, SEOTitle: value.SEOTitle, SEODescription: value.SEODescription, ExpiresAt: value.ExpiresAt, Status: news.StatusDraft}
	if err := h.validateNewsMedia(r.Context(), item); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Service.UpdateDraft(r.Context(), item, value.Version); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
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
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
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
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
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
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	h.write(w, r, r.PathValue("id"))
}

func (h Handler) publicGet(w http.ResponseWriter, r *http.Request) {
	// Browser links may use the short public URL. Keep the JSON contract for
	// API clients while serving the editorial page when HTML is requested.
	if strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/html") {
		h.publicGetHTML(w, r)
		return
	}
	h.publicGetWithRoute(w, r, r.PathValue("locale"), r.PathValue("slug"))
}

func (h Handler) publicGetByQuery(w http.ResponseWriter, r *http.Request) {
	h.publicGetWithRoute(w, r, r.URL.Query().Get("locale"), r.URL.Query().Get("slug"))
}

func (h Handler) publicGetWithRoute(w http.ResponseWriter, r *http.Request, rawLocale, rawSlug string) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := canonical.Locale(rawLocale)
	slug, err := canonical.Path(rawSlug)
	if locale == "" || err != nil || slug == "" {
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	item, err := h.Service.GetPublishedByRoute(r.Context(), h.DB, locale, slug)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseRequired) {
			http.Error(w, "news unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	api.WriteJSON(w, http.StatusOK, item)
}

func (h Handler) adminPage(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
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
	if !editorAllowed(r) {
		h.renderReadOnlyNews(w, r, item)
		return
	}
	fieldMap := make(map[string]adminstudio.Field)
	for _, f := range h.AdminCollection.Fields {
		fieldMap[f.ID] = f
	}
	var leftFields, rightMetaFields, rightSEOFields []adminstudio.Field
	for _, id := range []string{"title", "slug", "cover_media_id", "excerpt", "key_takeaways", "body"} {
		if f, ok := fieldMap[id]; ok {
			leftFields = append(leftFields, f)
		}
	}
	for _, id := range []string{"featured", "author_name", "category", "tags", "expires_at"} {
		if f, ok := fieldMap[id]; ok {
			rightMetaFields = append(rightMetaFields, f)
		}
	}
	for _, id := range []string{"seo_title", "seo_description"} {
		if f, ok := fieldMap[id]; ok {
			rightSEOFields = append(rightSEOFields, f)
		}
	}
	if !hasAdminField(rightMetaFields, "expires_at") {
		rightMetaFields = append(rightMetaFields, adminstudio.Field{ID: "expires_at", Label: "Otomatik yayından kalkma", Kind: adminstudio.FieldDate, Widget: "datetime-local"})
	}
	expiresValue := ""
	if item.ExpiresAt != nil {
		expiresValue = item.ExpiresAt.UTC().Format("2006-01-02T15:04")
	}
	values := map[string]string{
		"locale":          item.Locale,
		"slug":            item.Slug,
		"title":           item.Title,
		"body":            item.Body,
		"cover_media_id":  item.CoverMediaID,
		"excerpt":         item.Excerpt,
		"key_takeaways":   item.KeyTakeaways,
		"category":        item.Category,
		"author_name":     item.AuthorName,
		"featured":        boolText(item.Featured),
		"tags":            item.Tags,
		"seo_title":       item.SEOTitle,
		"seo_description": item.SEODescription,
		"expires_at":      expiresValue,
	}
	csrf := security.CSRFTokenFromContext(r.Context())
	uiLocale := ui.AdminLocaleFromRequest(r)

	var leftBuf bytes.Buffer
	leftCol := adminstudio.Collection{ID: "news_left", Label: "Haber İçeriği", Fields: leftFields, Actions: []adminstudio.Action{{Label: "Kaydet", Href: "/admin/news/" + item.ID}}}
	if err := adminstudio.RenderCreateForm(&leftBuf, adminstudio.FormData{Collection: leftCol, Action: "/admin/news/" + item.ID, CSRFToken: csrf, Values: values, HideSubmit: true, UILocale: uiLocale}); err != nil {
		http.Error(w, "news admin unavailable", http.StatusServiceUnavailable)
		return
	}

	var rightMetaBuf bytes.Buffer
	rightMetaCol := adminstudio.Collection{ID: "news_meta", Label: "Genel Bilgiler", Fields: rightMetaFields, Actions: []adminstudio.Action{{Label: "Kaydet", Href: "/admin/news/" + item.ID}}}
	if err := adminstudio.RenderCreateForm(&rightMetaBuf, adminstudio.FormData{Collection: rightMetaCol, Action: "/admin/news/" + item.ID, CSRFToken: csrf, Values: values, HideSubmit: true, UILocale: uiLocale}); err != nil {
		http.Error(w, "news admin unavailable", http.StatusServiceUnavailable)
		return
	}

	var rightSEOBuf bytes.Buffer
	rightSEOCol := adminstudio.Collection{ID: "news_seo", Label: "SEO & Arama Motoru", Fields: rightSEOFields, Actions: []adminstudio.Action{{Label: "Kaydet", Href: "/admin/news/" + item.ID}}}
	if err := adminstudio.RenderCreateForm(&rightSEOBuf, adminstudio.FormData{Collection: rightSEOCol, Action: "/admin/news/" + item.ID, CSRFToken: csrf, Values: values, HideSubmit: true, UILocale: uiLocale}); err != nil {
		http.Error(w, "news admin unavailable", http.StatusServiceUnavailable)
		return
	}

	var taxonomyBuf bytes.Buffer
	if taxonomy, ok := h.Service.Repository.(taxonomyAdminRepository); ok {
		terms, termsErr := taxonomy.ListTaxonomyTerms(r.Context(), h.DB)
		selected, selectedErr := taxonomy.ListTaxonomy(r.Context(), h.DB, item.ID)
		if termsErr != nil || selectedErr != nil {
			http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
			return
		}
		selectedIDs := make([]string, 0, len(selected))
		for _, term := range selected {
			selectedIDs = append(selectedIDs, term.ID)
		}
		if err := adminstudio.RenderTaxonomyEditor(&taxonomyBuf, adminstudio.TaxonomyEditorOptions{Terms: terms, Selected: selectedIDs}); err != nil {
			http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Haber düzenle", "Yayın merkezi", "news"); err != nil {
		return
	}
	if h.Presence != nil {
		endpoint := "/admin/news/" + url.PathEscape(item.ID) + "/presence"
		_ = designsystem.EditPresence(item.ID, endpoint, security.CSRFTokenFromContext(r.Context())).Render(r.Context(), w)
	}
	statusBadge := fmt.Sprintf(`<span class="status">%s</span>`, template.HTMLEscapeString(string(item.Status)))
	titleDisplay := item.Title
	if strings.TrimSpace(titleDisplay) == "" {
		titleDisplay = "[Başlıksız]"
	}
	previewEndpoint := "/admin/news/" + url.PathEscape(item.ID) + "/preview"
	copyLiveURL := ""
	if item.Status == news.StatusPublished && item.Slug != "" {
		publicPath := "/public/news/" + url.PathEscape(item.Locale) + "/" + url.PathEscape(item.Slug) + "/html"
		copyLiveURL = strings.TrimRight(strings.TrimSpace(h.PublicOrigin), "/") + publicPath
	}
	extraActions := new(strings.Builder)
	if copyLiveURL != "" {
		extraActions.WriteString(`<button type="button" class="button button-quiet" data-copy-text="` + template.HTMLEscapeString(copyLiveURL) + `" aria-label="Canlı URL&#39;yi kopyala" title="Canlı URL&#39;yi kopyala">🔗 URL kopyala</button><span data-copy-status style="font-size:.75rem;color:var(--muted)"></span>`)
	}
	extraActions.WriteString(`<button type="button" class="button button-quiet" data-device-preview-open data-preview-endpoint="` + template.HTMLEscapeString(previewEndpoint) + `" data-preview-csrf="` + template.HTMLEscapeString(csrf) + `" title="Önizleme">👁 Önizle</button>`)
	_, _ = fmt.Fprintf(w, `<div class="editor-top-nav"><div class="editor-top-info"><strong class="editor-live-title">%s</strong><span class="editor-live-sub">Haber düzenleniyor · %s</span></div><div class="editor-top-actions">%s<button type="submit" form="news-edit-form" class="button">Taslağı kaydet</button></div></div><form id="news-edit-form" class="studio-form" method="post" action="/admin/news/%s" data-local-autosave data-autosave-resource="news:%s"><div class="editor-split-container"><div class="editor-main-col"><section class="editor-card"><div class="editor-card-title"><span>Haber İçeriği</span></div>%s</section></div><div class="editor-side-col"><section class="editor-card"><div class="editor-card-title"><span>Yayın &amp; İşlem</span>%s</div><div class="editor-side-fields"><div class="editor-side-status"><span>Durum</span><strong>%s</strong></div><div class="editor-side-status"><span>Dil</span><strong>%s</strong></div><div class="editor-workflow-actions">%s</div></div></section><section class="editor-card"><div class="editor-card-title"><span>Genel Bilgiler</span></div><div class="editor-side-fields">%s</div></section><section class="editor-card"><div class="editor-card-title"><span>İçerik Sınıflandırması</span></div><div class="editor-side-fields">%s</div></section><section class="editor-card"><div class="editor-card-title"><span>SEO &amp; Arama Motoru</span></div><div class="editor-side-fields">%s</div></section></div></div><input type="hidden" name="locale" value="%s"><input type="hidden" name="version" value="%d"><input type="hidden" name="_csrf" value="%s"></form>%s<dialog class="device-preview-dialog" data-device-preview-dialog aria-labelledby="device-preview-title"><div class="device-preview-header"><div><h2 id="device-preview-title">Haber Önizlemesi</h2><p data-device-preview-status>Önizle butonuna basarak güncel taslağı yükleyin.</p></div><div class="studio-actions"><button type="button" class="button button-quiet" data-device-preview-size="desktop" aria-pressed="true" title="Masaüstü">🖥 Masaüstü</button><button type="button" class="button button-quiet" data-device-preview-size="tablet" aria-pressed="false" title="Tablet">📟 Tablet</button><button type="button" class="button button-quiet" data-device-preview-size="mobile" aria-pressed="false" title="Mobil">📱 Mobil</button><button type="button" class="button button-quiet" data-device-preview-close aria-label="Önizlemeyi kapat">✕ Kapat</button></div></div><div class="device-preview-stage"><iframe data-device-preview-frame src="about:blank" title="Haber önizlemesi" sandbox="allow-same-origin allow-scripts allow-forms"></iframe></div></dialog><script>(()=>{const titleInp=document.querySelector('#news-edit-form input[name="title"]'),liveTitle=document.querySelector('.editor-live-title');if(titleInp&&liveTitle){titleInp.addEventListener('input',()=>{liveTitle.textContent=titleInp.value.trim()||'[Başlıksız]';});}})();</script>`,
		template.HTMLEscapeString(titleDisplay), statusBadge,
		template.HTML(extraActions.String()),
		template.HTMLEscapeString(item.ID), template.HTMLEscapeString(item.ID),
		template.HTML(leftBuf.String()), statusBadge,
		template.HTMLEscapeString(string(item.Status)), template.HTMLEscapeString(item.Locale),
		workflowActionMarkup(h.Service, item),
		template.HTML(rightMetaBuf.String()), template.HTML(taxonomyBuf.String()), template.HTML(rightSEOBuf.String()),
		template.HTMLEscapeString(item.Locale), item.Version, template.HTMLEscapeString(csrf),
		template.HTML(workflowFormsMarkup(h.Service, item, csrf)))

	_ = ui.AdminEnd(w)
}

func (h Handler) renderReadOnlyNews(w http.ResponseWriter, r *http.Request, item news.Item) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Haberi görüntüle", "Salt okunur içerik", "news"); err != nil {
		return
	}
	body := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		for _, field := range []uicomponents.InputProps{
			{ID: "news-title-readonly", Name: "title", Label: "Başlık", Value: item.Title, ReadOnly: true},
			{ID: "news-slug-readonly", Name: "slug", Label: "Adres", Value: item.Slug, ReadOnly: true},
			{ID: "news-locale-readonly", Name: "locale", Label: "Dil", Value: item.Locale, ReadOnly: true},
			{ID: "news-status-readonly", Name: "status", Label: "Durum", Value: string(item.Status), ReadOnly: true},
			{ID: "news-category-readonly", Name: "category", Label: "Kategori", Value: item.Category, ReadOnly: true},
			{ID: "news-author-readonly", Name: "author", Label: "Yazar", Value: item.AuthorName, ReadOnly: true},
			{ID: "news-tags-readonly", Name: "tags", Label: "Etiketler", Value: item.Tags, ReadOnly: true},
		} {
			if err := uicomponents.Input(field).Render(ctx, out); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintf(out, `<article class="readonly-content-preview rich-article-content">%s</article>`, template.HTML(ui.RichTextHTML(item.Body)))
		return err
	})
	_ = uicomponents.Card(uicomponents.CardProps{Title: "Haber içeriği", Class: "studio-card", Body: body}).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func boolText(value bool) string {
	if value {
		return "on"
	}
	return ""
}

func hasAdminField(fields []adminstudio.Field, id string) bool {
	for _, field := range fields {
		if field.ID == id {
			return true
		}
	}
	return false
}
func parseExpiresAt(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02T15:04", value, time.UTC)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func workflowActionMarkup(service news.Service, item news.Item) template.HTML {
	var out strings.Builder
	if service.CanTransition(item.Status, v1.EditorialScheduled) {
		out.WriteString(`<label for="news-scheduled-at">Yayın tarihi ve saati</label><input id="news-scheduled-at" name="scheduled_at" type="datetime-local" form="news-schedule-form" required><button type="submit" form="news-schedule-form" class="button">Zamanla</button>`)
	}
	if service.CanTransition(item.Status, v1.EditorialPublished) {
		out.WriteString(`<button type="submit" form="news-publish-form" class="button button-success">Yayına Al</button>`)
	}
	if service.CanTransition(item.Status, v1.EditorialRetracted) {
		out.WriteString(`<button type="submit" form="news-retract-form" class="button button-quiet button-danger">Yayından Çek</button>`)
	}
	if out.Len() == 0 {
		out.WriteString(`<p class="muted">Bu durum için kullanılabilir yayın aksiyonu yok.</p>`)
	}
	return template.HTML(out.String())
}

func workflowFormsMarkup(service news.Service, item news.Item, csrf string) template.HTML {
	var out strings.Builder
	id := template.HTMLEscapeString(item.ID)
	token := template.HTMLEscapeString(csrf)
	for _, action := range []struct {
		id    string
		state v1.EditorialState
		path  string
	}{{"news-schedule-form", v1.EditorialScheduled, "schedule"}, {"news-publish-form", v1.EditorialPublished, "publish"}, {"news-retract-form", v1.EditorialRetracted, "retract"}} {
		if !service.CanTransition(item.Status, action.state) {
			continue
		}
		fmt.Fprintf(&out, `<form id="%s" method="post" action="/admin/news/%s/%s"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="version" value="%d"></form>`, action.id, id, action.path, token, item.Version)
	}
	return template.HTML(out.String())
}

func (h Handler) adminUpdate(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxNewsItemFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	version, err := strconv.Atoi(r.FormValue("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	locale := r.FormValue("locale")
	if locale == "" {
		locale = h.defaultLocale()
	}
	expiresAt, err := parseExpiresAt(r.FormValue("expires_at"))
	if err != nil {
		http.Error(w, "invalid expiry date", http.StatusBadRequest)
		return
	}
	item := news.Item{ID: r.PathValue("id"), Locale: locale, Slug: r.FormValue("slug"), Title: r.FormValue("title"), Body: r.FormValue("body"), CoverMediaID: r.FormValue("cover_media_id"), Excerpt: r.FormValue("excerpt"), KeyTakeaways: r.FormValue("key_takeaways"), Category: r.FormValue("category"), AuthorName: r.FormValue("author_name"), Featured: r.FormValue("featured") == "on", Tags: r.FormValue("tags"), SEOTitle: r.FormValue("seo_title"), SEODescription: r.FormValue("seo_description"), ExpiresAt: expiresAt, Status: news.StatusDraft}
	if err := h.validateNewsMedia(r.Context(), item); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Service.UpdateDraft(r.Context(), item, version); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	if repository, ok := h.Service.Repository.(taxonomyAdminRepository); ok {
		terms, err := repository.ListTaxonomyTerms(r.Context(), h.DB)
		if err == nil {
			allowed := make(map[string]v1.TaxonomyTerm, len(terms))
			for _, term := range terms {
				allowed[term.ID] = term
			}
			selected := make([]v1.TaxonomyTerm, 0, len(r.Form["taxonomy_term"]))
			for _, id := range r.Form["taxonomy_term"] {
				if term, exists := allowed[id]; exists {
					selected = append(selected, term)
				}
			}
			if err := replaceTaxonomyAtomically(r.Context(), h.DB, repository, item.ID, selected); err != nil {
				api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
				return
			}
		}
	}
	http.Redirect(w, r, "/admin/news/"+url.PathEscape(item.ID)+"?saved=1", http.StatusSeeOther)
}

func (h Handler) adminTaxonomy(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxNewsTaxonomyFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	repository, ok := h.Service.Repository.(taxonomyAdminRepository)
	if !ok {
		http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
		return
	}
	terms, err := repository.ListTaxonomyTerms(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
		return
	}
	allowed := make(map[string]v1.TaxonomyTerm, len(terms))
	for _, term := range terms {
		allowed[term.ID] = term
	}
	selected := make([]v1.TaxonomyTerm, 0, len(r.Form["taxonomy_term"]))
	for _, id := range r.Form["taxonomy_term"] {
		if term, exists := allowed[id]; exists {
			selected = append(selected, term)
		}
	}
	if err := replaceTaxonomyAtomically(r.Context(), h.DB, repository, r.PathValue("id"), selected); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	http.Redirect(w, r, "/admin/news/"+url.PathEscape(r.PathValue("id")), http.StatusSeeOther)
}

func replaceTaxonomyAtomically(ctx context.Context, executor database.Executor, repository taxonomyAdminRepository, contentID string, terms []v1.TaxonomyTerm) error {
	starter, ok := executor.(transactionStarter)
	if !ok {
		return repository.ReplaceTaxonomy(ctx, executor, contentID, terms)
	}
	tx, err := starter.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := repository.ReplaceTaxonomy(ctx, tx, contentID, terms); err != nil {
		return err
	}
	return tx.Commit()
}

func upsertTaxonomyTermAtomically(ctx context.Context, executor database.Executor, repository taxonomyAdminRepository, term v1.TaxonomyTerm) error {
	starter, ok := executor.(transactionStarter)
	if !ok {
		return repository.UpsertTaxonomyTerm(ctx, executor, term)
	}
	tx, err := starter.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := repository.UpsertTaxonomyTerm(ctx, tx, term); err != nil {
		return err
	}
	return tx.Commit()
}

func deleteTaxonomyTermAtomically(ctx context.Context, executor database.Executor, repository taxonomyAdminRepository, id string) error {
	starter, ok := executor.(transactionStarter)
	if !ok {
		return repository.DeleteTaxonomyTerm(ctx, executor, id)
	}
	tx, err := starter.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := repository.DeleteTaxonomyTerm(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (h Handler) adminSchedule(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxNewsWorkflowFormBytes)
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
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	http.Redirect(w, r, "/admin/news/"+url.PathEscape(r.PathValue("id")), http.StatusSeeOther)
}

func (h Handler) adminPublish(w http.ResponseWriter, r *http.Request) {
	h.adminTransition(w, r, "publish")
}

func (h Handler) adminRetract(w http.ResponseWriter, r *http.Request) {
	h.adminTransition(w, r, "retract")
}

func (h Handler) adminPreview(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	id := r.PathValue("id")
	item, err := h.Service.Repository.Get(r.Context(), h.DB, id)
	if err != nil {
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	publicPath := "/public/news/" + url.PathEscape(item.Locale) + "/" + url.PathEscape(item.Slug) + "/html"
	if item.Status != news.StatusPublished {
		if h.PreviewSigner == nil || (item.Status != news.StatusDraft && item.Status != news.StatusScheduled && item.Status != news.StatusRetracted) {
			http.Error(w, "news preview unavailable", http.StatusServiceUnavailable)
			return
		}
		claims := preview.Claims{ResourceType: "news", ResourceID: item.ID, Revision: item.Revision}
		if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
			claims.TenantID = scope.TenantID
		} else if h.RequireTenantScope {
			http.Error(w, "news preview unavailable", http.StatusForbidden)
			return
		}
		token, err := h.PreviewSigner.Issue(claims, time.Now().UTC(), 15*time.Minute)
		if err != nil {
			http.Error(w, "news preview unavailable", http.StatusServiceUnavailable)
			return
		}
		publicPath += "?preview_token=" + url.QueryEscape(token)
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"url":%q}`, publicPath)
}

func (h Handler) adminTransition(w http.ResponseWriter, r *http.Request, action string) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxNewsWorkflowFormBytes)
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
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	http.Redirect(w, r, "/admin/news/"+url.PathEscape(r.PathValue("id")), http.StatusSeeOther)
}

func (h Handler) write(w http.ResponseWriter, r *http.Request, id string) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
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
	api.WriteJSON(w, http.StatusOK, item)
}

func editorAllowed(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return auth.Can(user, auth.PermissionContentWrite)
}

func contentReadAllowed(r *http.Request) bool {
	return auth.Can(auth.UserFromContext(r.Context()), auth.PermissionContentRead)
}

func (h Handler) editPresenceHeartbeat(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	resource := r.PathValue("id")
	if h.Presence == nil || !editorAllowed(r) || user == nil || !user.Active || len(resource) > 128 || strings.ContainsAny(resource, "\x00\r\n") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	} else if h.RequireTenantScope {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	name := strings.TrimSpace(strings.SplitN(user.Email, "@", 2)[0])
	if name == "" {
		name = "Editor"
	}
	if err := h.Presence.Heartbeat(realtime.PresenceSession{TenantID: tenantID, Resource: "news:" + resource, UserID: user.ID, Name: name}); err != nil {
		http.Error(w, "presence unavailable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) editPresenceList(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if h.Presence == nil || !editorAllowed(r) || user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	} else if h.RequireTenantScope {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteJSON(w, http.StatusOK, h.Presence.Active(tenantID, "news:"+r.PathValue("id"), user.ID))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := security.DecodeJSONStrict(w, r, target, 1<<20); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return false
	}
	return true
}

func newID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func newsErrorStatus(err error) int {
	if errors.Is(err, news.ErrRepositoryRequired) || errors.Is(err, database.ErrDatabaseRequired) {
		return http.StatusServiceUnavailable
	}
	if errors.Is(err, news.ErrInvalidItem) {
		return http.StatusBadRequest
	}
	if errors.Is(err, news.ErrConflict) || errors.Is(err, news.ErrScheduleStale) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}
