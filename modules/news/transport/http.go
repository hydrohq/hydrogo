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
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
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
	if !editorAllowed(r) {
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
					title += fmt.Sprintf(" ⚠ %d eksik görsel", missing)
				}
			}
		}
		actions := []adminstudio.Action{{Label: "Düzenle", Href: "/admin/news/" + item.ID}}
		if item.Status == news.StatusPublished {
			publicPath := "/public/news/" + url.PathEscape(item.Locale) + "/" + url.PathEscape(item.Slug) + "/html"
			actions = append(actions, adminstudio.Action{Label: "Copy live URL", CopyText: strings.TrimRight(strings.TrimSpace(h.PublicOrigin), "/") + publicPath})
		}
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
		if err := adminstudio.RenderList(w, adminstudio.ListData{Collection: collection, Rows: rows, StatusFilter: r.URL.Query().Get("status")}); err != nil {
			http.Error(w, "news admin unavailable", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Haberler", "", security.NonceFromContext(r.Context()), "news"); err != nil {
		return
	}
	if err := ui.ListViewSwitcher("admin-content-list", "hydro-view-content").Render(r.Context(), w); err != nil {
		return
	}
	_, _ = fmt.Fprint(w, `<section class="studio-card"><div class="table-wrap" id="admin-content-list" data-view-mode="table">`)
	if viewsAvailable {
		_ = designsystem.ContentViewSort(designsystem.ContentViewSortProps{Path: "/admin/news", Status: r.URL.Query().Get("status"), Selected: r.URL.Query().Get("sort")}).Render(r.Context(), w)
	}
	if err := adminstudio.RenderList(w, adminstudio.ListData{Collection: collection, Rows: rows, StatusFilter: r.URL.Query().Get("status")}); err != nil {
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
	if err := ui.AdminStartForRequest(w, r, "Yeni haber", "Yayın merkezi", security.NonceFromContext(r.Context()), "news"); err != nil {
		return
	}
	var leftFields, rightMetaFields, rightSEOFields []adminstudio.Field
	for _, f := range h.AdminCollection.Fields {
		switch f.ID {
		case "title", "slug", "cover_media_id", "excerpt", "key_takeaways", "body":
			leftFields = append(leftFields, f)
		case "featured", "category", "author_name", "tags", "expires_at":
			rightMetaFields = append(rightMetaFields, f)
		case "seo_title", "seo_description":
			rightSEOFields = append(rightSEOFields, f)
		}
	}
	if !hasAdminField(rightMetaFields, "expires_at") {
		rightMetaFields = append(rightMetaFields, adminstudio.Field{ID: "expires_at", Label: "Otomatik yayından kalkma", Kind: adminstudio.FieldDate, Widget: "datetime-local"})
	}
	values := map[string]string{"locale": h.defaultLocale(), "expires_at": ""}
	csrf := security.CSRFTokenFromContext(r.Context())
	nonce := security.NonceFromContext(r.Context())

	var leftBuf bytes.Buffer
	leftCol := adminstudio.Collection{ID: "news_left", Label: "Haber İçeriği", Fields: leftFields, Actions: []adminstudio.Action{{Label: "Kaydet", Href: "/admin/news"}}}
	if err := adminstudio.RenderCreateForm(&leftBuf, adminstudio.FormData{Collection: leftCol, Action: "/admin/news", CSRFToken: csrf, Nonce: nonce, Values: values, HideSubmit: true}); err != nil {
		return
	}

	var rightMetaBuf bytes.Buffer
	rightMetaCol := adminstudio.Collection{ID: "news_meta", Label: "Genel Bilgiler", Fields: rightMetaFields, Actions: []adminstudio.Action{{Label: "Kaydet", Href: "/admin/news"}}}
	if err := adminstudio.RenderCreateForm(&rightMetaBuf, adminstudio.FormData{Collection: rightMetaCol, Action: "/admin/news", CSRFToken: csrf, Nonce: nonce, Values: values, HideSubmit: true}); err != nil {
		return
	}

	var rightSEOBuf bytes.Buffer
	rightSEOCol := adminstudio.Collection{ID: "news_seo", Label: "SEO & Arama Motoru", Fields: rightSEOFields, Actions: []adminstudio.Action{{Label: "Kaydet", Href: "/admin/news"}}}
	if err := adminstudio.RenderCreateForm(&rightSEOBuf, adminstudio.FormData{Collection: rightSEOCol, Action: "/admin/news", CSRFToken: csrf, Nonce: nonce, Values: values, HideSubmit: true}); err != nil {
		return
	}

	_, _ = fmt.Fprintf(w, `<div class="editor-top-nav"><div class="editor-top-info"><strong class="editor-live-title">[Başlıksız]</strong><span class="editor-live-sub">Yeni bir Haber oluşturuluyor</span></div><div class="editor-top-actions"><button type="submit" form="news-form" class="button">Taslağı kaydet</button></div></div><form id="news-form" class="studio-form" method="post" action="/admin/news"><div class="editor-split-container"><div class="editor-main-col"><section class="editor-card"><div class="editor-card-title"><span>Haber İçeriği</span></div>%s</section></div><div class="editor-side-col"><section class="editor-card"><div class="editor-card-title"><span>Yayın Bilgileri</span><span class="card-tag">YENİ</span></div><div class="editor-side-fields"><div class="editor-side-status"><span>Durum</span><strong>Taslak</strong></div><div class="editor-side-status"><span>Dil</span><strong>%s</strong></div><p class="muted" style="margin:8px 0 0;font-size:.78rem">Kaydedildikten sonra zamanlama ve yayına alma butonları aktifleşir.</p></div></section><section class="editor-card"><div class="editor-card-title"><span>Genel Bilgiler</span></div><div class="editor-side-fields">%s</div></section><section class="editor-card"><div class="editor-card-title"><span>SEO &amp; Arama Motoru</span></div><div class="editor-side-fields">%s</div></section></div></div><input type="hidden" name="locale" value="%s"><input type="hidden" name="_csrf" value="%s"></form><script nonce="%s">(()=>{const titleInp=document.querySelector('#news-form input[name="title"]'),liveTitle=document.querySelector('.editor-live-title');if(titleInp&&liveTitle){titleInp.addEventListener('input',()=>{liveTitle.textContent=titleInp.value.trim()||'[Başlıksız]';});}})();</script>`, template.HTML(leftBuf.String()), template.HTMLEscapeString(h.defaultLocale()), template.HTML(rightMetaBuf.String()), template.HTML(rightSEOBuf.String()), template.HTMLEscapeString(h.defaultLocale()), template.HTMLEscapeString(csrf), template.HTMLEscapeString(nonce))
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
	if err := ui.AdminStartForRequest(w, r, "Taxonomy", "Haber sınıflandırması", security.NonceFromContext(r.Context()), "news"); err != nil {
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
		writeError(w, err)
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
		writeError(w, err)
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
		writeError(w, err)
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
	writeJSON(w, http.StatusOK, items)
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
	item := news.Item{ID: r.PathValue("id"), Locale: value.Locale, Slug: value.Slug, Title: value.Title, Body: value.Body, CoverMediaID: value.CoverMediaID, Excerpt: value.Excerpt, KeyTakeaways: value.KeyTakeaways, Category: value.Category, AuthorName: value.AuthorName, Featured: value.Featured, Tags: value.Tags, SEOTitle: value.SEOTitle, SEODescription: value.SEODescription, ExpiresAt: value.ExpiresAt, Status: news.StatusDraft}
	if err := h.validateNewsMedia(r.Context(), item); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
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
	writeJSON(w, http.StatusOK, item)
}

// publicGetHTML exposes a human-facing counterpart to the JSON public route.
// The JSON endpoint remains stable for headless clients.
func (h Handler) publicGetHTML(w http.ResponseWriter, r *http.Request) {
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
	item, err := h.Service.GetPublishedByRoute(r.Context(), h.DB, locale, slug)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseRequired) {
			http.Error(w, "news unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}

	// Fetch popular/recent news items for the sticky sidebar
	var sidebarItems []news.Item
	if repository, ok := h.Service.Repository.(relatedListRepository); ok {
		if list, err := repository.ListRelatedPublished(r.Context(), h.DB, item.ID, locale, 4); err == nil {
			sidebarItems = list
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	consentNotice := h.analyticsConsentNotice(r, locale)
	if h.AnalyticsEnabled {
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
			`<a class="trending-item" href="/public/news/%s/%s/html"><span class="trending-number">%02d</span><div class="trending-content"><div class="trending-kicker">%s</div><h4 class="trending-title">%s</h4></div></a>`,
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
		relatedCards.WriteString(`<section class="related-content" aria-labelledby="related-content-title"><h2 id="related-content-title">Bunlar da ilginizi çekebilir</h2><div class="related-content-grid">`)
		for _, related := range sidebarItems {
			if related.Slug == item.Slug {
				continue
			}
			fmt.Fprintf(&relatedCards, `<article class="related-content-card">%s<h3><a href="/public/news/%s/%s/html">%s</a></h3><p>%d dk okuma</p></article>`, h.newsCoverMarkup(related), template.HTMLEscapeString(locale), template.HTMLEscapeString(related.Slug), template.HTMLEscapeString(related.Title), readingMinutes(related.Excerpt))
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
	nonce := template.HTMLEscapeString(security.NonceFromContext(r.Context()))
	readingEnhancement := fmt.Sprintf(`<style nonce="%s">%s</style><script nonce="%s">%s</script>`, nonce, ui.PublicArticleCSS, nonce, ui.PublicArticleJS)
	var helpfulFeedback bytes.Buffer
	if h.Feedback != nil {
		scope, scopeErr := tenant.FromContext(r.Context())
		if scopeErr == nil || !h.RequireTenantScope {
			tenantID := "default"
			if scopeErr == nil {
				tenantID = scope.TenantID
			}
			totals, _ := h.Feedback.FeedbackTotals(r.Context(), tenantID, "news", item.ID)
			if err := designsystem.HelpfulFeedback(designsystem.HelpfulFeedbackProps{ContentType: "news", ContentID: item.ID, ElementID: "article-feedback", CSRF: security.CSRFTokenFromContext(r.Context()), Nonce: nonce, Up: totals.Up, Down: totals.Down, LoadHTMX: true}).Render(r.Context(), &helpfulFeedback); err != nil {
				helpfulFeedback.Reset()
			}
		}
	}
	footer := localizePublicLinks(enterpriseFooterMarkup(), locale)
	if consentNotice != "" || h.PublicSupport.Enabled || h.PublicContactSettings != nil {
		footer = strings.Replace(footer, `</body>`, consentNotice+h.socialLinksMarkup(r.Context())+h.floatingContact(r.Context(), locale)+`</body>`, 1)
	}

	_, _ = fmt.Fprintf(w, `<main class="news-portal news-article-page"><nav class="news-breadcrumb" aria-label="Breadcrumb"><a href="/public/news/%s/html">← Haber Odası</a><span class="sep">/</span><span class="current">%s</span></nav><div class="news-detail-layout"><div class="news-article-column"><article class="news-article-card"><div class="news-article-kicker">HYDROGO NEWSROOM · %s</div><h1 class="news-article-title">%s</h1><div class="news-meta-bar">%s</div>%s%s%s%s<div class="news-article-body rich-article-content article-reading-body">%s</div>%s%s%s<div class="news-share-box"><span class="share-label">Haberi Paylaş:</span><div class="share-actions"><a class="btn-secondary share-btn" href="mailto:?subject=%s&amp;body=/public/news/%s/%s/html">✉ E-Posta</a><button class="btn-secondary share-btn" type="button" data-copy-url>🔗 Bağlantıyı Kopyala</button></div></div></article></div><aside class="news-sidebar-sticky"><div class="sidebar-widget"><h3><span class="widget-icon">🔗</span> Bununla ilgili</h3><div class="trending-news-list">%s</div></div><div class="sidebar-widget widget-card-promo"><h3><span class="widget-icon">🚀</span> HydroGo Çözümleri</h3><p>Sözleşmeli modüler mimari ve yüksek performanslı Go altyapısı ile projelerinizi büyütün.</p><a class="btn-primary" href="/shop">Mağazayı Ziyaret Et →</a></div></aside></div></main>%s%s`,
		template.HTMLEscapeString(locale),
		template.HTMLEscapeString(item.Title),
		strings.ToUpper(template.HTMLEscapeString(locale)),
		template.HTMLEscapeString(item.Title),
		newsMetaBadgesMarkup(item),
		h.newsCoverMarkup(item),
		newsKeyTakeawaysMarkup(item),
		readingTools.String(),
		template.HTML(tocHTML),
		template.HTML(articleHTML),
		newsTagsMarkup(item),
		helpfulFeedback.String(),
		template.HTML(relatedCards.String()),
		template.URLQueryEscaper(item.Title),
		template.HTMLEscapeString(locale),
		template.HTMLEscapeString(item.Slug),
		trendingHTML.String(),
		readingEnhancement,
		footer,
	)
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
	documentStart := newsDocumentStart(locale, "Haberler & Duyurular", h.themeNavigation(r.Context(), locale))
	if h.PublicSupport.Enabled || h.PublicContactSettings != nil {
		documentStart = strings.Replace(documentStart, `</head>`, `<link rel="stylesheet" href="/designsystem/static/designsystem.css"></head>`, 1)
	}
	_, _ = fmt.Fprint(w, localizePublicLinks(documentStart, locale))
	_, _ = fmt.Fprintf(w, `<main data-testid="public-news" class="news-portal"><section class="news-hero-banner"><div class="news-hero-content"><div class="news-tag-badge">HYDROGO YAYIN MERKEZİ</div><h1>Haberler, Duyurular ve Güncellemeler</h1><p>Panelden yönetilen tüm editoryal içerikler, ürün notları ve sistem duyuruları anlık olarak burada listelenir.</p><div class="news-hero-actions"><a class="btn-primary" href="/shop">Mağazayı Keşfet →</a><a class="btn-secondary" href="/admin/news">Admin Panelinde Düzenle ↗</a></div></div></section>`)
	if len(items) == 0 {
		_, _ = fmt.Fprint(w, `<section class="news-empty-state"><div class="empty-icon">📰</div><h2>Henüz yayınlanmış bir haber bulunmuyor</h2><p>Admin panelinden yeni bir haber oluşturup yayınlayarak burada görüntüleyebilirsiniz.</p><a class="btn-primary" href="/admin/news">Yeni Haber Ekle</a></section>`)
	} else {
		// Featured item
		featured := items[0]
		for _, candidate := range items {
			if candidate.Featured {
				featured = candidate
				break
			}
		}
		featuredKicker := "ÖNE ÇIKAN İÇERİK"
		if strings.TrimSpace(featured.Category) != "" {
			featuredKicker = strings.ToUpper(featured.Category)
		}
		_, _ = fmt.Fprintf(w, `<section class="news-featured-section"><div class="section-badge" style="margin-bottom:12px;">SON YAYINLANAN MANŞET</div><article class="featured-card">%s<div class="featured-card-body"><div class="featured-kicker">%s</div><h2><a href="/public/news/%s/%s/html">%s</a></h2><p class="featured-lead">%s</p><div class="featured-footer"><div class="featured-meta"><span>%s</span>%s</div><a class="read-more-btn" href="/public/news/%s/%s/html">Haberi Oku →</a></div></div></article></section>`, h.newsCoverMarkup(featured), template.HTMLEscapeString(featuredKicker), template.HTMLEscapeString(locale), template.HTMLEscapeString(featured.Slug), template.HTMLEscapeString(featured.Title), template.HTMLEscapeString(newsLeadItem(featured)), template.HTMLEscapeString(featured.AuthorName), newsCategoryMeta(featured), template.HTMLEscapeString(locale), template.HTMLEscapeString(featured.Slug))

		// Grid for all items
		_, _ = fmt.Fprintf(w, `<section class="news-grid-section" style="margin-top:48px;"><div class="section-header" style="margin-bottom:24px;"><div><div class="section-badge">TÜM YAYINLAR</div><h2 style="font-size:1.8rem;font-weight:800;color:var(--theme-ink);margin:4px 0 0;">Haber Akışı (%d)</h2></div><span class="section-desc" style="color:var(--theme-muted);font-size:.95rem;">Panelde aktif olan tüm kayıtlar</span></div><div class="news-cards-grid">`, len(items))
		for _, item := range items {
			cardKicker := "YAYIN"
			if strings.TrimSpace(item.Category) != "" {
				cardKicker = strings.ToUpper(item.Category)
			}
			_, _ = fmt.Fprintf(w, `<article class="news-card">%s<div class="news-card-body"><div class="card-header"><span class="card-kicker">%s</span><span class="card-author">%s</span></div><h3 class="card-title"><a href="/public/news/%s/%s/html">%s</a></h3><p class="card-lead">%s</p><div class="card-footer"><span class="card-category">%s</span><a class="card-link" href="/public/news/%s/%s/html">Detaylar →</a></div></div></article>`, h.newsCoverMarkup(item), template.HTMLEscapeString(cardKicker), template.HTMLEscapeString(item.AuthorName), template.HTMLEscapeString(locale), template.HTMLEscapeString(item.Slug), template.HTMLEscapeString(item.Title), template.HTMLEscapeString(newsLeadItem(item)), template.HTMLEscapeString(item.Category), template.HTMLEscapeString(locale), template.HTMLEscapeString(item.Slug))
		}
		_, _ = fmt.Fprint(w, `</div></section>`)
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
		navigationHTML = `<a href="/" class="nav-item">Ana Sayfa</a><a href="/public/pages/tr/showcase/html" class="nav-item">Showcase</a><a href="/public/pages/tr/about/html" class="nav-item">Hakkında</a><a href="/public/news/tr/html" class="nav-item is-active">Haberler</a><a href="/shop" class="nav-item">Mağaza</a>`
	}
	return fmt.Sprintf(`<!doctype html><html lang="%s" dir="%s"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>%s · HydroGo Newsroom</title><link rel="stylesheet" href="/theme/static/theme.css?v=e0fcce17"></head><body class="theme-corporate-body"><header class="news-header"><div class="header-container"><a class="news-logo" href="/" aria-label="HydroGo Ana Sayfa"><div class="logo-mark"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg></div><div class="logo-text"><span>HYDROGO</span></div></a><button class="nav-toggle" type="button" aria-expanded="false" aria-controls="primary-nav" aria-label="Menüyü aç"><span class="hamburger-bar"></span><span class="hamburger-bar"></span><span class="hamburger-bar"></span></button><nav id="primary-nav" class="news-nav" aria-label="Ana navigasyon">%s<a href="/member/login" class="nav-item nav-login-link">Giriş</a><a class="nav-admin-btn" href="/admin">Studio Paneli ↗</a></nav></div></header>`, template.HTMLEscapeString(locale), canonical.Direction(locale), template.HTMLEscapeString(title), navigationHTML)
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
	return `<footer class="news-footer"><div class="footer-container"><div class="footer-brand-col"><div class="footer-logo"><div class="logo-mark"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg></div><strong>HYDROGO <span>PLATFORM</span></strong></div><p class="footer-summary">Yüksek performanslı Go altyapısı, katı mimari sınırlar ve sözleşme testli çift veritabanı desteği ile kurumsal içerik ve e-ticaret platformu.</p><div class="footer-badge-list"><span class="f-badge">⚡ Go 1.22+</span><span class="f-badge">🛡️ Tip Güvenli</span><span class="f-badge">📦 Modüler Core</span></div></div><div class="footer-nav-col"><h4>Platform</h4><nav><a href="/">Ana Sayfa</a><a href="/public/pages/tr/showcase/html">Showcase</a><a href="/public/pages/tr/about/html">Hakkında</a><a href="/public/news/tr/html">Haberler & Blog</a></nav></div><div class="footer-nav-col"><h4>Çözümler</h4><nav><a href="/shop">E-Ticaret Mağazası</a><a href="/admin">Admin Studio Paneli</a><a href="/api/v1/pages">REST API v1</a></nav></div></div><div class="footer-bottom-bar"><p>© 2026 HydroGo Platform. Tüm hakları saklıdır.</p><div class="footer-sub-links"><a href="/public/pages/tr/about/html">Güvenlik & Felsefe</a><span>•</span><a href="/admin">Yönetim Girişi</a></div></div></footer><script src="/theme/static/theme.js?v=60d7340a"></script></body></html>`
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
	if strings.TrimSpace(item.CoverMediaID) == "" {
		return ""
	}
	key := template.URLQueryEscaper(item.CoverMediaID)
	src := "/media/" + key + "?variant=card"
	fallback := "/media/" + key
	if h.Media != nil {
		if publicURL, err := h.Media.PublicURL(item.CoverMediaID); err == nil && strings.TrimSpace(publicURL) != "" {
			src = publicURL
			fallback = publicURL
		}
	}
	return template.HTML(`<img class="news-cover-image" src="` + template.HTMLEscapeString(src) + `" alt="` + template.HTMLEscapeString(item.Title) + `" loading="lazy" data-fallback-src="` + template.HTMLEscapeString(fallback) + `">`)
}

func newsMetaBadgesMarkup(item news.Item) template.HTML {
	var b strings.Builder
	if strings.TrimSpace(item.Category) != "" {
		b.WriteString(`<span class="meta-pill-category">` + template.HTMLEscapeString(item.Category) + `</span>`)
	}
	if strings.TrimSpace(item.AuthorName) != "" {
		b.WriteString(`<span class="meta-pill-text">✍ ` + template.HTMLEscapeString(item.AuthorName) + `</span>`)
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
			itemsHTML.WriteString(`<li><span class="takeaway-bullet">✓</span><span class="takeaway-text">` + template.HTMLEscapeString(trimmed) + `</span></li>`)
		}
	}
	if itemsHTML.Len() == 0 {
		return ""
	}
	return template.HTML(`<div class="news-key-takeaways"><div class="takeaways-header"><span class="takeaways-icon">📌</span><h3>Önemli Çıkarımlar & Özet</h3></div><ul class="takeaways-list">` + itemsHTML.String() + `</ul></div>`)
}

func renderParagraphs(body string) string {
	return ui.RichTextHTML(body)
}

func (h Handler) adminPage(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
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
	var leftFields, rightMetaFields, rightSEOFields []adminstudio.Field
	for _, f := range h.AdminCollection.Fields {
		switch f.ID {
		case "title", "slug", "cover_media_id", "excerpt", "key_takeaways", "body":
			leftFields = append(leftFields, f)
		case "featured", "category", "author_name", "tags", "expires_at":
			rightMetaFields = append(rightMetaFields, f)
		case "seo_title", "seo_description":
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
	nonce := security.NonceFromContext(r.Context())

	var leftBuf bytes.Buffer
	leftCol := adminstudio.Collection{ID: "news_left", Label: "Haber İçeriği", Fields: leftFields, Actions: []adminstudio.Action{{Label: "Kaydet", Href: "/admin/news/" + item.ID}}}
	if err := adminstudio.RenderCreateForm(&leftBuf, adminstudio.FormData{Collection: leftCol, Action: "/admin/news/" + item.ID, CSRFToken: csrf, Nonce: nonce, Values: values, HideSubmit: true}); err != nil {
		http.Error(w, "news admin unavailable", http.StatusServiceUnavailable)
		return
	}

	var rightMetaBuf bytes.Buffer
	rightMetaCol := adminstudio.Collection{ID: "news_meta", Label: "Genel Bilgiler", Fields: rightMetaFields, Actions: []adminstudio.Action{{Label: "Kaydet", Href: "/admin/news/" + item.ID}}}
	if err := adminstudio.RenderCreateForm(&rightMetaBuf, adminstudio.FormData{Collection: rightMetaCol, Action: "/admin/news/" + item.ID, CSRFToken: csrf, Nonce: nonce, Values: values, HideSubmit: true}); err != nil {
		http.Error(w, "news admin unavailable", http.StatusServiceUnavailable)
		return
	}

	var rightSEOBuf bytes.Buffer
	rightSEOCol := adminstudio.Collection{ID: "news_seo", Label: "SEO & Arama Motoru", Fields: rightSEOFields, Actions: []adminstudio.Action{{Label: "Kaydet", Href: "/admin/news/" + item.ID}}}
	if err := adminstudio.RenderCreateForm(&rightSEOBuf, adminstudio.FormData{Collection: rightSEOCol, Action: "/admin/news/" + item.ID, CSRFToken: csrf, Nonce: nonce, Values: values, HideSubmit: true}); err != nil {
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
	if err := ui.AdminStartForRequest(w, r, "Haber düzenle", "Yayın merkezi", security.NonceFromContext(r.Context()), "news"); err != nil {
		return
	}
	if h.Presence != nil {
		endpoint := "/admin/news/" + url.PathEscape(item.ID) + "/presence"
		_ = designsystem.EditPresence(item.ID, endpoint, security.CSRFTokenFromContext(r.Context()), security.NonceFromContext(r.Context())).Render(r.Context(), w)
	}
	statusBadge := fmt.Sprintf(`<span class="status">%s</span>`, template.HTMLEscapeString(string(item.Status)))
	titleDisplay := item.Title
	if strings.TrimSpace(titleDisplay) == "" {
		titleDisplay = "[Başlıksız]"
	}
	_, _ = fmt.Fprintf(w, `<div class="editor-top-nav"><div class="editor-top-info"><strong class="editor-live-title">%s</strong><span class="editor-live-sub">Haber düzenleniyor · %s</span></div><div class="editor-top-actions"><button type="submit" form="news-edit-form" class="button">Taslağı kaydet</button></div></div><form id="news-edit-form" class="studio-form" method="post" action="/admin/news/%s" data-local-autosave data-autosave-resource="news:%s"><div class="editor-split-container"><div class="editor-main-col"><section class="editor-card"><div class="editor-card-title"><span>Haber İçeriği</span></div>%s</section></div><div class="editor-side-col"><section class="editor-card"><div class="editor-card-title"><span>Yayın &amp; İşlem</span>%s</div><div class="editor-side-fields"><div class="editor-side-status"><span>Durum</span><strong>%s</strong></div><div class="editor-side-status"><span>Dil</span><strong>%s</strong></div><div class="editor-workflow-actions">%s</div></div></section><section class="editor-card"><div class="editor-card-title"><span>Genel Bilgiler</span></div><div class="editor-side-fields">%s</div></section><section class="editor-card"><div class="editor-card-title"><span>İçerik Sınıflandırması</span></div><div class="editor-side-fields">%s</div></section><section class="editor-card"><div class="editor-card-title"><span>SEO &amp; Arama Motoru</span></div><div class="editor-side-fields">%s</div></section></div></div><input type="hidden" name="locale" value="%s"><input type="hidden" name="version" value="%d"><input type="hidden" name="_csrf" value="%s"></form><script nonce="%s">(()=>{const titleInp=document.querySelector('#news-edit-form input[name="title"]'),liveTitle=document.querySelector('.editor-live-title');if(titleInp&&liveTitle){titleInp.addEventListener('input',()=>{liveTitle.textContent=titleInp.value.trim()||'[Başlıksız]';});}})();</script>`, template.HTMLEscapeString(titleDisplay), statusBadge, template.HTMLEscapeString(item.ID), template.HTMLEscapeString(item.ID), template.HTML(leftBuf.String()), statusBadge, template.HTMLEscapeString(string(item.Status)), template.HTMLEscapeString(item.Locale), workflowActionMarkup(h.Service, item, csrf), template.HTML(rightMetaBuf.String()), template.HTML(taxonomyBuf.String()), template.HTML(rightSEOBuf.String()), template.HTMLEscapeString(item.Locale), item.Version, template.HTMLEscapeString(csrf), template.HTMLEscapeString(nonce))
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

func workflowActionMarkup(service news.Service, item news.Item, csrf string) template.HTML {
	var out strings.Builder
	id := template.HTMLEscapeString(item.ID)
	token := template.HTMLEscapeString(csrf)
	if service.CanTransition(item.Status, v1.EditorialScheduled) {
		fmt.Fprintf(&out, `<form method="post" action="/admin/news/%s/schedule"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="version" value="%d"><input name="scheduled_at" type="datetime-local"><button type="submit" class="button">Zamanla</button></form>`, id, token, item.Version)
	}
	if service.CanTransition(item.Status, v1.EditorialPublished) {
		fmt.Fprintf(&out, `<form method="post" action="/admin/news/%s/publish"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="version" value="%d"><button type="submit" class="button button-success">Yayına Al</button></form>`, id, token, item.Version)
	}
	if service.CanTransition(item.Status, v1.EditorialRetracted) {
		fmt.Fprintf(&out, `<form method="post" action="/admin/news/%s/retract"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="version" value="%d"><button type="submit" class="button button-quiet button-danger">Yayından Çek</button></form>`, id, token, item.Version)
	}
	if out.Len() == 0 {
		out.WriteString(`<p class="muted">Bu durum için kullanılabilir yayın aksiyonu yok.</p>`)
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
		writeError(w, err)
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
				writeError(w, err)
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
		writeError(w, err)
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
	http.Redirect(w, r, "/admin/news/"+url.PathEscape(r.PathValue("id")), http.StatusSeeOther)
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
	writeJSON(w, http.StatusOK, item)
}

func editorAllowed(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return auth.Can(user, auth.PermissionContentWrite)
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
	writeJSON(w, http.StatusOK, h.Presence.Active(tenantID, "news:"+r.PathValue("id"), user.ID))
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
