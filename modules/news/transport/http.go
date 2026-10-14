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
	"regexp"
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

// publicGetHTML exposes a human-facing counterpart to the JSON public route.
// The JSON endpoint remains stable for headless clients.
func (h Handler) publicGetHTML(w http.ResponseWriter, r *http.Request) {
	previewed := r.URL.Query().Get("preview_token") != ""
	if previewed {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		w.Header().Set("Referrer-Policy", "no-referrer")
	}
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := canonical.Locale(r.PathValue("locale"))
	slug, err := canonical.Path(r.PathValue("slug"))
	if locale == "" || err != nil {
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	item, previewed, err := h.publicHTMLItem(r, locale, slug)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseRequired) {
			http.Error(w, "news unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	if previewed {
		security.AllowSameOriginFrame(w.Header())
	}

	// Fetch popular/recent news items for the sticky sidebar
	var sidebarItems []news.Item
	if repository, ok := h.Service.Repository.(relatedListRepository); ok {
		if list, err := repository.ListRelatedPublished(r.Context(), h.DB, item.ID, locale, 4); err == nil {
			sidebarItems = list
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	consentNotice := ""
	if !previewed {
		consentNotice = h.analyticsConsentNotice(r, locale)
	}
	if h.AnalyticsEnabled && !previewed {
		w.Header().Set("Vary", "Cookie")
	}
	documentStart := newsDocumentStart(locale, item.Title, h.themeNavigation(r.Context(), locale))
	if consentNotice != "" || h.PublicSupport.Enabled || h.PublicContactSettings != nil || h.Feedback != nil {
		documentStart = strings.Replace(documentStart, `</head>`, `<link rel="stylesheet" href="/designsystem/static/designsystem.css"></head>`, 1)
	}
	_, _ = fmt.Fprint(w, localizePublicLinks(documentStart, locale))

	var trendingHTML strings.Builder
	itemIndex := 1
	for _, sb := range sidebarItems {
		if sb.Slug == item.Slug {
			continue
		}
		sbKicker := "HABER"
		if strings.TrimSpace(sb.Category) != "" {
			sbKicker = strings.ToUpper(sb.Category)
		}
		trendingHTML.WriteString(fmt.Sprintf(
			`<a href="/public/news/%s/%s/html" class="flex items-center gap-3 p-2 rounded-lg hover:bg-slate-50 transition group border border-slate-100"><span class="w-6 h-6 rounded-full bg-[#ed1c24] text-white font-black text-xs flex items-center justify-center shrink-0">%02d</span><div class="flex-1 min-w-0"><span class="text-[10px] font-bold text-[#ed1c24] uppercase block">%s</span><h4 class="text-xs font-bold text-slate-800 group-hover:text-red-600 line-clamp-2">%s</h4></div></a>`,
			template.HTMLEscapeString(locale),
			template.HTMLEscapeString(sb.Slug),
			itemIndex,
			template.HTMLEscapeString(sbKicker),
			template.HTMLEscapeString(sb.Title),
		))
		itemIndex++
		if itemIndex > 4 {
			break
		}
	}
	if trendingHTML.Len() == 0 {
		trendingHTML.WriteString(`<p style="color:var(--theme-muted);font-size:.88rem;margin:0;">Henüz benzer içerik bulunmuyor.</p>`)
	}
	var relatedCards strings.Builder
	if len(sidebarItems) > 0 {
		relatedCards.WriteString(`<section class="mt-8 border-t border-slate-200 pt-6"><div class="flex items-center gap-2 border-b-2 border-[#ed1c24] pb-2 mb-4"><span class="w-2.5 h-6 bg-[#ed1c24] rounded-sm"></span><h2 class="text-lg font-black text-slate-900 uppercase">Bunlar da İlginizi Çekebilir</h2></div><div class="grid grid-cols-1 md:grid-cols-3 gap-4">`)
		for _, related := range sidebarItems {
			if related.Slug == item.Slug {
				continue
			}
			relCat := "GÜNDEM"
			if strings.TrimSpace(related.Category) != "" {
				relCat = strings.ToUpper(related.Category)
			}
			fmt.Fprintf(&relatedCards, `<article class="bg-white rounded-xl border border-slate-200 overflow-hidden shadow-xs hover:shadow-md transition flex flex-col group"><div class="aspect-[16/9] relative overflow-hidden bg-slate-100">%s<span class="absolute top-2 left-2 bg-red-600 text-white text-[10px] font-black px-2 py-0.5 rounded shadow-xs uppercase">%s</span></div><div class="p-3 flex flex-col justify-between flex-1"><h3 class="text-xs font-extrabold text-slate-900 group-hover:text-red-600 line-clamp-2 mb-2 leading-snug"><a href="/public/news/%s/%s/html">%s</a></h3><div class="flex items-center gap-1.5 text-[10px] text-slate-400 mt-auto"><svg class="w-3 h-3 text-slate-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg><span>%d dk okuma</span></div></div></article>`, h.newsCoverMarkup(related), template.HTMLEscapeString(relCat), template.HTMLEscapeString(locale), template.HTMLEscapeString(related.Slug), template.HTMLEscapeString(related.Title), readingMinutes(related.Body))
		}
		relatedCards.WriteString(`</div></section>`)
	}
	articleHTML, tocHTML, err := ui.PrepareArticleHTML(renderParagraphs(item.Body))
	if err != nil {
		http.Error(w, "news unavailable", http.StatusInternalServerError)
		return
	}
	var readingTools bytes.Buffer
	if err := ui.PublicArticleTools().Render(r.Context(), &readingTools); err != nil {
		http.Error(w, "news unavailable", http.StatusInternalServerError)
		return
	}
	readingEnhancement := `<link rel="stylesheet" href="/designsystem/static/article-reading.css"><script src="/designsystem/static/article-reading.js" defer></script>`
	var helpfulFeedback bytes.Buffer
	if h.Feedback != nil && !previewed {
		scope, scopeErr := tenant.FromContext(r.Context())
		if scopeErr == nil || !h.RequireTenantScope {
			tenantID := "default"
			if scopeErr == nil {
				tenantID = scope.TenantID
			}
			totals, _ := h.Feedback.FeedbackTotals(r.Context(), tenantID, "news", item.ID)
			if err := designsystem.HelpfulFeedback(designsystem.HelpfulFeedbackProps{ContentType: "news", ContentID: item.ID, ElementID: "article-feedback", CSRF: security.CSRFTokenFromContext(r.Context()), Up: totals.Up, Down: totals.Down, LoadHTMX: true}).Render(r.Context(), &helpfulFeedback); err != nil {
				helpfulFeedback.Reset()
			}
		}
	}
	footer := localizePublicLinks(enterpriseFooterMarkup(), locale)
	if consentNotice != "" || h.PublicSupport.Enabled || h.PublicContactSettings != nil {
		footer = strings.Replace(footer, `</body>`, consentNotice+h.socialLinksMarkup(r.Context())+h.floatingContact(r.Context(), locale)+`</body>`, 1)
	}

	_, _ = fmt.Fprintf(w, `<main class="max-w-7xl mx-auto px-4 py-6 w-full"><nav class="flex items-center gap-2 text-xs font-semibold text-slate-500 mb-4" aria-label="Breadcrumb"><a href="/" class="hover:text-red-600 transition">Ana Sayfa</a><svg class="w-3 h-3 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg><a href="/?category=%s#news-grid" class="text-red-600 hover:underline uppercase">%s</a><svg class="w-3 h-3 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg><span class="text-slate-800 truncate max-w-md">%s</span></nav><div class="grid grid-cols-1 lg:grid-cols-12 gap-8"><article class="lg:col-span-8 flex flex-col"><div class="mb-3"><span class="bg-red-600 text-white text-xs font-black px-3 py-1 rounded shadow-xs uppercase tracking-wider">%s</span></div><h1 class="text-2xl md:text-4xl font-black text-slate-900 leading-tight tracking-tight mb-4">%s</h1><div class="bg-slate-50 border border-slate-200/80 p-5 rounded-xl text-base md:text-lg font-medium text-slate-700 leading-relaxed mb-6">%s</div><div class="flex flex-wrap items-center justify-between gap-3 border-y border-slate-200 py-3 mb-6"><div class="flex items-center gap-2.5 text-xs text-slate-600"><div class="w-8 h-8 rounded-full bg-slate-200 flex items-center justify-center font-bold text-slate-700"><svg class="w-4 h-4 text-slate-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg></div><div><span class="font-bold text-slate-800 block text-sm">%s</span><span class="text-slate-400 text-xs">%s · %d dk okuma</span></div></div><div class="flex items-center gap-2"><span class="text-xs font-bold text-slate-400 mr-1 hidden sm:inline">Paylaş:</span><a href="https://wa.me/?text=%s" target="_blank" rel="noopener" class="bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold px-3 py-1.5 rounded-lg flex items-center gap-1.5 transition shadow-xs"><svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="currentColor"><path d="M12.04 2C6.58 2 2.13 6.45 2.13 11.91C2.13 13.66 2.59 15.36 3.45 16.86L2.05 22L7.3 20.62C8.75 21.41 10.38 21.83 12.04 21.83C17.5 21.83 21.95 17.38 21.95 11.92C21.95 9.27 20.92 6.78 19.05 4.91C17.18 3.03 14.69 2 12.04 2M12.05 3.67C14.25 3.67 16.31 4.53 17.87 6.09C19.42 7.65 20.28 9.72 20.28 11.92C20.28 16.46 16.58 20.15 12.04 20.15C10.56 20.15 9.11 19.76 7.85 19L7.55 18.83L4.43 19.65L5.26 16.61L5.06 16.29C4.24 14.99 3.81 13.47 3.81 11.91C3.81 7.37 7.5 3.67 12.05 3.67Z"/></svg>WhatsApp</a><a href="https://twitter.com/intent/tweet?text=%s" target="_blank" rel="noopener" class="bg-black hover:bg-slate-800 text-white text-xs font-bold px-3 py-1.5 rounded-lg flex items-center gap-1.5 transition shadow-xs"><svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="currentColor"><path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/></svg>X</a><button type="button" onclick="navigator.clipboard.writeText(window.location.href);alert('Haber bağlantısı kopyalandı!')" class="bg-slate-100 hover:bg-slate-200 text-slate-700 text-xs font-bold px-3 py-1.5 rounded-lg flex items-center gap-1.5 transition cursor-pointer border border-slate-200 shadow-xs"><svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path></svg>Kopyala</button></div></div><div class="w-full aspect-[16/9] rounded-xl overflow-hidden mb-6 bg-slate-100 border border-slate-200 shadow-xs">%s</div><div class="bg-white p-6 md:p-8 rounded-xl border border-slate-200 shadow-xs text-slate-800 editorial-prose mb-6">%s%s%s%s%s</div>%s%s</article><aside class="lg:col-span-4 flex flex-col gap-6"><div class="bg-white rounded-xl border border-slate-200 p-5 shadow-xs"><div class="flex items-center gap-2 border-b-2 border-red-600 pb-2 mb-4"><span class="w-2 h-5 bg-red-600 rounded-sm"></span><h3 class="text-sm font-black text-slate-900 uppercase tracking-wider flex items-center gap-1.5"><svg class="w-4 h-4 text-red-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg>Günün Öne Çıkanları</h3></div><div class="space-y-3">%s</div></div><div class="bg-gradient-to-br from-slate-900 to-slate-800 text-white p-6 rounded-xl shadow-md"><span class="bg-red-600 text-white text-[10px] font-black px-2.5 py-1 rounded uppercase tracking-wider inline-flex items-center gap-1"><svg class="w-3 h-3 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"></path><polyline points="22,6 12,13 2,6"></polyline></svg>OKUR BÜLTENİ</span><h4 class="text-base font-black mt-3 mb-1">Önemli Haberleri Kaçırmayın</h4><p class="text-xs text-slate-300 mb-4 leading-relaxed">Günün en önemli gelişmelerini her sabah e-postanıza gönderiyoruz.</p><a href="/member/register" class="bg-red-600 hover:bg-red-700 text-white font-bold text-xs py-2 px-4 rounded-lg block text-center transition shadow-xs">Ücretsiz Abone Ol →</a></div><div class="bg-white rounded-xl border border-slate-200 p-4 shadow-xs text-center"><span class="text-xs text-slate-500 block mb-2 font-medium">Haber Merkezi Yönetimi</span><a href="/admin/news" class="bg-slate-900 hover:bg-slate-800 text-white font-bold text-xs py-2 px-4 rounded-lg inline-flex items-center gap-1.5 transition shadow-xs"><svg class="w-3.5 h-3.5 text-amber-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>Studio Editör Masası ↗</a></div></aside></div></main>%s%s`,
		template.HTMLEscapeString(strings.ToLower(item.Category)),
		template.HTMLEscapeString(item.Category),
		template.HTMLEscapeString(item.Title),
		strings.ToUpper(template.HTMLEscapeString(item.Category)),
		template.HTMLEscapeString(item.Title),
		template.HTMLEscapeString(newsLeadItem(item)),
		template.HTMLEscapeString(item.AuthorName),
		template.HTMLEscapeString(turkishDate(item.UpdatedAt, false)),
		readingMinutes(item.Body),
		template.URLQueryEscaper(item.Title),
		template.URLQueryEscaper(item.Title),
		h.newsCoverMarkup(item),
		newsKeyTakeawaysMarkup(item),
		readingTools.String(),
		template.HTML(tocHTML),
		template.HTML(articleHTML),
		newsTagsMarkup(item),
		helpfulFeedback.String(),
		template.HTML(relatedCards.String()),
		trendingHTML.String(),
		readingEnhancement,
		footer,
	)
}

func (h Handler) publicHTMLItem(r *http.Request, locale, slug string) (news.Item, bool, error) {
	token := r.URL.Query().Get("preview_token")
	if token == "" {
		item, err := h.Service.GetPublishedByRoute(r.Context(), h.DB, locale, slug)
		return item, false, err
	}
	if h.PreviewSigner == nil {
		return news.Item{}, false, errors.New("preview unavailable")
	}
	claims, err := h.PreviewSigner.Verify(token, time.Now().UTC())
	if err != nil || claims.ResourceType != "news" {
		return news.Item{}, false, errors.New("preview unavailable")
	}
	if h.RequireTenantScope || claims.TenantID != "" {
		scope, scopeErr := tenant.Require(r.Context())
		if scopeErr != nil || scope.TenantID != claims.TenantID {
			return news.Item{}, false, errors.New("preview unavailable")
		}
	}
	item, err := h.Service.Repository.Get(r.Context(), h.DB, claims.ResourceID)
	if err != nil || item.Status == news.StatusPublished || item.Revision != claims.Revision || item.Locale != locale || item.Slug != slug {
		return news.Item{}, false, errors.New("preview unavailable")
	}
	return item, true, nil
}

func (h Handler) analyticsConsentNotice(r *http.Request, locale string) string {
	if !h.AnalyticsEnabled {
		return ""
	}
	if cookie, err := r.Cookie(analytics.ConsentCookie); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return ""
	}
	props := designsystem.CookieNoticeProps{Action: "/analytics/consent", CSRF: security.CSRFTokenFromContext(r.Context()), ReturnTo: r.URL.Path, Variant: "bar"}
	if strings.HasPrefix(canonical.Locale(locale), "en") {
		props.Heading, props.Message, props.AcceptLabel, props.RejectLabel = "Privacy choices", "Allow anonymous, daily page-view totals? HydroGo does not use advertising cookies.", "Allow analytics", "Decline"
	} else {
		props.Heading, props.Message, props.AcceptLabel, props.RejectLabel = "Gizlilik tercihiniz", "Anonim günlük sayfa görüntülenme toplamlarına izin verilsin mi? HydroGo reklam çerezi kullanmaz.", "Analitiğe izin ver", "Reddet"
	}
	var notice strings.Builder
	if designsystem.CookieNotice(props).Render(r.Context(), &notice) != nil {
		return ""
	}
	return notice.String()
}

func (h Handler) publicListHTML(w http.ResponseWriter, r *http.Request) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	repository, ok := h.Service.Repository.(publishedListRepository)
	if !ok {
		http.Error(w, "news list unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := canonical.Locale(r.PathValue("locale"))
	if locale == "" {
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	items, err := repository.ListPublished(r.Context(), h.DB, locale, 50)
	if err != nil {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	documentStart := newsDocumentStart(locale, "Son Dakika ve Güncel Haberler", h.themeNavigation(r.Context(), locale))
	if h.PublicSupport.Enabled || h.PublicContactSettings != nil {
		documentStart = strings.Replace(documentStart, `</head>`, `<link rel="stylesheet" href="/designsystem/static/designsystem.css"></head>`, 1)
	}
	_, _ = fmt.Fprint(w, localizePublicLinks(documentStart, locale))
	_, _ = fmt.Fprintf(w, `<main data-testid="public-news" class="max-w-7xl mx-auto px-4 py-4 w-full">`)

	if len(items) == 0 {
		_, _ = fmt.Fprint(w, `<div class="bg-white rounded-xl border border-slate-200 p-12 text-center my-6"><div class="w-12 h-12 mx-auto mb-3 text-slate-400 flex items-center justify-center"><svg class="w-10 h-10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><line x1="7" y1="8" x2="17" y2="8"></line><line x1="7" y1="12" x2="17" y2="12"></line><line x1="7" y1="16" x2="13" y2="16"></line></svg></div><h2 class="text-xl font-extrabold text-slate-900 mb-2">Henüz yayınlanmış bir haber bulunmuyor</h2><p class="text-sm text-slate-500 mb-6">Admin panelinden yeni bir haber oluşturup yayınlayarak burada görüntüleyebilirsiniz.</p><a class="inline-flex items-center gap-2 bg-[#ed1c24] hover:bg-red-700 text-white font-bold text-sm px-5 py-2.5 rounded-lg transition" href="/admin/news">Yeni Haber Ekle ↗</a></div>`)
	} else {
		featured := items[0]
		for _, candidate := range items {
			if candidate.Featured {
				featured = candidate
				break
			}
		}

		// Prepare slider items (up to 10 items)
		sliderItems := []news.Item{featured}
		for _, item := range items {
			if item.ID != featured.ID && len(sliderItems) < 10 {
				sliderItems = append(sliderItems, item)
			}
		}

		// Helper to resolve cover URL
		getCoverURL := func(item news.Item) string {
			if strings.TrimSpace(item.CoverMediaID) == "" {
				return ""
			}
			key := template.URLQueryEscaper(item.CoverMediaID)
			src := "/media/" + key + "?variant=card"
			if h.Media != nil {
				if publicURL, err := h.Media.PublicURL(item.CoverMediaID); err == nil && strings.TrimSpace(publicURL) != "" {
					src = publicURL
				}
			}
			return src
		}

		// 1. TGRT HABER HERO: MANŞET SLIDER (8 col) + YAN MANŞETLER (4 col)
		_, _ = fmt.Fprint(w, `<section class="grid grid-cols-1 lg:grid-cols-12 gap-4 mb-8">`)

		// Left: Main Slider (lg:col-span-8)
		featCover := getCoverURL(featured)
		featCat := "GÜNDEM"
		if strings.TrimSpace(featured.Category) != "" {
			featCat = strings.ToUpper(featured.Category)
		}
		featLead := newsLeadItem(featured)

		_, _ = fmt.Fprintf(w, `<div class="lg:col-span-8"><div id="main-slider-box" class="relative bg-slate-900 rounded-xl overflow-hidden aspect-[16/9] shadow-md group"><div class="absolute inset-0">`)
		if featCover != "" {
			_, _ = fmt.Fprintf(w, `<img id="slider-main-img" src="%s" alt="%s" class="w-full h-full object-cover transition duration-300">`, template.HTMLEscapeString(featCover), template.HTMLEscapeString(featured.Title))
		} else {
			_, _ = fmt.Fprintf(w, `<div id="slider-main-img-fallback" class="w-full h-full bg-slate-800 flex items-center justify-center text-slate-500"><svg class="w-16 h-16 text-slate-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg></div>`)
		}
		_, _ = fmt.Fprintf(w, `</div><div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/40 to-transparent flex flex-col justify-end p-5 md:p-8"><div class="mb-2"><span id="slider-main-tag" class="bg-[#ed1c24] text-white text-xs font-black px-3 py-1 rounded shadow-md uppercase tracking-wider">%s</span></div><h2 class="mb-2"><a id="slider-main-title" href="/public/news/%s/%s/html" class="text-white font-black text-xl md:text-3xl leading-snug hover:text-red-400 transition block drop-shadow-md">%s</a></h2><p id="slider-main-excerpt" class="text-slate-200 text-xs md:text-sm line-clamp-2 max-w-2xl drop-shadow mb-3">%s</p><div class="flex items-center gap-3"><a id="slider-main-link" href="/public/news/%s/%s/html" class="bg-[#ed1c24] hover:bg-red-700 text-white font-bold text-xs px-4 py-2 rounded transition flex items-center gap-1.5 shadow">Haberi Oku →</a></div></div></div>`, template.HTMLEscapeString(featCat), template.HTMLEscapeString(locale), template.HTMLEscapeString(featured.Slug), template.HTMLEscapeString(featured.Title), template.HTMLEscapeString(featLead), template.HTMLEscapeString(locale), template.HTMLEscapeString(featured.Slug))

		// Numbered Buttons Bar below Slider (1 - 10)
		_, _ = fmt.Fprint(w, `<div class="flex items-center justify-between gap-1.5 mt-2 bg-white p-2 rounded-xl border border-slate-200 shadow-sm overflow-x-auto whitespace-nowrap">`)
		for i, sl := range sliderItems {
			cURL := getCoverURL(sl)
			cCat := "GÜNDEM"
			if strings.TrimSpace(sl.Category) != "" {
				cCat = strings.ToUpper(sl.Category)
			}
			cLead := newsLeadItem(sl)
			activeCls := "w-9 h-8 flex items-center justify-center font-bold text-xs rounded bg-slate-900/80 text-white hover:bg-slate-700 transition cursor-pointer"
			if i == 0 {
				activeCls = "w-9 h-8 flex items-center justify-center font-black text-xs rounded bg-[#ed1c24] text-white scale-110 shadow cursor-pointer"
			}
			_, _ = fmt.Fprintf(w, `<button type="button" data-slide-btn="%d" data-index="%d" data-title="%s" data-img="%s" data-cat="%s" data-url="/public/news/%s/%s/html" data-excerpt="%s" onclick="setSlide(%d)" class="%s">%d</button>`, i, i, template.HTMLEscapeString(sl.Title), template.HTMLEscapeString(cURL), template.HTMLEscapeString(cCat), template.HTMLEscapeString(locale), template.HTMLEscapeString(sl.Slug), template.HTMLEscapeString(cLead), i, activeCls, i+1)
		}
		_, _ = fmt.Fprint(w, `</div></div>`)

		// Right: Yan Manşetler (lg:col-span-4)
		_, _ = fmt.Fprint(w, `<div class="lg:col-span-4 flex flex-col gap-4">`)
		sideCount := 0
		for _, sItem := range sliderItems {
			if sItem.ID == featured.ID {
				continue
			}
			sCat := "GÜNDEM"
			if strings.TrimSpace(sItem.Category) != "" {
				sCat = strings.ToUpper(sItem.Category)
			}
			_, _ = fmt.Fprintf(w, `<article class="bg-white rounded-xl border border-slate-200 overflow-hidden shadow-sm hover:shadow-md transition flex flex-col group flex-1"><div class="relative aspect-[16/9] overflow-hidden bg-slate-100">%s<span class="absolute top-2 left-2 bg-[#ed1c24] text-white text-[10px] font-black px-2 py-0.5 rounded shadow">%s</span></div><div class="p-3.5 flex flex-col justify-between flex-1"><h3 class="text-slate-900 font-extrabold text-sm line-clamp-2 group-hover:text-red-600 leading-snug mb-1"><a href="/public/news/%s/%s/html">%s</a></h3><p class="text-slate-500 text-xs line-clamp-2 mb-2 leading-relaxed">%s</p><div class="flex items-center justify-between text-[11px] text-slate-400 border-t border-slate-100 pt-2 mt-auto"><span class="inline-flex items-center gap-1"><svg class="w-3 h-3 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>%s</span><span>%d dk okuma</span></div></div></article>`, h.newsCoverMarkup(sItem), template.HTMLEscapeString(sCat), template.HTMLEscapeString(locale), template.HTMLEscapeString(sItem.Slug), template.HTMLEscapeString(sItem.Title), template.HTMLEscapeString(newsLeadItem(sItem)), template.HTMLEscapeString(sItem.AuthorName), readingMinutes(sItem.Body))
			sideCount++
			if sideCount >= 2 {
				break
			}
		}
		_, _ = fmt.Fprint(w, `</div></section>`)

		// 2. KATEGORİLİ HABER AKIŞI IZGARASI (GÜNÜN TÜM HABERLERİ)
		_, _ = fmt.Fprintf(w, `<section id="news-grid" class="mb-10"><div class="flex flex-col md:flex-row md:items-center justify-between gap-3 border-b-2 border-[#ed1c24] pb-3 mb-4"><div class="flex items-center gap-2"><span class="w-2.5 h-6 bg-[#ed1c24] rounded-sm"></span><h2 class="text-xl font-black text-slate-900 tracking-tight">GÜNÜN TÜM HABERLERİ</h2><span id="news-counter" class="text-xs font-bold text-slate-500 bg-slate-100 px-2 py-0.5 rounded-full ml-1">(%d)</span></div><div class="flex items-center gap-1.5 overflow-x-auto whitespace-nowrap scrollbar-none"><button onclick="selectCategory('all')" class="filter-pill px-3.5 py-1.5 rounded-full text-xs font-bold bg-[#ed1c24] text-white shadow-sm transition" data-filter-pill="all">Tümü</button><button onclick="selectCategory('gundem')" class="filter-pill px-3.5 py-1.5 rounded-full text-xs font-bold bg-white text-slate-700 hover:bg-slate-200 border border-slate-300 transition" data-filter-pill="gundem">Gündem</button><button onclick="selectCategory('ekonomi')" class="filter-pill px-3.5 py-1.5 rounded-full text-xs font-bold bg-white text-slate-700 hover:bg-slate-200 border border-slate-300 transition" data-filter-pill="ekonomi">Ekonomi</button><button onclick="selectCategory('teknoloji')" class="filter-pill px-3.5 py-1.5 rounded-full text-xs font-bold bg-white text-slate-700 hover:bg-slate-200 border border-slate-300 transition" data-filter-pill="teknoloji">Teknoloji</button><button onclick="selectCategory('dunya')" class="filter-pill px-3.5 py-1.5 rounded-full text-xs font-bold bg-white text-slate-700 hover:bg-slate-200 border border-slate-300 transition" data-filter-pill="dunya">Dünya</button><button onclick="selectCategory('spor')" class="filter-pill px-3.5 py-1.5 rounded-full text-xs font-bold bg-white text-slate-700 hover:bg-slate-200 border border-slate-300 transition" data-filter-pill="spor">Spor</button><button onclick="selectCategory('yasam')" class="filter-pill px-3.5 py-1.5 rounded-full text-xs font-bold bg-white text-slate-700 hover:bg-slate-200 border border-slate-300 transition" data-filter-pill="yasam">Yaşam</button></div></div><div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4" id="cards-container">`, len(items))

		for _, item := range items {
			catStr := "Gündem"
			if strings.TrimSpace(item.Category) != "" {
				catStr = item.Category
			}
			cardRead := readingMinutes(item.Body)
			_, _ = fmt.Fprintf(w, `<article data-news-card data-cat="%s" data-title="%s" class="bg-white rounded-xl border border-slate-200 overflow-hidden shadow-sm hover:shadow-lg transition duration-200 flex flex-col group"><div class="relative aspect-[16/9] overflow-hidden bg-slate-100">%s<span class="absolute top-2 left-2 bg-[#ed1c24] text-white text-[10px] font-black px-2 py-0.5 rounded shadow uppercase">%s</span></div><div class="p-3.5 flex flex-col justify-between flex-1"><h3 class="text-slate-900 font-extrabold text-sm line-clamp-2 group-hover:text-red-600 leading-snug mb-1.5"><a href="/public/news/%s/%s/html">%s</a></h3><p class="text-slate-500 text-xs line-clamp-2 leading-relaxed mb-3">%s</p><div class="mt-auto pt-2 border-t border-slate-100 flex items-center justify-between text-[11px] text-slate-400 font-medium"><span class="inline-flex items-center gap-1"><svg class="w-3 h-3 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>%s</span><span>%d dk okuma</span></div></div></article>`, template.HTMLEscapeString(strings.ToLower(catStr)), template.HTMLEscapeString(strings.ToLower(item.Title)), h.newsCoverMarkup(item), template.HTMLEscapeString(strings.ToUpper(catStr)), template.HTMLEscapeString(locale), template.HTMLEscapeString(item.Slug), template.HTMLEscapeString(item.Title), template.HTMLEscapeString(newsLeadItem(item)), template.HTMLEscapeString(item.AuthorName), cardRead)
		}
		_, _ = fmt.Fprint(w, `</div><div id="no-news-alert" style="display:none;" class="col-span-full py-12 text-center bg-white rounded-xl border border-slate-200 p-8 my-4"><div class="w-12 h-12 mx-auto mb-2 text-slate-400 flex items-center justify-center"><svg class="w-10 h-10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg></div><h3 class="text-base font-bold text-slate-800 mb-1">Bu kategoride henüz haber bulunamadı</h3><p class="text-xs text-slate-500 mb-4">Farklı bir kategori seçebilir veya tüm haberleri görüntüleyebilirsiniz.</p><button onclick="selectCategory('all')" class="bg-[#ed1c24] hover:bg-red-700 text-white font-bold text-xs px-4 py-2 rounded-lg transition">Tüm Haberleri Göster</button></div></section>`)

		// 3. ALT BÖLÜM: ÇOK OKUNANLAR (TOP 5) & HABER BÜLTENİ
		_, _ = fmt.Fprint(w, `<section class="grid grid-cols-1 lg:grid-cols-12 gap-6 mb-8"><div class="lg:col-span-7"><div class="flex items-center gap-2 border-b-2 border-[#ed1c24] pb-2 mb-3"><span class="w-2 h-5 bg-[#ed1c24] rounded-sm"></span><h3 class="text-base font-black text-slate-900 uppercase flex items-center gap-1.5"><svg class="w-4 h-4 text-red-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8.5 14.5A2.5 2.5 0 0 0 11 12c0-1.38-.5-2-1-3-1.072-2.143-.224-4.054 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 1 1-14 0c0-1.153.433-2.294 1-3a2.5 2.5 0 0 0 2.5 2.5z"></path></svg>Günün En Çok Okunanları</h3></div><div class="space-y-2.5">`)
		for i, tr := range sliderItems {
			if i >= 5 {
				break
			}
			tCat := "GÜNDEM"
			if strings.TrimSpace(tr.Category) != "" {
				tCat = strings.ToUpper(tr.Category)
			}
			_, _ = fmt.Fprintf(w, `<a href="/public/news/%s/%s/html" class="flex items-center gap-3 p-2.5 bg-white hover:bg-slate-50 rounded-xl border border-slate-200 shadow-sm transition group"><span class="w-7 h-7 rounded-full bg-[#ed1c24] text-white font-black text-xs flex items-center justify-center shrink-0 shadow-sm">%d</span><div class="flex-1 min-w-0"><span class="text-[10px] font-black text-[#ed1c24] uppercase">%s</span><h4 class="text-xs font-bold text-slate-900 group-hover:text-red-600 truncate">%s</h4></div><span class="text-[11px] text-slate-400 shrink-0 hidden sm:inline-flex items-center gap-1"><svg class="w-3 h-3 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>%s</span></a>`, template.HTMLEscapeString(locale), template.HTMLEscapeString(tr.Slug), i+1, template.HTMLEscapeString(tCat), template.HTMLEscapeString(tr.Title), template.HTMLEscapeString(tr.AuthorName))
		}
		_, _ = fmt.Fprint(w, `</div></div><div class="lg:col-span-5"><div class="bg-gradient-to-br from-slate-900 to-slate-800 text-white p-6 rounded-xl shadow-md h-full flex flex-col justify-between"><span class="bg-[#ed1c24] text-white text-[10px] font-black px-2.5 py-1 rounded uppercase tracking-wider self-start">E-BÜLTEN</span><div><h3 class="text-lg font-black mt-3 mb-1">Günün Önemli Gelişmelerini Kaçırmayın</h3><p class="text-xs text-slate-300 leading-relaxed mb-4">Her sabah en sıcak gündem, ekonomi ve teknoloji haberleri doğrudan e-posta kutunuzda.</p></div><form action="/member/register" method="get" class="flex gap-2"><input type="email" placeholder="E-posta adresiniz..." required class="flex-1 px-3.5 py-2 text-xs bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-red-600"><button type="submit" class="bg-[#ed1c24] hover:bg-red-700 text-white font-bold text-xs px-4 py-2 rounded-lg transition shrink-0">Abone Ol</button></form></div></div></section>`)

		// 4. CLIENT JAVASCRIPT: SLIDER AND CATEGORY FILTERING
		_, _ = fmt.Fprint(w, `<script>
let currentSlide = 0;
const slideBtns = document.querySelectorAll('[data-slide-btn]');
const slideCount = slideBtns.length;
let slideTimer = null;

function setSlide(idx) {
	const btn = document.querySelector('[data-slide-btn="' + idx + '"]');
	if (!btn) return;
	currentSlide = idx;

	const mainImg = document.getElementById('slider-main-img');
	const mainTag = document.getElementById('slider-main-tag');
	const mainTitle = document.getElementById('slider-main-title');
	const mainLink = document.getElementById('slider-main-link');
	const mainExcerpt = document.getElementById('slider-main-excerpt');

	if (mainImg && btn.dataset.img) mainImg.src = btn.dataset.img;
	if (mainTag && btn.dataset.cat) mainTag.textContent = btn.dataset.cat.toUpperCase();
	if (mainTitle && btn.dataset.title) {
		mainTitle.textContent = btn.dataset.title;
		mainTitle.href = btn.dataset.url;
	}
	if (mainLink && btn.dataset.url) mainLink.href = btn.dataset.url;
	if (mainExcerpt && btn.dataset.excerpt) mainExcerpt.textContent = btn.dataset.excerpt;

	slideBtns.forEach(function(b) {
		b.className = 'w-9 h-8 flex items-center justify-center font-bold text-xs rounded bg-slate-900/80 text-white hover:bg-slate-700 transition cursor-pointer';
	});
	btn.className = 'w-9 h-8 flex items-center justify-center font-black text-xs rounded bg-[#ed1c24] text-white scale-110 shadow cursor-pointer';
}

function nextSlide() {
	if (slideCount > 1) {
		setSlide((currentSlide + 1) % slideCount);
	}
}

function startSlideTimer() {
	if (slideTimer) clearInterval(slideTimer);
	slideTimer = setInterval(nextSlide, 4500);
}

const sliderBox = document.getElementById('main-slider-box');
if (sliderBox) {
	sliderBox.addEventListener('mouseenter', function() { clearInterval(slideTimer); });
	sliderBox.addEventListener('mouseleave', startSlideTimer);
	startSlideTimer();
}

function normTr(str) {
	if (!str) return '';
	return str.toLowerCase().replace(/ğ/g, 'g').replace(/ü/g, 'u').replace(/ş/g, 's').replace(/ı/g, 'i').replace(/ö/g, 'o').replace(/ç/g, 'c').trim();
}

function selectCategory(cat) {
	const norm = normTr(cat);

	document.querySelectorAll('[data-filter-pill]').forEach(function(pill) {
		if (normTr(pill.dataset.filterPill) === norm) {
			pill.className = 'filter-pill px-3.5 py-1.5 rounded-full text-xs font-bold bg-[#ed1c24] text-white shadow-sm transition';
		} else {
			pill.className = 'filter-pill px-3.5 py-1.5 rounded-full text-xs font-bold bg-white text-slate-700 hover:bg-slate-200 border border-slate-300 transition';
		}
	});

	document.querySelectorAll('#main-nav-links button').forEach(function(nb) {
		if (normTr(nb.dataset.cat) === norm) {
			nb.className = 'nav-cat-btn px-4 py-2.5 transition flex items-center gap-1 bg-red-800 text-white';
		} else {
			nb.className = 'nav-cat-btn px-4 py-2.5 hover:bg-red-700 transition text-white';
		}
	});

	let visibleCount = 0;
	const cards = document.querySelectorAll('[data-news-card]');
	cards.forEach(function(card) {
		const cardCat = normTr(card.dataset.cat);
		if (norm === 'all' || norm === '' || cardCat === norm) {
			card.style.display = 'flex';
			visibleCount++;
		} else {
			card.style.display = 'none';
		}
	});

	const counter = document.getElementById('news-counter');
	if (counter) counter.textContent = '(' + visibleCount + ')';

	const emptyBox = document.getElementById('no-news-alert');
	if (emptyBox) {
		emptyBox.style.display = visibleCount === 0 ? 'block' : 'none';
	}

	try {
		const url = new URL(window.location);
		if (norm === 'all') {
			url.searchParams.delete('category');
		} else {
			url.searchParams.set('category', cat);
		}
		window.history.replaceState({}, '', url);
	} catch(e) {}
}

function liveFilterNews(term) {
	const q = normTr(term);
	let visibleCount = 0;
	const cards = document.querySelectorAll('[data-news-card]');
	cards.forEach(function(card) {
		const title = normTr(card.dataset.title);
		const cat = normTr(card.dataset.cat);
		if (!q || title.indexOf(q) !== -1 || cat.indexOf(q) !== -1) {
			card.style.display = 'flex';
			visibleCount++;
		} else {
			card.style.display = 'none';
		}
	});
	const counter = document.getElementById('news-counter');
	if (counter) counter.textContent = '(' + visibleCount + ')';
	const emptyBox = document.getElementById('no-news-alert');
	if (emptyBox) {
		emptyBox.style.display = visibleCount === 0 ? 'block' : 'none';
	}
}

window.addEventListener('DOMContentLoaded', function() {
	const params = new URLSearchParams(window.location.search);
	const cat = params.get('category');
	if (cat) {
		selectCategory(cat);
	}
});
</script>`)
	}

	footer := localizePublicLinks(enterpriseFooterMarkup(), locale)
	if h.PublicSupport.Enabled || h.PublicContactSettings != nil {
		footer = strings.Replace(footer, `</body>`, h.socialLinksMarkup(r.Context())+h.floatingContact(r.Context(), locale)+`</body>`, 1)
	}
	_, _ = fmt.Fprintf(w, `</main>%s`, footer)
}

func (h Handler) socialLinksMarkup(ctx context.Context) string {
	if h.PublicContactSettings == nil {
		return ""
	}
	settings, err := h.PublicContactSettings(ctx)
	if err != nil {
		return ""
	}
	profiles := site.SocialProfilesFromSettings(settings)
	links := make([]designsystem.SocialLink, 0, len(profiles))
	for _, profile := range profiles {
		links = append(links, designsystem.SocialLink{Platform: profile.Platform, URL: profile.URL})
	}
	var output bytes.Buffer
	if designsystem.SocialLinks(designsystem.SocialLinksProps{Links: links}).Render(ctx, &output) != nil {
		return ""
	}
	return output.String()
}

func (h Handler) floatingContact(ctx context.Context, locale string) string {
	props := designsystem.FloatingContactProps{Phone: h.PublicSupport.Phone, WhatsAppURL: h.PublicSupport.WhatsAppURL, AriaLabel: "İletişim seçenekleri", PhoneLabel: "Ara", WhatsAppLabel: "WhatsApp", Position: "bottom-right"}
	enabled := h.PublicSupport.Enabled
	if h.PublicContactSettings != nil {
		settings, err := h.PublicContactSettings(ctx)
		if err != nil {
			return ""
		}
		if site.ContactWidgetConfigured(settings) {
			value := site.ContactWidgetFromSettings(settings)
			enabled = value.Enabled
			props.Phone, props.WhatsAppNumber, props.Message = value.Phone, value.WhatsAppNumber, value.Message
			props.Position, props.MobileOnly = value.Position, value.MobileOnly
			props.WhatsAppURL = ""
		}
	}
	if !enabled {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(locale), "en") {
		props.AriaLabel, props.PhoneLabel = "Contact options", "Call"
	}
	var output bytes.Buffer
	if err := designsystem.FloatingContact(props).Render(ctx, &output); err != nil {
		return ""
	}
	return output.String()
}

func localizePublicLinks(markup, locale string) string {
	locale = template.HTMLEscapeString(strings.TrimSpace(locale))
	if locale == "" {
		locale = "tr"
	}
	markup = strings.ReplaceAll(markup, "/public/pages/tr/", "/public/pages/"+locale+"/")
	return strings.ReplaceAll(markup, "/public/news/tr/", "/public/news/"+locale+"/")
}

func newsDocumentStart(locale, title string, navigation template.HTML) string {
	navigationHTML := string(navigation)
	if navigationHTML == "" {
		navigationHTML = `<button onclick="selectCategory('all')" class="nav-cat-btn px-4 py-2.5 transition flex items-center gap-1 bg-red-800 text-white font-bold" data-cat="all">ANA SAYFA</button><button onclick="selectCategory('gundem')" class="nav-cat-btn px-4 py-2.5 hover:bg-red-700 transition font-bold" data-cat="gundem">GÜNDEM</button><button onclick="selectCategory('ekonomi')" class="nav-cat-btn px-4 py-2.5 hover:bg-red-700 transition font-bold" data-cat="ekonomi">EKONOMİ</button><button onclick="selectCategory('teknoloji')" class="nav-cat-btn px-4 py-2.5 hover:bg-red-700 transition font-bold" data-cat="teknoloji">TEKNOLOJİ</button><button onclick="selectCategory('dunya')" class="nav-cat-btn px-4 py-2.5 hover:bg-red-700 transition font-bold" data-cat="dunya">DÜNYA</button><button onclick="selectCategory('spor')" class="nav-cat-btn px-4 py-2.5 hover:bg-red-700 transition font-bold" data-cat="spor">SPOR</button><button onclick="selectCategory('yasam')" class="nav-cat-btn px-4 py-2.5 hover:bg-red-700 transition font-bold" data-cat="yasam">YAŞAM</button>`
	}
	return `<!doctype html><html lang="` + template.HTMLEscapeString(locale) + `" dir="` + canonical.Direction(locale) + `"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>` + template.HTMLEscapeString(title) + ` · HYDRO HABER</title><link rel="stylesheet" href="/theme/static/theme.css?v=00013d55"><link rel="preconnect" href="https://fonts.googleapis.com"><link rel="preconnect" href="https://fonts.gstatic.com" crossorigin><link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&display=swap" rel="stylesheet"><style>` + newsThemeMasterCSS + `</style></head><body class="bg-slate-50 text-slate-800 antialiased min-h-screen flex flex-col font-sans"><div class="bg-slate-900 text-slate-300 text-xs border-b border-slate-800"><div class="max-w-7xl mx-auto px-4 py-1.5 flex flex-wrap items-center justify-between gap-2"><div class="flex items-center gap-3"><span class="inline-flex items-center gap-1.5 font-medium text-slate-200"><svg class="w-3.5 h-3.5 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect><line x1="16" y1="2" x2="16" y2="6"></line><line x1="8" y1="2" x2="8" y2="6"></line><line x1="3" y1="10" x2="21" y2="10"></line></svg>` + template.HTMLEscapeString(turkishDate(time.Now(), true)) + `</span></div><div class="flex items-center gap-3"><a href="/member/login" class="hover:text-white transition inline-flex items-center gap-1 font-medium text-xs"><svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>Okur Girişi</a><span class="text-slate-700">|</span><a href="/member/register" class="hover:text-white transition text-xs font-medium">Kayıt Ol</a><span class="text-slate-700">|</span><a href="/admin" class="text-amber-400 hover:text-amber-300 font-bold transition inline-flex items-center gap-1 text-xs"><svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>Studio Paneli ↗</a></div></div></div><header class="bg-white border-b border-slate-200 shadow-xs sticky top-0 z-40"><div class="max-w-7xl mx-auto px-4 py-3 flex flex-wrap items-center justify-between gap-4"><a href="/" class="flex items-center gap-2 group text-decoration-none shrink-0"><span class="bg-red-600 text-white font-black text-2xl px-3 py-0.5 rounded tracking-tighter shadow-xs group-hover:bg-red-700 transition">HYDRO</span><span class="text-slate-900 font-black text-2xl tracking-tighter">HABER</span><span class="hidden md:inline-block border-l-2 border-slate-200 pl-3 ml-1 text-[11px] font-bold text-slate-500 uppercase tracking-wider">Doğru · Hızlı · Bağımsız</span></a><div class="flex-1 max-w-md mx-4 hidden md:block"><div class="relative"><input id="news-live-search" type="text" placeholder="Haberlerde ara (başlık, konu veya yazar)..." class="w-full pl-9 pr-4 py-1.5 text-xs bg-slate-100 border border-slate-200 rounded-full focus:bg-white focus:outline-none focus:ring-2 focus:ring-red-600 transition" oninput="liveFilterNews(this.value)"><svg class="w-4 h-4 text-slate-400 absolute left-3 top-2.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg></div></div><div class="flex items-center gap-3"><a href="/#news-grid" class="inline-flex items-center gap-1.5 bg-red-600 hover:bg-red-700 text-white text-xs font-black px-3.5 py-1.5 rounded-full shadow-xs transition"><span class="w-2 h-2 rounded-full bg-white animate-pulse"></span>CANLI AKIŞ</a><a href="/public/pages/tr/about/html" class="text-xs font-bold text-slate-600 hover:text-slate-900 px-2 py-1">Hakkında</a></div></div><nav class="bg-red-600 text-white shadow"><div class="max-w-7xl mx-auto px-4 flex items-center justify-between overflow-x-auto whitespace-nowrap"><div class="flex items-center text-xs font-black tracking-wide" id="main-nav-links">` + navigationHTML + `</div><div class="hidden lg:flex items-center text-xs font-bold text-red-100 pr-2"><span>7/24 KESİNTİSİZ HABER</span></div></div></nav></header><div class="bg-red-50 border-b border-red-200 py-2"><div class="max-w-7xl mx-auto px-4 flex items-center gap-3"><div class="bg-red-600 text-white text-[11px] font-black px-2.5 py-1 rounded inline-flex items-center gap-1.5 shrink-0 shadow-xs"><span class="w-2 h-2 rounded-full bg-white animate-pulse"></span>SON DAKİKA</div><div class="flex-1 overflow-hidden"><a id="ticker-headline-link" href="/#news-grid" class="text-xs md:text-sm font-bold text-slate-800 hover:text-red-600 transition truncate block">HydroGo 2.0 Yayında: Tip Güvenli Mimari ve Editoryal Haber Odası Devrede!</a></div><span class="text-xs text-slate-500 font-semibold hidden md:inline shrink-0">Hızlı Akış</span></div></div>`
}

func (h Handler) themeNavigation(ctx context.Context, locale string) template.HTML {
	if h.NavigationProvider == nil {
		return ""
	}
	items, err := h.NavigationProvider(ctx, locale)
	if err != nil {
		return ""
	}
	markup, err := v1.RenderThemeNavigation(items)
	if err != nil {
		return ""
	}
	return markup
}

func enterpriseFooterMarkup() string {
	return `<footer class="bg-slate-900 text-slate-400 py-12 px-4 mt-12 border-t border-slate-800"><div class="max-w-7xl mx-auto"><div class="grid grid-cols-1 md:grid-cols-4 gap-8 pb-10 border-b border-slate-800"><div><div class="flex items-center gap-2 mb-3"><span class="bg-red-600 text-white font-black text-xl px-2.5 py-0.5 rounded tracking-tighter">HYDRO</span><span class="text-white font-black text-xl tracking-tighter">HABER</span></div><p class="text-xs text-slate-400 leading-relaxed mb-4">Türkiye'nin ve dünyanın en önemli gelişmelerini tarafsız, doğru ve hızlı bir şekilde sunan bağımsız dijital haber portalı.</p><div class="flex gap-2"><span class="bg-slate-800 text-slate-300 text-[11px] font-medium px-2.5 py-1 rounded inline-flex items-center gap-1"><svg class="w-3 h-3 text-red-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg>Go Altyapısı</span><span class="bg-slate-800 text-slate-300 text-[11px] font-medium px-2.5 py-1 rounded inline-flex items-center gap-1"><svg class="w-3 h-3 text-emerald-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg>Tip Güvenli</span></div></div><div><h4 class="text-white font-bold text-sm mb-3 uppercase tracking-wider">Kategoriler</h4><ul class="space-y-2 text-xs"><li><button onclick="selectCategory('gundem')" class="hover:text-white transition text-left">Gündem Haberleri</button></li><li><button onclick="selectCategory('ekonomi')" class="hover:text-white transition text-left">Ekonomi & Finans</button></li><li><button onclick="selectCategory('teknoloji')" class="hover:text-white transition text-left">Teknoloji & Bilim</button></li><li><button onclick="selectCategory('dunya')" class="hover:text-white transition text-left">Dünya Gelişmeleri</button></li><li><button onclick="selectCategory('spor')" class="hover:text-white transition text-left">Spor Haberleri</button></li></ul></div><div><h4 class="text-white font-bold text-sm mb-3 uppercase tracking-wider">Kurumsal</h4><ul class="space-y-2 text-xs"><li><a href="/public/pages/tr/about/html" class="hover:text-white transition">Hakkımızda & İlkeler</a></li><li><a href="/member/login" class="hover:text-white transition">Okur Girişi</a></li><li><a href="/member/register" class="hover:text-white transition">Üye Ol</a></li><li><a href="/admin" class="hover:text-white transition">Studio Yönetim Paneli</a></li><li><a href="/api/news?locale=tr" class="text-amber-400 hover:text-amber-300 font-mono">REST API / RSS</a></li></ul></div><div><h4 class="text-white font-bold text-sm mb-3 uppercase tracking-wider">İletişim & Künye</h4><p class="text-xs leading-relaxed text-slate-400 mb-3">HydroGo Medya & Yayıncılık A.Ş.<br>Maslak, İstanbul / Türkiye<br>haber@haber.test</p><p class="text-[11px] text-slate-500">Basın ahlak ilkelerine uymayı taahhüt ederiz.</p></div></div><div class="pt-6 flex flex-col sm:flex-row items-center justify-between gap-4 text-xs text-slate-500"><p>© 2026 HYDRO HABER. Tüm hakları saklıdır. İçerikler kaynak gösterilmeden iktibas edilemez.</p><div class="flex gap-4"><a href="/public/pages/tr/about/html" class="hover:text-slate-400">Gizlilik Politikası</a><span>•</span><a href="/public/pages/tr/about/html" class="hover:text-slate-400">Kullanım Şartları</a><span>•</span><a href="/admin" class="hover:text-slate-400">Yönetim</a></div></div></div></footer><script src="/theme/static/theme.js?v=60d7340a"></script></body></html>`
}

func newsLead(body string) string {
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		body = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(ui.RichTextHTML(body), " ")
	}
	body = strings.Join(strings.Fields(body), " ")
	if len([]rune(body)) > 160 {
		return string([]rune(body)[:160]) + "…"
	}
	return body
}

func readingMinutes(text string) int {
	return max(1, (len(strings.Fields(text))+199)/200)
}

func newsLeadItem(item news.Item) string {
	if strings.TrimSpace(item.Excerpt) != "" {
		return strings.TrimSpace(item.Excerpt)
	}
	return newsLead(item.Body)
}

func (h Handler) newsCoverMarkup(item news.Item) template.HTML {
	key := template.URLQueryEscaper(item.CoverMediaID)
	src := "/media/" + key + "?variant=card"
	fallback := "/media/" + key
	if h.Media != nil {
		if publicURL, err := h.Media.PublicURL(item.CoverMediaID); err == nil && strings.TrimSpace(publicURL) != "" {
			src = publicURL
			fallback = publicURL
		}
	}
	if strings.TrimSpace(item.CoverMediaID) == "" {
		return template.HTML(`<div class="w-full h-full bg-slate-100 flex items-center justify-center text-slate-400 select-none"><svg class="w-8 h-8 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg></div>`)
	}
	return template.HTML(`<img class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" src="` + template.HTMLEscapeString(src) + `" alt="` + template.HTMLEscapeString(item.Title) + `" loading="lazy" onerror="this.onerror=null;this.src='` + template.HTMLEscapeString(fallback) + `';">`)
}

func newsMetaBadgesMarkup(item news.Item) template.HTML {
	var b strings.Builder
	if strings.TrimSpace(item.Category) != "" {
		b.WriteString(`<span class="meta-pill-category">` + template.HTMLEscapeString(item.Category) + `</span>`)
	}
	if strings.TrimSpace(item.AuthorName) != "" {
		b.WriteString(`<span class="meta-pill-text">` + template.HTMLEscapeString(item.AuthorName) + `</span>`)
	}
	return template.HTML(b.String())
}

func newsCategoryMeta(item news.Item) template.HTML {
	if strings.TrimSpace(item.Category) == "" {
		return ""
	}
	return template.HTML(`<span class="featured-meta-separator">·</span><span>` + template.HTMLEscapeString(item.Category) + `</span>`)
}

func newsTagsMarkup(item news.Item) template.HTML {
	if strings.TrimSpace(item.Tags) == "" {
		return ""
	}
	raw := strings.Split(item.Tags, ",")
	var tagsHTML strings.Builder
	for _, t := range raw {
		tag := strings.TrimSpace(t)
		if tag != "" {
			tagsHTML.WriteString(`<span class="news-tag-pill">#` + template.HTMLEscapeString(tag) + `</span>`)
		}
	}
	if tagsHTML.Len() == 0 {
		return ""
	}
	return template.HTML(`<div class="news-tags-wrapper"><span class="tags-label">Etiketler:</span><div class="tags-list">` + tagsHTML.String() + `</div></div>`)
}

func newsKeyTakeawaysMarkup(item news.Item) template.HTML {
	if strings.TrimSpace(item.KeyTakeaways) == "" {
		return ""
	}
	lines := strings.Split(item.KeyTakeaways, "\n")
	var itemsHTML strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		trimmed = strings.TrimPrefix(trimmed, "•")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed != "" {
			itemsHTML.WriteString(`<li><svg class="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg><span class="takeaway-text">` + template.HTMLEscapeString(trimmed) + `</span></li>`)
		}
	}
	if itemsHTML.Len() == 0 {
		return ""
	}
	return template.HTML(`<div class="news-key-takeaways"><div class="takeaways-header"><svg class="w-4 h-4 text-red-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon></svg><h3>Önemli Çıkarımlar & Özet</h3></div><ul class="takeaways-list">` + itemsHTML.String() + `</ul></div>`)
}

func renderParagraphs(body string) string {
	return ui.RichTextHTML(body)
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
