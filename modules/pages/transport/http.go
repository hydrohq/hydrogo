package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	themev1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	adminstudiosql "github.com/ardaozyildirim/hydrogo/internal/adminstudio/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/audit"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	uicomponents "github.com/ardaozyildirim/hydrogo/internal/ui/components"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	blockrender "github.com/ardaozyildirim/hydrogo/modules/pages/render"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
	"github.com/ardaozyildirim/hydrogo/modules/seo"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type Handler struct {
	DB      database.Executor
	Content service.ContentService
	Theme   *themev1.ThemeBundle
	Media   interface {
		Get(context.Context, database.Executor, string) (media.Asset, error)
	}
	AdminCollection     adminstudio.Collection
	LayoutTemplates     []pages.LayoutTemplate
	DefaultLocale       string
	SupportedLocales    []string
	AllowDemoFrontend   bool
	SEOOrigin           string
	PublicSEOSettings   func(context.Context) PublicSEOSettings
	NavigationProvider  func(context.Context, string) ([]site.NavigationItem, error)
	OGRenderer          *seo.OGRenderer
	DiscoveryProvider   seo.DiscoveryProvider
	DiscoveryCache      *seo.DiscoveryCache
	Notifications       *adminstudiosql.Store
	Presence            *realtime.Presence
	WeeklyDigestEnabled bool
	PreviewSigner       *preview.Signer
	PreviewOrigin       string
	RequireTenantScope  bool
	SearchEnabled       bool
	ContentViews        interface {
		ContentViewCounts(context.Context, string, string, []string) (map[string]int64, error)
	}
	Feedback interface {
		FeedbackTotals(context.Context, string, string, string) (analytics.FeedbackTotals, error)
		FeedbackCounts(context.Context, string, string, []string) (map[string]analytics.FeedbackTotals, error)
	}
	AnalyticsEnabled      bool
	RecordSlugRedirect    func(context.Context, string, string, string) error
	RecordNotFound        func(context.Context, string) error
	PublicSupport         adminstudio.SupportConfig
	PublicContactSettings func(context.Context) ([]site.Setting, error)
	Assistant             ai.Assistant
	AIAudit               audit.Writer
}

type PublicSEOSettings struct {
	Brand, FontCSS, AppearanceCSS, FaviconURL, SocialImageURL string
	BusinessProfile                                           *site.BusinessProfile
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

func (h Handler) feedbackRenderOptions(r *http.Request) blockrender.RenderOptions {
	options := blockrender.RenderOptions{CSRF: security.CSRFTokenFromContext(r.Context())}
	if !h.AnalyticsEnabled || h.Feedback == nil {
		return options
	}
	options.FeedbackEnabled = true
	options.FeedbackCounts = func(contentIDs []string) map[string][2]int64 {
		tenantID := "default"
		if scope, err := tenant.FromContext(r.Context()); err == nil {
			tenantID = scope.TenantID
		} else if h.RequireTenantScope {
			return map[string][2]int64{}
		}
		totals, err := h.Feedback.FeedbackCounts(r.Context(), tenantID, "faq", contentIDs)
		if err != nil {
			return map[string][2]int64{}
		}
		counts := make(map[string][2]int64, len(totals))
		for id, value := range totals {
			counts[id] = [2]int64{value.Up, value.Down}
		}
		return counts
	}
	return options
}

type pageInput struct {
	Locale string        `json:"locale"`
	Slug   string        `json:"slug"`
	Title  string        `json:"title"`
	Body   string        `json:"body"`
	Blocks []pages.Block `json:"blocks"`
	Tags   []string      `json:"tags,omitempty"`
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
	routes := []Route{
		{Pattern: "POST /api/pages", Handler: http.HandlerFunc(h.create), Protected: true},
		{Pattern: "GET /api/pages", Handler: http.HandlerFunc(h.listPublished)},
		{Pattern: "GET /api/pages/{id}", Handler: http.HandlerFunc(h.get), Protected: true},
		{Pattern: "GET /api/pages/{id}/preview", Handler: http.HandlerFunc(h.preview), Protected: true},
		{Pattern: "PUT /api/pages/{id}", Handler: http.HandlerFunc(h.update), Protected: true},
		{Pattern: "POST /api/pages/{id}/publish", Handler: http.HandlerFunc(h.publish), Protected: true},
		{Pattern: "GET /public/pages/{locale}/{slug}", Handler: http.HandlerFunc(h.getPublished)},
		{Pattern: "GET /api/pages/by-slug", Handler: http.HandlerFunc(h.getPublishedByQuery)},
		{Pattern: "GET /public/pages/{locale}/{slug}/html", Handler: http.HandlerFunc(h.getPublishedHTML)},
		{Pattern: "GET /og/{locale}/{slug}", Handler: http.HandlerFunc(h.openGraphImage)},
		{Pattern: "GET /admin/pages/new", Handler: http.HandlerFunc(h.adminNewPage), Protected: true},
		{Pattern: "GET /admin/pages/import", Handler: http.HandlerFunc(h.adminImport), Protected: true},
		{Pattern: "POST /admin/pages/import", Handler: http.HandlerFunc(h.adminImport), Protected: true},
		{Pattern: "POST /admin/pages/bulk", Handler: http.HandlerFunc(h.adminBulk), Protected: true},
		{Pattern: "GET /admin/notifications", Handler: http.HandlerFunc(h.adminNotifications), Protected: true},
		{Pattern: "POST /admin/notifications", Handler: http.HandlerFunc(h.adminNotifications), Protected: true},
		{Pattern: "GET /admin/notifications/unread", Handler: http.HandlerFunc(h.adminUnreadNotifications), Protected: true},
		{Pattern: "GET /admin/guide", Handler: http.HandlerFunc(h.adminGuide), Protected: true},
		{Pattern: "GET /admin/pages/{id}", Handler: http.HandlerFunc(h.adminPage), Protected: true},
		{Pattern: "GET /admin/pages", Handler: http.HandlerFunc(h.adminList), Protected: true},
		{Pattern: "POST /admin/pages", Handler: http.HandlerFunc(h.adminCreateForm), Protected: true},
		{Pattern: "POST /admin/pages/{id}", Handler: http.HandlerFunc(h.adminUpdate), Protected: true},
		{Pattern: "POST /admin/pages/{id}/publish", Handler: http.HandlerFunc(h.adminPublish), Protected: true},
		{Pattern: "POST /admin/pages/{id}/translate", Handler: http.HandlerFunc(h.adminTranslate), Protected: true},
		{Pattern: "POST /admin/pages/{id}/revisions/{revision}/restore", Handler: http.HandlerFunc(h.restoreRevision), Protected: true},
	}
	if h.Presence != nil {
		routes = append(routes,
			Route{Pattern: "POST /admin/pages/{id}/presence", Handler: http.HandlerFunc(h.editPresenceHeartbeat), Protected: true},
			Route{Pattern: "GET /admin/pages/{id}/presence", Handler: http.HandlerFunc(h.editPresenceList), Protected: true},
		)
	}
	if h.PreviewSigner != nil {
		routes = append(routes, Route{Pattern: "POST /admin/pages/{id}/preview-link", Handler: http.HandlerFunc(h.createPreviewLink), Protected: true}, Route{Pattern: "GET /preview/pages/{id}", Handler: http.HandlerFunc(h.publicDraftPreview)})
	}
	for _, route := range seo.DiscoveryRoutes(h.DiscoveryProvider, h.DiscoveryCache) {
		routes = append(routes, Route{Pattern: route.Pattern, Handler: route.Handler})
	}
	if h.assistantEnabled() {
		routes = append(routes,
			Route{Pattern: "POST /admin/pages/{id}/ai-suggest", Handler: http.HandlerFunc(h.adminAISuggest), Protected: true},
			Route{Pattern: "POST /admin/pages/{id}/ai-applied", Handler: http.HandlerFunc(h.adminAIApplied), Protected: true},
		)
	}
	return routes
}

func (h Handler) assistantEnabled() bool {
	return h.Assistant.Enabled && h.Assistant.Provider != nil && h.AIAudit != nil
}

func (h Handler) adminBulk(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "Bu işlem için yetkiniz yok.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil || (r.FormValue("action") != "export" && r.FormValue("action") != "CSV dışa aktar") {
		http.Error(w, "Toplu işlem geçersiz.", http.StatusBadRequest)
		return
	}
	ids, err := adminstudio.NormalizeBulkIDs(r.Form["ids"], 500)
	if err != nil {
		http.Error(w, "En fazla 500 kayıt seçin.", http.StatusBadRequest)
		return
	}
	repo, ok := h.Content.Pages.(adminListRepository)
	if !ok {
		http.Error(w, "İşlem tamamlanamadı.", http.StatusServiceUnavailable)
		return
	}
	items, err := repo.ListAdmin(r.Context(), h.DB, 5000)
	if err != nil {
		http.Error(w, "İşlem tamamlanamadı.", http.StatusServiceUnavailable)
		return
	}
	selected := map[string]struct{}{}
	for _, id := range ids {
		selected[id] = struct{}{}
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="pages.csv"`)
	out := csv.NewWriter(w)
	_ = out.Write([]string{"locale", "slug", "title", "status"})
	for _, item := range items {
		if _, ok := selected[item.ID]; ok {
			_ = out.Write([]string{item.Locale, item.Slug, item.Title, string(item.Status)})
		}
	}
	out.Flush()
}

func (h Handler) adminNotifications(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || !editorAllowed(r) {
		http.Error(w, "Bu alan için yetkiniz yok.", http.StatusForbidden)
		return
	}
	if h.Notifications == nil {
		http.Error(w, "Bildirimler şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "İşlem tamamlanamadı.", http.StatusBadRequest)
			return
		}
		var err error
		switch r.FormValue("action") {
		case "read_all":
			err = h.Notifications.MarkAllRead(r.Context(), h.DB, user.ID)
		case "dismiss":
			err = h.Notifications.Dismiss(r.Context(), h.DB, user.ID, r.FormValue("id"))
		case "digest":
			if !h.WeeklyDigestEnabled {
				http.Error(w, "E-posta özeti şu anda kullanılamıyor.", http.StatusServiceUnavailable)
				return
			}
			err = h.Notifications.SetDigestPreferenceWithFrequency(r.Context(), h.DB, user.ID, r.FormValue("timezone"), r.FormValue("frequency"), r.FormValue("enabled") == "true")
		default:
			http.Error(w, "İşlem geçersiz.", http.StatusBadRequest)
			return
		}
		if err != nil {
			if errors.Is(err, adminstudiosql.ErrInvalid) {
				http.Error(w, "Saat dilimi tercihi geçersiz.", http.StatusBadRequest)
				return
			}
			http.Error(w, "İşlem tamamlanamadı.", http.StatusServiceUnavailable)
			return
		}
		http.Redirect(w, r, "/admin/notifications", http.StatusSeeOther)
		return
	}
	queryDB, ok := h.DB.(database.QueryExecutor)
	if !ok {
		http.Error(w, "Bildirimler yüklenemedi.", http.StatusServiceUnavailable)
		return
	}
	items, err := h.Notifications.List(r.Context(), queryDB, user.ID, 50)
	if err != nil {
		http.Error(w, "Bildirimler yüklenemedi.", http.StatusServiceUnavailable)
		return
	}
	view := make([]ui.NotificationItem, 0, len(items))
	for _, item := range items {
		view = append(view, ui.NotificationItem{ID: item.ID, Title: item.Title, Message: item.Message, TargetPath: item.TargetPath, Created: item.CreatedAt.Local().Format("02.01.2006 15:04"), Read: item.Read})
	}
	digestEnabled, timezone, frequency, err := h.Notifications.DigestPreferenceDetailsFor(r.Context(), h.DB, user.ID)
	if err != nil {
		http.Error(w, "Tercihler yüklenemedi.", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if ui.AdminStartForRequest(w, r, "Bildirimler", "Sistem bildirimleri", "dashboard") != nil {
		return
	}
	_ = ui.NotificationCenter(view, security.CSRFTokenFromContext(r.Context()), digestEnabled, timezone, frequency, h.WeeklyDigestEnabled).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) adminUnreadNotifications(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || !editorAllowed(r) {
		http.Error(w, "Bu alan için yetkiniz yok.", http.StatusForbidden)
		return
	}
	if h.Notifications == nil {
		http.Error(w, "Bildirimler şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	count, err := h.Notifications.UnreadCount(r.Context(), h.DB, user.ID)
	if err != nil {
		http.Error(w, "Bildirimler yüklenemedi.", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(struct {
		Unread int `json:"unread"`
	}{Unread: count})
}

func (h Handler) adminGuide(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "Bu alan için yetkiniz yok.", http.StatusForbidden)
		return
	}
	support := adminstudio.SupportConfigFromContext(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if ui.AdminStartForRequest(w, r, "Yardım ve rehberler", "", "dashboard") != nil {
		return
	}
	_ = ui.GuideCenter(support.GuideURL).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
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
		viewCounts, countsErr = h.ContentViews.ContentViewCounts(r.Context(), tenantID, "page", ids)
		viewsAvailable = countsErr == nil
	}
	if viewsAvailable {
		sortPagesByViews(items, viewCounts, r.URL.Query().Get("sort"))
	}
	rows := make([]adminstudio.Row, 0, len(items))
	for _, item := range items {
		title := item.Title
		if h.Media != nil {
			if content, contentErr := h.Content.Get(r.Context(), h.DB, item.ID); contentErr == nil {
				serialized, _ := json.Marshal(content.Blocks)
				references := media.MissingReferences(content.Body+string(serialized), nil)
				missing := 0
				for _, id := range references {
					if _, getErr := h.Media.Get(r.Context(), h.DB, id); getErr != nil {
						missing++
					}
				}
				if missing > 0 {
					title += fmt.Sprintf(" ⚠ %d eksik görsel", missing)
				}
			}
		}
		actions := []adminstudio.Action{{Label: "Düzenle", Href: "/admin/pages/" + item.ID}}
		if item.Status == pages.StatusPublished {
			publicPath := "/public/pages/" + url.PathEscape(item.Locale) + "/" + url.PathEscape(item.Slug) + "/html"
			actions = append(actions, adminstudio.Action{Label: "Canlı URL'yi kopyala", CopyText: strings.TrimRight(strings.TrimSpace(h.SEOOrigin), "/") + publicPath})
		}
		rows = append(rows, adminstudio.Row{ID: item.ID, Values: map[string]string{"locale": item.Locale, "slug": item.Slug, "title": title, "status": string(item.Status), "views": strconv.FormatInt(viewCounts[item.ID], 10)}, ValueHTML: map[string]template.HTML{"views": renderViewCountBadge(r.Context(), viewCounts[item.ID])}, Status: adminstudio.StatusInfo{Value: string(item.Status)}, Actions: actions})
	}
	collection := h.AdminCollection
	if viewsAvailable {
		collection.Fields = append(append([]adminstudio.Field(nil), collection.Fields...), adminstudio.Field{ID: "views", Label: "Views", Kind: adminstudio.FieldText, VisibleInList: true})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Sayfalar", "", "pages"); err != nil {
		return
	}
	if err := ui.ListViewSwitcher("admin-content-list", "hydro-view-content").Render(r.Context(), w); err != nil {
		return
	}
	_, _ = fmt.Fprint(w, `<section class="studio-card"><div class="studio-actions"><a class="button button-quiet" href="/admin/pages/import">CSV / Excel içe aktar</a></div><div class="table-wrap" id="admin-content-list" data-view-mode="table">`)
	if viewsAvailable {
		_ = designsystem.ContentViewSort(designsystem.ContentViewSortProps{Path: "/admin/pages", Status: r.URL.Query().Get("status"), Selected: r.URL.Query().Get("sort")}).Render(r.Context(), w)
	}
	if err := adminstudio.RenderList(w, adminstudio.ListData{Collection: collection, Rows: rows, StatusFilter: r.URL.Query().Get("status")}); err != nil {
		http.Error(w, "page admin unavailable", http.StatusInternalServerError)
		return
	}
	_, _ = fmt.Fprint(w, `</div></section>`)
	_ = ui.AdminEnd(w)
}

func sortPagesByViews(items []pages.Page, counts map[string]int64, direction string) {
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

func (h Handler) adminImport(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "Bu işlem için yetkiniz yok.", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Method == http.MethodGet {
		if ui.AdminStartForRequest(w, r, "İçerik içe aktar", "", "pages") != nil {
			return
		}
		_ = ui.ImportWizard(nil, security.CSRFTokenFromContext(r.Context()), false).Render(r.Context(), w)
		_ = ui.AdminEnd(w)
		return
	}
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		http.Error(w, "Dosya okunamadı. CSV veya Excel dosyasını yeniden seçin.", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Bir CSV veya Excel dosyası seçin.", http.StatusBadRequest)
		return
	}
	defer file.Close()
	rows, err := adminstudio.ParseImportFile(file, header.Filename)
	if err != nil {
		http.Error(w, "Dosya biçimi okunamadı. CSV veya .xlsx dosyası kullanın.", http.StatusBadRequest)
		return
	}
	headers := rows[0]
	if r.FormValue("map_title") == "" || r.FormValue("map_slug") == "" {
		if ui.AdminStartForRequest(w, r, "Sütunları eşleştir", "İçe aktarma sihirbazı", "pages") != nil {
			return
		}
		_ = ui.ImportWizard(headers, security.CSRFTokenFromContext(r.Context()), true).Render(r.Context(), w)
		_ = ui.AdminEnd(w)
		return
	}
	columns := map[string]int{}
	for _, field := range []string{"locale", "slug", "title", "body"} {
		raw := r.FormValue("map_" + field)
		if raw == "" {
			continue
		}
		index, parseErr := strconv.Atoi(raw)
		if parseErr != nil || index < 0 || index >= len(headers) {
			http.Error(w, "Sütun eşlemesi geçersiz. Lütfen yeniden deneyin.", http.StatusBadRequest)
			return
		}
		columns[field] = index
	}
	if _, ok := columns["body"]; !ok {
		http.Error(w, "İçerik metni için bir sütun seçin.", http.StatusBadRequest)
		return
	}
	localeIndex, localeMapped := columns["locale"]
	if columns["slug"] == columns["title"] || columns["slug"] == columns["body"] || columns["title"] == columns["body"] || (localeMapped && (localeIndex == columns["slug"] || localeIndex == columns["title"] || localeIndex == columns["body"])) {
		http.Error(w, "Başlık, bağlantı adı ve metin için farklı sütunlar seçin.", http.StatusBadRequest)
		return
	}
	mapValues := map[string]string{}
	for name, index := range columns {
		mapValues[name] = strconv.Itoa(index)
	}
	repo, ok := h.Content.Pages.(adminListRepository)
	if !ok {
		http.Error(w, "Aktarım şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	existing, err := repo.ListAdmin(r.Context(), h.DB, 5000)
	if err != nil {
		http.Error(w, "İçerikler yüklenemedi.", http.StatusServiceUnavailable)
		return
	}
	seen := map[string]bool{}
	for _, item := range existing {
		seen[canonical.Locale(item.Locale)+"\x00"+item.Slug] = true
	}
	valid, duplicates, invalid := planImport(rows[1:], columns, seen)
	if r.FormValue("action") != "import" {
		if ui.AdminStartForRequest(w, r, "Aktarım önizlemesi", "İçe aktarma önizlemesi", "pages") != nil {
			return
		}
		_ = ui.ImportPreview(security.CSRFTokenFromContext(r.Context()), mapValues, len(valid), duplicates, invalid).Render(r.Context(), w)
		_ = ui.AdminEnd(w)
		return
	}
	created, skipped := 0, duplicates+invalid
	for _, page := range valid {
		page.ID = newImportID()
		if page.ID == "" || h.Content.CreateDraft(r.Context(), page) != nil {
			skipped++
			continue
		}
		created++
	}
	if ui.AdminStartForRequest(w, r, "Aktarım tamamlandı", "İçe aktarma sonucu", "pages") != nil {
		return
	}
	_, _ = fmt.Fprintf(w, `<section class="studio-card" role="status"><h2>Aktarım tamamlandı</h2><p>%d taslak eklendi. %d satır atlandı.</p><a href="/admin/pages">Sayfalara dön</a></section>`, created, skipped)
	_ = ui.AdminEnd(w)
}

func planImport(rows [][]string, columns map[string]int, existing map[string]bool) ([]pages.Page, int, int) {
	valid := make([]pages.Page, 0, len(rows))
	duplicates, invalid := 0, 0
	seen := make(map[string]bool, len(existing))
	for key := range existing {
		seen[key] = true
	}
	for _, row := range rows {
		value := func(name, fallback string) string {
			if i, ok := columns[name]; ok && i >= 0 && i < len(row) {
				return strings.TrimSpace(row[i])
			}
			return fallback
		}
		locale := canonical.Locale(value("locale", "tr"))
		slug, err := canonical.Path(value("slug", ""))
		title, body := value("title", ""), value("body", "")
		if err != nil || locale == "" || len(locale) > 16 || slug == "" || len(slug) > 256 || title == "" || len(title) > 256 || body == "" || len(body) > 1<<20 {
			invalid++
			continue
		}
		key := locale + "\x00" + slug
		if seen[key] {
			duplicates++
			continue
		}
		seen[key] = true
		valid = append(valid, pages.Page{Locale: locale, Slug: slug, Title: title, Body: body, Status: pages.StatusDraft})
	}
	return valid, duplicates, invalid
}

func newImportID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	return "page-import-" + hex.EncodeToString(b[:])
}

func (h Handler) adminNewPage(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	layouts := append(pages.BuiltinLayoutTemplates(), h.LayoutTemplates...)
	if err := pages.ValidateLayoutTemplates(layouts); err != nil {
		http.Error(w, "page templates unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Yeni sayfa", "Sayfa yönetimi", "pages"); err != nil {
		return
	}
	_, _ = fmt.Fprint(w, `<section class="studio-card"><form class="studio-form" method="post" action="/admin/pages">`)
	options := make([]designsystem.PageLayoutOption, 0, len(layouts))
	locale := ui.AdminLocaleFromRequest(r)
	for _, layout := range layouts {
		name, description := pageLayoutCopy(layout, locale)
		options = append(options, designsystem.PageLayoutOption{ID: layout.ID, Name: name, Description: description})
	}
	legend, emptyMessage := "Sayfa düzeni", "Kullanılabilir sayfa düzeni yok."
	if locale == "en" {
		legend, emptyMessage = "Page layout", "No layout options available."
	}
	if err := designsystem.PageLayoutPicker(designsystem.PageLayoutPickerProps{Options: options, Legend: legend, EmptyMessage: emptyMessage}).Render(r.Context(), w); err != nil {
		return
	}
	if err := adminstudio.RenderCreateForm(w, adminstudio.FormData{
		Collection:  h.AdminCollection,
		Action:      "/admin/pages",
		CSRFToken:   security.CSRFTokenFromContext(r.Context()),
		Values:      map[string]string{"locale": "tr"},
		SubmitLabel: "Taslak oluştur",
	}); err != nil {
		return
	}
	if err := uicomponents.Input(uicomponents.InputProps{ID: "page-tags", Name: "tags", Type: "text", Label: "İçerik etiketleri (virgülle ayrılmış)"}).Render(r.Context(), w); err != nil {
		return
	}
	_, _ = fmt.Fprint(w, `</form></section>`)
	_ = ui.AdminEnd(w)
}

func pageLayoutCopy(layout pages.LayoutTemplate, locale string) (string, string) {
	if locale == "en" {
		return layout.Name, layout.Description
	}
	switch layout.ID {
	case "blank":
		return "Boş sayfa", "Sıfırdan başlayıp içeriğinizi oluşturun."
	case "service_showcase":
		return "Hizmet tanıtımı", "Hizmetinizi ve sunduğu faydaları tanıtın."
	case "landing":
		return "Açılış sayfası", "Ziyaretçileri net bir mesaj ve eylemle karşılayın."
	case "case_study":
		return "Başarı hikâyesi", "İhtiyacı, sonucu ve müşteri deneyimini anlatın."
	default:
		return layout.Name, layout.Description
	}
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
	templateID := r.FormValue("layout_template")
	if templateID == "" {
		templateID = "blank"
	}
	blocks, err := pages.ResolveLayoutTemplate(templateID, h.LayoutTemplates)
	if err != nil {
		http.Error(w, "invalid page layout", http.StatusBadRequest)
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
		Blocks: blocks,
		Tags:   strings.Split(r.FormValue("tags"), ","),
		Status: pages.StatusDraft,
	}
	if err := h.validatePageMedia(r.Context(), page); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
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
	api.WriteJSON(w, http.StatusOK, summarizePages(visible))
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
		if pageStatus(err) == http.StatusNotFound {
			h.recordNotFound(r)
		}
		http.Error(w, "page lookup failed", pageReadStatus(err))
		return
	}
	if !h.publicPageAllowed(page) {
		http.NotFound(w, r)
		return
	}
	api.WriteJSON(w, http.StatusOK, page)
}

func (h Handler) getPublishedHTML(w http.ResponseWriter, r *http.Request) {
	page, err := h.Content.GetPublished(r.Context(), h.DB, r.PathValue("locale"), r.PathValue("slug"))
	if err != nil {
		if pageStatus(err) == http.StatusNotFound {
			h.recordNotFound(r)
		}
		http.Error(w, "page lookup failed", pageReadStatus(err))
		return
	}
	if !h.publicPageAllowed(page) {
		http.NotFound(w, r)
		return
	}
	renderBlocks := make([]pages.Block, len(page.Blocks))
	for i, block := range page.Blocks {
		renderBlocks[i], err = h.withPublicMedia(r.Context(), block)
		if err != nil {
			http.Error(w, "page media lookup failed", http.StatusInternalServerError)
			return
		}
	}
	blocks, err := blockrender.BlocksWithOptions(renderBlocks, h.feedbackRenderOptions(r))
	if err != nil {
		http.Error(w, "page render failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	seoMeta, err := h.pageSEOMeta(r.Context(), page)
	if err != nil {
		http.Error(w, "page render failed", http.StatusInternalServerError)
		return
	}
	designHead := ""
	showConsentNotice := h.AnalyticsEnabled && !hasAnalyticsConsent(r)
	if h.AnalyticsEnabled {
		w.Header().Set("Vary", "Cookie")
	}
	if pageHasDesignSystemBlocks(page.Blocks) || showConsentNotice || h.PublicSupport.Enabled || h.PublicContactSettings != nil {
		var head bytes.Buffer
		if err := ui.DesignSystemHead().Render(r.Context(), &head); err != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
			return
		}
		designHead = head.String()
	}
	supportMarkup := h.socialLinksMarkup(r.Context()) + h.floatingContact(r.Context(), page.Locale)
	searchMarkup := ""
	if h.SearchEnabled {
		var search bytes.Buffer
		if err := ui.PublicSearchBox().Render(r.Context(), &search); err != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
			return
		}
		searchMarkup = search.String()
	}
	designHead += seoMeta
	articleHTML, tocHTML, err := ui.PrepareArticleHTML(ui.RichTextHTML(page.Body))
	if err != nil {
		http.Error(w, "page render failed", http.StatusInternalServerError)
		return
	}
	designHead += `<link rel="stylesheet" href="/designsystem/static/article-reading.css"><script src="/designsystem/static/article-reading.js" defer></script>`
	var readingTools bytes.Buffer
	if err := ui.PublicArticleTools().Render(r.Context(), &readingTools); err != nil {
		http.Error(w, "page render failed", http.StatusInternalServerError)
		return
	}
	if editorAllowed(r) {
		designHead += `<style>.hydro-inline-edit{position:fixed;right:18px;bottom:18px;z-index:9999;display:inline-flex;gap:7px;align-items:center;padding:10px 14px;border-radius:999px;background:#0284c7;color:#fff;text-decoration:none;font:600 14px/1.2 system-ui;box-shadow:0 4px 16px #0003}.hydro-inline-edit:hover{background:#0369a1}</style>`
	}
	consentMarkup := ""
	if showConsentNotice {
		var notice bytes.Buffer
		props := designsystem.CookieNoticeProps{Action: "/analytics/consent", CSRF: security.CSRFTokenFromContext(r.Context()), ReturnTo: r.URL.Path, Variant: "bar"}
		if strings.HasPrefix(canonical.Locale(page.Locale), "en") {
			props.Heading, props.Message, props.AcceptLabel, props.RejectLabel = "Privacy choices", "Allow anonymous, daily page-view totals? HydroGo does not use advertising cookies.", "Allow analytics", "Decline"
		} else {
			props.Heading, props.Message, props.AcceptLabel, props.RejectLabel = "Gizlilik tercihiniz", "Anonim günlük sayfa görüntülenme toplamlarına izin verilsin mi? HydroGo reklam çerezi kullanmaz.", "Analitiğe izin ver", "Reddet"
		}
		if err := designsystem.CookieNotice(props).Render(r.Context(), &notice); err != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
			return
		}
		consentMarkup = notice.String()
	}
	for i, block := range renderBlocks {
		if block.Type != "faq" {
			continue
		}
		var payload struct {
			Items []ui.FAQItem `json:"items"`
		}
		if json.Unmarshal(block.Payload, &payload) != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
			return
		}
		var schema bytes.Buffer
		if err := ui.FAQSchema(fmt.Sprintf("page-faq-%d", i), payload.Items).Render(r.Context(), &schema); err != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
			return
		}
		designHead += schema.String()
	}
	relatedMarkup := template.HTML("")
	if related, relatedErr := h.Content.Related(r.Context(), h.DB, page, 4); relatedErr == nil {
		var relatedHTML bytes.Buffer
		if renderErr := pages.RelatedPages(page.ID, page.Locale, related).Render(r.Context(), &relatedHTML); renderErr == nil {
			relatedMarkup = template.HTML(relatedHTML.String())
		}
	}
	if h.Theme != nil {
		var content strings.Builder
		content.WriteString(searchMarkup)
		fmt.Fprintf(&content, `<div data-testid="public-page"><h1>%s</h1>%s%s<div class="content-body article-reading-body">%s</div></div>`, template.HTMLEscapeString(page.Title), readingTools.String(), template.HTML(tocHTML), template.HTML(articleHTML))
		if editorAllowed(r) {
			content.WriteString(renderInPlaceEditLink(r, page.ID))
		}
		for _, block := range renderBlocks {
			var themed strings.Builder
			if err := h.Theme.RenderBlock(&themed, block); err == nil {
				styled, styleErr := blockrender.StyleBlockHTML(block, themed.String())
				if styleErr != nil {
					http.Error(w, "page render failed", http.StatusInternalServerError)
					return
				}
				content.WriteString(styled)
			} else if errors.Is(err, themev1.ErrThemeBlockUnavailable) {
				fallback, fallbackErr := blockrender.BlocksWithOptions([]pages.Block{block}, h.feedbackRenderOptions(r))
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
		content.WriteString(string(relatedMarkup))
		content.WriteString(consentMarkup)
		content.WriteString(supportMarkup)
		navigation := make([]themev1.ThemeNavigationItem, 0)
		if h.NavigationProvider != nil {
			items, navErr := h.NavigationProvider(r.Context(), page.Locale)
			if navErr != nil {
				http.Error(w, "page render failed", http.StatusInternalServerError)
				return
			}
			for _, item := range items {
				navigation = append(navigation, themev1.ThemeNavigationItem{ID: item.ID, ParentID: item.ParentID, Label: item.Label, Href: item.Href, Target: item.Target, Position: item.Position})
			}
		}
		navigationHTML, navErr := themev1.RenderThemeNavigation(navigation)
		if navErr != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
			return
		}
		navigationHTML = template.HTML(string(navigationHTML) + searchMarkup)
		if err := h.Theme.RenderPage(w, themev1.PageTemplateData{Title: page.Title, Slug: page.Slug, Locale: page.Locale, Direction: canonical.Direction(page.Locale), Content: template.HTML(content.String()), Navigation: navigation, Slots: map[string]template.HTML{"content": template.HTML(content.String()), "head": template.HTML(designHead), "navigation": navigationHTML}}); err != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
		}
		return
	}
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="%s" dir="%s"><head><meta charset="utf-8"><title>%s</title>%s</head><body>%s<main data-testid="public-page"><h1>%s</h1>%s%s<div class="content-body article-reading-body">%s</div>%s%s</main>%s%s</body></html>`, template.HTMLEscapeString(page.Locale), canonical.Direction(page.Locale), template.HTMLEscapeString(page.Title), designHead, template.HTML(searchMarkup), template.HTMLEscapeString(page.Title), renderInPlaceEditLink(r, page.ID), readingTools.String(), template.HTML(articleHTML), blocks, relatedMarkup, template.HTML(consentMarkup), template.HTML(supportMarkup))
}

func (h Handler) recordNotFound(r *http.Request) {
	if h.RecordNotFound != nil && len(r.URL.Path) <= 2048 {
		_ = h.RecordNotFound(r.Context(), r.URL.Path)
	}
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

func hasAnalyticsConsent(r *http.Request) bool {
	cookie, err := r.Cookie(analytics.ConsentCookie)
	return err == nil && strings.TrimSpace(cookie.Value) != ""
}

func renderInPlaceEditLink(r *http.Request, pageID string) string {
	if !editorAllowed(r) {
		return ""
	}
	var link bytes.Buffer
	if err := ui.InPlaceEditLink("/admin/pages/"+url.PathEscape(pageID)).Render(r.Context(), &link); err != nil {
		return ""
	}
	return link.String()
}

func (h Handler) pageSEOMeta(ctx context.Context, page pages.Page) (string, error) {
	settings := PublicSEOSettings{}
	if h.PublicSEOSettings != nil {
		settings = h.PublicSEOSettings(ctx)
	}
	var head strings.Builder
	if settings.AppearanceCSS != "" {
		fmt.Fprintf(&head, `<style>%s</style>`, settings.AppearanceCSS)
	}
	if validLocalFontCSS(settings.FontCSS) {
		fmt.Fprintf(&head, `<link rel="stylesheet" href="%s">`, template.HTMLEscapeString(settings.FontCSS))
	}
	if site.ValidAppearanceAssetURL(settings.FaviconURL) && settings.FaviconURL != "" {
		fmt.Fprintf(&head, `<link rel="icon" href="%s">`, template.HTMLEscapeString(settings.FaviconURL))
	}
	if settings.BusinessProfile != nil {
		if jsonLD, err := site.BusinessJSONLD(*settings.BusinessProfile); err == nil {
			fmt.Fprintf(&head, `<script type="application/ld+json">%s</script>`, jsonLD)
		}
	}
	origin := strings.TrimSpace(h.SEOOrigin)
	if origin == "" {
		return head.String(), nil
	}
	ogInput := h.pageOGInput(page, settings)
	imagePath := "/og/" + url.PathEscape(page.Locale) + "/" + url.PathEscape(page.Slug) + ".png?v=" + seo.OGVersion(ogInput)
	view, err := site.BuildSEO(site.SEOInput{Origin: origin, Locale: page.Locale, Path: "/" + page.Slug, Title: page.Title, OGImage: strings.TrimRight(origin, "/") + imagePath})
	if err != nil {
		return "", err
	}
	if translations, ok := h.Content.Pages.(interface {
		ListPublishedTranslations(context.Context, database.Executor, string, int) ([]pages.Page, error)
	}); ok {
		variants, listErr := translations.ListPublishedTranslations(ctx, h.DB, page.Slug, 100)
		if listErr != nil {
			return "", listErr
		}
		for _, variant := range variants {
			locale := canonical.Locale(variant.Locale)
			if locale == "" || locale != variant.Locale || variant.Slug != page.Slug {
				continue
			}
			href := strings.TrimRight(origin, "/") + "/public/pages/" + url.PathEscape(locale) + "/" + url.PathEscape(variant.Slug) + "/html"
			fmt.Fprintf(&head, `<link rel="alternate" hreflang="%s" href="%s">`, template.HTMLEscapeString(locale), template.HTMLEscapeString(href))
		}
	}
	ogImage := view.OGImage
	if site.ValidAppearanceAssetURL(settings.SocialImageURL) && settings.SocialImageURL != "" {
		ogImage = strings.TrimRight(origin, "/") + settings.SocialImageURL
	}
	fmt.Fprintf(&head, `<link rel="canonical" href="%s"><meta name="description" content="%s"><meta property="og:type" content="website"><meta property="og:title" content="%s"><meta property="og:description" content="%s"><meta property="og:url" content="%s"><meta property="og:image" content="%s"><meta name="twitter:card" content="summary_large_image"><meta name="twitter:image" content="%s">`, template.HTMLEscapeString(view.CanonicalURL), template.HTMLEscapeString(view.Description), template.HTMLEscapeString(view.OGTitle), template.HTMLEscapeString(view.OGDescription), template.HTMLEscapeString(view.CanonicalURL), template.HTMLEscapeString(ogImage), template.HTMLEscapeString(ogImage))
	return head.String(), nil
}

func (h Handler) publicSEOSettings(ctx context.Context) PublicSEOSettings {
	if h.PublicSEOSettings == nil {
		return PublicSEOSettings{}
	}
	return h.PublicSEOSettings(ctx)
}

func (h Handler) pageOGInput(page pages.Page, settings PublicSEOSettings) seo.OGInput {
	input := seo.OGInput{Title: page.Title, Category: "Page"}
	input.Brand = settings.Brand
	return input
}

func validLocalFontCSS(value string) bool {
	value = strings.TrimSpace(value)
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != value || !strings.HasPrefix(value, "/fonts/font-") || !strings.HasSuffix(value, ".css") || strings.ContainsAny(value, "\\\r\n\x00") {
		return false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(value, "/fonts/font-"), ".css")
	if name == "" {
		return false
	}
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_') {
			return false
		}
	}
	return true
}

func (h Handler) openGraphImage(w http.ResponseWriter, r *http.Request) {
	if h.OGRenderer == nil {
		http.NotFound(w, r)
		return
	}
	// Clients request /og/{locale}/{slug}.png; ServeMux wildcards must own a
	// whole segment, so the .png suffix is parsed here instead of the pattern.
	slug := strings.TrimSuffix(r.PathValue("slug"), ".png")
	page, err := h.Content.GetPublished(r.Context(), h.DB, r.PathValue("locale"), slug)
	if err != nil || !h.publicPageAllowed(page) {
		http.NotFound(w, r)
		return
	}
	data, err := h.OGRenderer.Render(h.pageOGInput(page, h.publicSEOSettings(r.Context())))
	if err != nil {
		http.Error(w, "image unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(data)
}

func pageHasDesignSystemBlocks(blocks []pages.Block) bool {
	return len(blocks) > 0
}

func (h Handler) withPublicMedia(ctx context.Context, block pages.Block) (pages.Block, error) {
	if h.Media == nil || block.Type != "hero" && block.Type != "before_after" {
		return block, nil
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(block.Payload, &payload); err != nil {
		return block, err
	}
	keys := map[string]string{"media_asset_id": "media_url"}
	if block.Type == "before_after" {
		keys = map[string]string{"before_image_asset_id": "before_image_url", "after_image_asset_id": "after_image_url"}
	}
	changed := false
	for idField, urlField := range keys {
		var assetID string
		if err := json.Unmarshal(payload[idField], &assetID); err != nil || assetID == "" {
			continue
		}
		asset, err := h.Media.Get(ctx, h.DB, assetID)
		if err != nil || asset.Status != "ready" || asset.Visibility != media.VisibilityPublic {
			continue
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
		payload[urlField] = encodedURL
		changed = true
	}
	if !changed {
		return block, nil
	}
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
	api.WriteJSON(w, http.StatusOK, page)
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
	page := pages.Page{ID: id, Locale: input.Locale, Slug: input.Slug, Title: input.Title, Body: input.Body, Blocks: input.Blocks, Tags: input.Tags}
	if err := h.validatePageMedia(r.Context(), page); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
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

func (h Handler) createPreviewLink(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) || h.PreviewSigner == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	page, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil || page.Status != pages.StatusDraft {
		http.Error(w, "draft unavailable", http.StatusNotFound)
		return
	}
	tenantID := ""
	if h.RequireTenantScope {
		scope, e := tenant.Require(r.Context())
		if e != nil {
			http.Error(w, "preview unavailable", http.StatusForbidden)
			return
		}
		tenantID = scope.TenantID
	}
	now := time.Now().UTC()
	token, err := h.PreviewSigner.Issue(preview.Claims{PageID: page.ID, TenantID: tenantID, Revision: page.Revision}, now, 24*time.Hour)
	if err != nil {
		http.Error(w, "preview unavailable", http.StatusServiceUnavailable)
		return
	}
	target := "/preview/pages/" + url.PathEscape(page.ID) + "?token=" + url.QueryEscape(token)
	if h.PreviewOrigin != "" {
		target = strings.TrimRight(h.PreviewOrigin, "/") + target
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = ui.DraftPreviewLinkResult(target, now.Add(24*time.Hour).Format(time.RFC3339)).Render(r.Context(), w)
		return
	}
	api.WriteJSON(w, http.StatusCreated, map[string]any{"url": target, "expires_at": now.Add(24 * time.Hour)})
}

func (h Handler) publicDraftPreview(w http.ResponseWriter, r *http.Request) {
	if h.PreviewSigner == nil {
		http.NotFound(w, r)
		return
	}
	claims, err := h.PreviewSigner.Verify(r.URL.Query().Get("token"), time.Now().UTC())
	if err != nil || claims.PageID != r.PathValue("id") {
		http.NotFound(w, r)
		return
	}
	if h.RequireTenantScope {
		scope, e := tenant.Require(r.Context())
		if e != nil || scope.TenantID != claims.TenantID {
			http.NotFound(w, r)
			return
		}
	}
	page, err := h.Content.Get(r.Context(), h.DB, claims.PageID)
	if err != nil || page.Status != pages.StatusDraft || page.Revision != claims.Revision {
		http.NotFound(w, r)
		return
	}
	blocks, err := blockrender.Blocks(page.Blocks)
	if err != nil {
		http.Error(w, "draft preview unavailable", http.StatusInternalServerError)
		return
	}
	article, _, err := ui.PrepareArticleHTML(ui.RichTextHTML(page.Body))
	if err != nil {
		http.Error(w, "draft preview unavailable", http.StatusInternalServerError)
		return
	}
	body := `<main><h1>` + template.HTMLEscapeString(page.Title) + `</h1><div class="content-body article-reading-body">` + article + `</div>` + blocks + `</main>`
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if h.Theme != nil {
		if err := h.Theme.RenderPage(w, themev1.PageTemplateData{Title: page.Title, Slug: page.Slug, Locale: page.Locale, Direction: canonical.Direction(page.Locale), Content: template.HTML(body), Slots: map[string]template.HTML{"content": template.HTML(body)}}); err != nil {
			return
		}
		return
	}
	_, _ = fmt.Fprintf(w, "<!doctype html><html lang=\"%s\"><head><meta charset=\"utf-8\"><meta name=\"robots\" content=\"noindex,nofollow\"><title>%s</title></head><body>%s</body></html>", template.HTMLEscapeString(page.Locale), template.HTMLEscapeString(page.Title), body)
}

func (h Handler) writePage(w http.ResponseWriter, r *http.Request, id string) {
	page, err := h.Content.Get(r.Context(), h.DB, id)
	if err != nil {
		http.Error(w, "page lookup failed", pageReadStatus(err))
		return
	}
	api.WriteJSON(w, http.StatusOK, page)
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
	page := pages.Page{ID: r.PathValue("id"), Locale: input.Locale, Slug: input.Slug, Title: input.Title, Body: input.Body, Blocks: input.Blocks, Tags: input.Tags}
	if err := h.validatePageMedia(r.Context(), page); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Content.UpdateDraft(r.Context(), page, input.Version); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	h.writePage(w, r, page.ID)
}

func (h Handler) recordPublishedSlugChange(ctx context.Context, previous pages.Page, locale, slug string) error {
	if h.RecordSlugRedirect == nil || previous.Status != pages.StatusPublished || previous.Slug == slug {
		return nil
	}
	if locale != previous.Locale {
		return nil
	}
	if h.DefaultLocale != "" && previous.Locale != h.DefaultLocale {
		return nil
	}
	return h.RecordSlugRedirect(ctx, previous.Locale, previous.Slug, slug)
}

func (h Handler) recordPreviousPublishedSlug(ctx context.Context, oldLocale, oldSlug string, page pages.Page) error {
	return h.recordPublishedSlugChange(ctx, pages.Page{Locale: oldLocale, Slug: oldSlug, Status: pages.StatusPublished}, page.Locale, page.Slug)
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
	oldLocale, oldSlug, err := h.Content.PublishedSlug(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		http.Error(w, "page publication unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := h.Content.Publish(r.Context(), r.PathValue("id"), expectedVersion); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	if oldSlug != "" {
		if page, getErr := h.Content.Get(r.Context(), h.DB, r.PathValue("id")); getErr == nil {
			if err := h.recordPreviousPublishedSlug(r.Context(), oldLocale, oldSlug, page); err != nil {
				http.Error(w, "page published, but its old link could not be redirected", http.StatusServiceUnavailable)
				return
			}
		}
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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := ui.AdminStartForRequest(w, r, "Sayfa çevirisi", "Yan yana çeviri", "pages"); err != nil {
			return
		}
		_ = ui.TranslationEditor(data).Render(r.Context(), w)
		_ = ui.AdminEnd(w)
		return
	}
	var form bytes.Buffer
	if err := adminstudio.RenderCreateForm(&form, adminstudio.FormData{Collection: h.AdminCollection, Action: "/admin/pages/" + page.ID, CSRFToken: security.CSRFTokenFromContext(r.Context()), HideSubmit: true, Values: map[string]string{"locale": page.Locale, "slug": page.Slug, "title": page.Title, "body": page.Body}}); err != nil {
		http.Error(w, "page admin unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := uicomponents.Input(uicomponents.InputProps{ID: "page-tags", Name: "tags", Type: "text", Label: "İçerik etiketleri (virgülle ayrılmış)", Value: strings.Join(page.Tags, ", ")}).Render(r.Context(), &form); err != nil {
		http.Error(w, "page admin unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Sayfa düzenle", "İçerik stüdyosu", "pages"); err != nil {
		return
	}
	if h.Presence != nil {
		endpoint := "/admin/pages/" + url.PathEscape(page.ID) + "/presence"
		_ = designsystem.EditPresence(page.ID, endpoint, security.CSRFTokenFromContext(r.Context())).Render(r.Context(), w)
	}
	if h.PreviewSigner != nil {
		_ = designsystem.DeviceViewportPreview(designsystem.DeviceViewportPreviewProps{Endpoint: "/admin/pages/" + url.PathEscape(page.ID) + "/preview-link", CSRF: security.CSRFTokenFromContext(r.Context())}).Render(r.Context(), w)
		_ = ui.DraftPreviewLinkForm("/admin/pages/"+url.PathEscape(page.ID)+"/preview-link", security.CSRFTokenFromContext(r.Context())).Render(r.Context(), w)
	}
	if history, ok := h.Content.Pages.(interface {
		ListRevisions(context.Context, database.Executor, string, int) ([]pages.RevisionSnapshot, error)
	}); ok {
		revisions, historyErr := history.ListRevisions(r.Context(), h.DB, page.ID, 50)
		if historyErr == nil && len(revisions) > 0 {
			selected := revisions[0]
			if requested, parseErr := strconv.Atoi(r.URL.Query().Get("revision")); parseErr == nil {
				for _, revision := range revisions {
					if revision.Revision == requested {
						selected = revision
						break
					}
				}
			}
			beforePreview, beforeErr := h.revisionPreview(r.Context(), selected.Title, selected.Body, selected.Blocks)
			afterPreview, afterErr := h.revisionPreview(r.Context(), page.Title, page.Body, page.Blocks)
			if beforeErr == nil && afterErr == nil {
				_ = ui.VisualRevisionCompare(beforePreview, afterPreview, "Sürüm "+strconv.Itoa(selected.Revision), "Geçerli taslak").Render(r.Context(), w)
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
			_ = ui.RevisionDiffPanel(changes, editor, selected.CreatedAt.Format("02.01.2006 15:04"), "/admin/pages/"+url.PathEscape(page.ID)+"/revisions/"+strconv.Itoa(selected.Revision)+"/restore", security.CSRFTokenFromContext(r.Context())).Render(r.Context(), w)
		}
	}
	blockEditor := renderBlockEditor(page.Blocks, security.CSRFTokenFromContext(r.Context()))
	if metrics := h.faqFeedbackMetrics(r, page.Blocks); len(metrics) > 0 {
		if err := designsystem.FAQFeedbackReport(metrics).Render(r.Context(), w); err != nil {
			http.Error(w, "page admin unavailable", http.StatusServiceUnavailable)
			return
		}
	}
	_, _ = fmt.Fprintf(w, `<section class="studio-card"><form class="studio-form" method="post" action="/admin/pages/%s" data-local-autosave data-autosave-resource="page:%s">%s<div class="block-editor"><div class="block-editor-heading"><div><h3>İçerik blokları</h3><p>Sayfanın görsel bölümlerini ve sıralamasını yönetin.</p></div><span>%d blok</span></div><fieldset>%s</fieldset></div><input type="hidden" name="version" value="%d"><div class="page-editor-submit-bar"><button type="submit" class="button button-primary">Taslağı kaydet</button></div></form></section><section class="studio-card publishing-card"><h2>Yayınlama</h2><p class="muted">Taslağı kaydedin veya doğrudan yayına alın.</p><div class="studio-actions"><form method="post" action="/admin/pages/%s/publish"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="version" value="%d"><button type="submit" class="button button-success">Yayına Al</button></form></div></section>`, template.HTMLEscapeString(page.ID), template.HTMLEscapeString(page.ID), form.String(), len(page.Blocks), blockEditor, page.Version, template.HTMLEscapeString(page.ID), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), page.Version)
	if h.assistantEnabled() {
		_ = designsystem.AIAssistantRequest("/admin/pages/"+url.PathEscape(page.ID)+"/ai-suggest", security.CSRFTokenFromContext(r.Context()), h.SupportedLocales).Render(r.Context(), w)
		if suggestion, ok := r.Context().Value(pageSuggestionKey{}).(pageSuggestion); ok {
			_ = designsystem.AIAssistantSuggestion(suggestion.BeforeTitle, suggestion.BeforeBody, suggestion.Result, "/admin/pages/"+url.PathEscape(page.ID)+"/ai-applied", security.CSRFTokenFromContext(r.Context())).Render(r.Context(), w)
		}
	}
	_ = ui.AdminEnd(w)
}

type pageSuggestionKey struct{}

type pageSuggestion struct {
	BeforeTitle string
	BeforeBody  string
	Result      ai.Suggestion
}

func (h Handler) adminAISuggest(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) || !h.assistantEnabled() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	operation := ai.Operation(strings.TrimSpace(r.FormValue("operation")))
	targetLocale := strings.TrimSpace(r.FormValue("target_locale"))
	if r.FormValue("consent") != "yes" || (operation != ai.Refine && operation != ai.SEO && operation != ai.Summary && operation != ai.Translate) || (operation == ai.Translate && !hasLocale(h.SupportedLocales, targetLocale)) {
		http.Error(w, "Öneri için bir işlem seçin ve içerik paylaşımını onaylayın.", http.StatusBadRequest)
		return
	}
	page, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		http.Error(w, "page unavailable", pageReadStatus(err))
		return
	}
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.Content.Transactions == nil {
		http.Error(w, "Öneri hizmeti şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	auditID := "ai-request-" + fmt.Sprintf("%x", sha256.Sum256([]byte(page.ID+"\x00"+user.ID+"\x00"+time.Now().UTC().Format(time.RFC3339Nano))))[:32]
	if err := h.Content.Transactions.Within(r.Context(), func(tx database.Executor) error {
		return h.AIAudit.Record(r.Context(), tx, auditID, "ai.suggestion.requested", page.ID)
	}); err != nil {
		http.Error(w, "Öneri hizmeti şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	result, err := h.Assistant.Suggest(r.Context(), ai.Request{
		Operation:    operation,
		Locale:       page.Locale,
		TargetLocale: targetLocale,
		Tone:         r.FormValue("tone"),
		Title:        page.Title,
		Body:         page.Body,
		Consent:      r.FormValue("consent") == "yes",
	})
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, ai.ErrConsent) || errors.Is(err, ai.ErrInvalidRequest) {
			status = http.StatusBadRequest
		}
		http.Error(w, "Öneri üretilemedi. İçeriğiniz değiştirilmedi.", status)
		return
	}
	suggestionContext := context.WithValue(r.Context(), pageSuggestionKey{}, pageSuggestion{BeforeTitle: page.Title, BeforeBody: page.Body, Result: result})
	h.adminPage(w, r.WithContext(suggestionContext))
}

func hasLocale(locales []string, target string) bool {
	for _, locale := range locales {
		if strings.EqualFold(strings.ReplaceAll(strings.TrimSpace(locale), "_", "-"), strings.ReplaceAll(strings.TrimSpace(target), "_", "-")) {
			return true
		}
	}
	return false
}

func (h Handler) adminAIApplied(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !editorAllowed(r) || !h.assistantEnabled() || user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	pageID := r.PathValue("id")
	if _, err := h.Content.Get(r.Context(), h.DB, pageID); err != nil {
		http.Error(w, "page unavailable", pageReadStatus(err))
		return
	}
	if h.Content.Transactions == nil {
		http.Error(w, "Öneri hizmeti şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	auditID := "ai-applied-" + fmt.Sprintf("%x", sha256.Sum256([]byte(pageID+"\x00"+user.ID+"\x00"+time.Now().UTC().Format(time.RFC3339Nano))))[:32]
	if err := h.Content.Transactions.Within(r.Context(), func(tx database.Executor) error {
		return h.AIAudit.Record(r.Context(), tx, auditID, "ai.suggestion.applied", pageID)
	}); err != nil {
		http.Error(w, "Öneri denetim kaydı tamamlanamadı.", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) editPresenceHeartbeat(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if h.Presence == nil || !editorAllowed(r) || user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if len(r.PathValue("id")) > 128 || strings.ContainsAny(r.PathValue("id"), "\x00\r\n") {
		http.Error(w, "invalid resource", http.StatusBadRequest)
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
	err := h.Presence.Heartbeat(realtime.PresenceSession{TenantID: tenantID, Resource: "page:" + r.PathValue("id"), UserID: user.ID, Name: name})
	if err != nil {
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
	api.WriteJSON(w, http.StatusOK, h.Presence.Active(tenantID, "page:"+r.PathValue("id"), user.ID))
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
	return `<!doctype html><html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><style>body{font:16px/1.6 system-ui,sans-serif;color:#172033;margin:0;padding:24px}main{max-width:860px;margin:auto}h1{font-size:2rem;line-height:1.2}.content-body{white-space:normal}img,video{max-width:100%;height:auto}a{color:#0369a1}.page-block{margin:24px 0}</style></head><body><main><h1>` + template.HTMLEscapeString(title) + `</h1><div class="content-body">` + ui.RichTextHTML(body) + `</div>` + rendered + `</main></body></html>`, nil
}

func (h Handler) restoreRevision(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	current, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		http.Error(w, "page unavailable", pageReadStatus(err))
		return
	}
	history, ok := h.Content.Pages.(interface {
		ListRevisions(context.Context, database.Executor, string, int) ([]pages.RevisionSnapshot, error)
	})
	if !ok {
		http.Error(w, "revision unavailable", http.StatusServiceUnavailable)
		return
	}
	revisions, err := history.ListRevisions(r.Context(), h.DB, current.ID, 50)
	if err != nil {
		http.Error(w, "revision unavailable", http.StatusServiceUnavailable)
		return
	}
	wanted, err := strconv.Atoi(r.PathValue("revision"))
	if err != nil || wanted <= 0 {
		http.Error(w, "revision unavailable", http.StatusBadRequest)
		return
	}
	for _, revision := range revisions {
		if revision.Revision != wanted {
			continue
		}
		current.Title, current.Body, current.Blocks = revision.Title, revision.Body, revision.Blocks
		if err := h.Content.UpdateDraft(r.Context(), current, current.Version); err != nil {
			http.Error(w, pageError(err), pageStatus(err))
			return
		}
		http.Redirect(w, r, "/admin/pages/"+url.PathEscape(current.ID), http.StatusSeeOther)
		return
	}
	http.NotFound(w, r)
}

func (h Handler) adminTranslate(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	source, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		http.Error(w, "page unavailable", pageReadStatus(err))
		return
	}
	locale := canonical.Locale(r.FormValue("locale"))
	if locale == "" || locale == source.Locale {
		http.Error(w, "invalid translation", http.StatusBadRequest)
		return
	}
	repository, ok := h.Content.Pages.(adminListRepository)
	if !ok {
		http.Error(w, "page unavailable", http.StatusServiceUnavailable)
		return
	}
	items, err := repository.ListAdmin(r.Context(), h.DB, 100)
	if err != nil {
		http.Error(w, "page unavailable", http.StatusServiceUnavailable)
		return
	}
	target := pages.Page{}
	for _, item := range items {
		if item.Locale == locale && item.Slug == source.Slug {
			target, err = h.Content.Get(r.Context(), h.DB, item.ID)
			if err != nil {
				http.Error(w, "page unavailable", pageReadStatus(err))
				return
			}
			break
		}
	}
	target.Locale, target.Slug, target.Title, target.Body = locale, source.Slug, strings.TrimSpace(r.FormValue("title")), r.FormValue("body")
	if target.ID == "" {
		id := make([]byte, 16)
		if _, err := rand.Read(id); err != nil {
			http.Error(w, "translation could not be saved", http.StatusInternalServerError)
			return
		}
		target.ID = "page-" + hex.EncodeToString(id)
		if err := h.Content.CreateDraft(r.Context(), target); err != nil {
			http.Error(w, pageError(err), pageStatus(err))
			return
		}
	} else {
		version, parseErr := strconv.Atoi(r.FormValue("version"))
		if parseErr != nil || version != target.Version {
			http.Error(w, "translation changed; reload and try again", http.StatusConflict)
			return
		}
		if err := h.Content.UpdateDraft(r.Context(), target, version); err != nil {
			http.Error(w, pageError(err), pageStatus(err))
			return
		}
	}
	http.Redirect(w, r, "/admin/pages/"+url.PathEscape(target.ID), http.StatusSeeOther)
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

func renderBlockEditor(blocks []pages.Block, csrf string) string {
	count := len(blocks)
	if count == 0 {
		count = 1
	}
	if count > pages.MaxBlocks {
		count = pages.MaxBlocks
	}
	var out strings.Builder
	out.WriteString(`<input type="hidden" name="block_count" value="` + strconv.Itoa(count) + `"><div data-block-editor>`)
	for i := 0; i < count; i++ {
		blockType, title, subtitle, buttonText, buttonURL, mediaAssetID, items := "", "", "", "", "", "", "[]"
		background, padding := "", ""
		payload := "{}"
		if i < len(blocks) {
			blockType, title, subtitle, buttonText, buttonURL, mediaAssetID, items = blockEditorValues(blocks[i])
			background, padding = blockStyleEditorValues(blocks[i])
			payload = string(blocks[i].Payload)
		}
		var styleFields bytes.Buffer
		if err := ui.BlockPresetFields(ui.BlockStyleFieldsProps{Index: i, Background: background, Padding: padding}).Render(context.Background(), &styleFields); err != nil {
			return ""
		}
		typeLabel := blockType
		for _, opt := range []struct{ typ, label string }{{"hero", "Hero"}, {"faq", "FAQ"}, {"cta", "CTA"}, {"pricing_table", "Pricing table"}, {"testimonials", "Testimonials"}, {"before_after", "Before / after"}, {"stats", "Statistics"}, {"contact_cta", "Call / WhatsApp"}, {"video", "Video"}, {"masonry", "Masonry gallery"}, {"hotspots", "Image hotspots"}, {"callout", "Callout"}, {"action_card", "Quick action card"}, {"carousel", "Image carousel"}, {"columns", "Columns"}, {"feature_list", "Feature list"}, {"video_hero", "Video hero"}, {"trust_bar", "Trust badges"}, {"figure", "Image with caption"}, {"business_status", "Business status"}} {
			if opt.typ == blockType {
				typeLabel = opt.label
				break
			}
		}
		if typeLabel == "" {
			typeLabel = "Yeni Blok"
		}
		fmt.Fprintf(&out, `<div class="block-editor-row" data-block-row data-block-type="%s"><div class="block-editor-row-header"><div class="block-editor-row-title"><strong data-block-number>Blok %d</strong><span class="block-type-badge" data-block-type-badge>%s</span></div><div class="block-editor-order"><button type="button" data-block-up aria-label="Move block %d up" title="Yukarı taşı">↑</button><button type="button" data-block-down aria-label="Move block %d down" title="Aşağı taşı">↓</button><button type="button" class="block-delete-btn" data-block-remove aria-label="Bloku kaldır" title="Bloku kaldır">✕ Kaldır</button></div></div>`, template.HTMLEscapeString(blockType), i+1, template.HTMLEscapeString(typeLabel), i+1, i+1)
		fmt.Fprintf(&out, `<label class="block-field block-field-type" data-block-field="type"><span class="block-field-label">Blok türü</span><select name="block_type_%d"><option value="">-- Blok Kaldır / Seçilmedi --</option>`, i)
		for _, option := range []struct{ typ, label string }{{"hero", "Hero (Giriş Başlığı & Buton)"}, {"faq", "FAQ (Sıkça Sorulan Sorular)"}, {"cta", "CTA (Eyleme Çağrı)"}, {"pricing_table", "Fiyatlandırma Tablosu"}, {"testimonials", "Müşteri Yorumları"}, {"before_after", "Öncesi / Sonrası"}, {"stats", "İstatistikler"}, {"contact_cta", "Arama / WhatsApp İletişim"}, {"video", "Video"}, {"masonry", "Fotoğraf Galerisi (Masonry)"}, {"hotspots", "Görsel Vurguları (Hotspots)"}, {"callout", "Vurgu Kutusu (Callout)"}, {"action_card", "Hızlı Eylem Kartı"}, {"carousel", "Görsel Kaydırıcı (Carousel)"}, {"columns", "Sütunlar"}, {"feature_list", "Özellik Listesi"}, {"video_hero", "Video Hero"}, {"trust_bar", "Güven Rozetleri"}, {"figure", "Başlıklı Görsel"}, {"business_status", "Çalışma Durumu"}} {
			fmt.Fprintf(&out, `<option value="%s" %s>%s</option>`, option.typ, selected(blockType, option.typ), option.label)
		}
		fmt.Fprintf(&out, `</select></label><label class="block-field" data-block-field="title"><span class="block-field-label">Başlık</span><input name="block_title_%d" value="%s" placeholder="Blok ana başlığı"></label><label class="block-field" data-block-field="subtitle"><span class="block-field-label">Alt başlık</span><input name="block_subtitle_%d" value="%s" placeholder="Açıklayıcı alt metin"></label><div class="block-field media-field" data-block-field="media"><label><span class="block-field-label">Medya Asset</span><input id="block_media_asset_id_%d" name="block_media_asset_id_%d" value="%s" placeholder="Medyadan seçin"></label><button class="button button-quiet media-picker-open" type="button" data-media-target="block_media_asset_id_%d">Medyadan seç</button></div><label class="block-field" data-block-field="buttonText"><span class="block-field-label">Buton metni</span><input name="block_button_text_%d" value="%s" placeholder="Örn: İncele →"></label><label class="block-field" data-block-field="buttonURL"><span class="block-field-label">Buton URL</span><input name="block_button_url_%d" value="%s" placeholder="/shop"></label><div class="block-field block-field-items" data-block-field="items"><span class="block-field-label">İçerik Maddeleri</span><textarea name="block_items_%d" class="block-items-raw" spellcheck="false">%s</textarea></div>%s<div class="block-field block-field-payload" data-block-field="payload"><span class="block-field-label">Blok JSON Verisi</span><textarea name="block_payload_%d" spellcheck="false">%s</textarea></div><input type="hidden" name="block_csrf_%d" value="%s"></div>`, i, template.HTMLEscapeString(title), i, template.HTMLEscapeString(subtitle), i, i, template.HTMLEscapeString(mediaAssetID), i, i, template.HTMLEscapeString(buttonText), i, template.HTMLEscapeString(buttonURL), i, template.HTMLEscapeString(items), template.HTML(styleFields.String()), i, template.HTMLEscapeString(payload), i, template.HTMLEscapeString(csrf))
	}
	fmt.Fprintf(&out, `<button type="button" class="block-add-btn" data-block-add %s>+ Yeni İçerik Bloku Ekle</button><p role="status" aria-live="polite" data-block-status></p></div>`, map[bool]string{true: `disabled aria-disabled="true"`}[count >= pages.MaxBlocks])
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

func blockStyleEditorValues(block pages.Block) (string, string) {
	var data map[string]json.RawMessage
	if json.Unmarshal(block.Payload, &data) != nil {
		return "", ""
	}
	var background, padding string
	_ = json.Unmarshal(data["bg_color"], &background)
	_ = json.Unmarshal(data["padding_y"], &padding)
	return background, padding
}

func parseBlockForm(r *http.Request) ([]pages.Block, error) {
	count, err := strconv.Atoi(r.FormValue("block_count"))
	if err != nil || count < 0 || count > pages.MaxBlocks {
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
		} else if isJSONPayloadBlock(typ) {
			payload = nil
			if json.Unmarshal([]byte(r.FormValue("block_payload_"+strconv.Itoa(i))), &payload) != nil || payload == nil {
				return nil, pages.ErrInvalidBlocks
			}
		} else {
			payload["subtitle"] = r.FormValue("block_subtitle_" + strconv.Itoa(i))
			payload["button_text"] = r.FormValue("block_button_text_" + strconv.Itoa(i))
			payload["button_url"] = r.FormValue("block_button_url_" + strconv.Itoa(i))
		}
		if background := r.FormValue("block_bg_color_" + strconv.Itoa(i)); background != "" && background != "default" {
			payload["bg_color"] = background
		}
		if padding := r.FormValue("block_padding_y_" + strconv.Itoa(i)); padding != "" && padding != "normal" {
			payload["padding_y"] = padding
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

func isJSONPayloadBlock(typ string) bool {
	switch typ {
	case "before_after", "stats", "contact_cta", "video", "masonry", "hotspots", "callout", "action_card", "carousel", "columns", "feature_list", "video_hero", "trust_bar", "figure", "business_status":
		return true
	}
	return false
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

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	if err := security.DecodeJSONStrict(w, r, target, 1<<20); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return err
	}
	return nil
}

func pageError(err error) string {
	if errors.Is(err, service.ErrInvalidPage) {
		return "Bilgiler eksik veya geçersiz. Alanları kontrol edip yeniden deneyin."
	}
	if errors.Is(err, service.ErrConflict) {
		return "Bu içerik başka biri tarafından güncellendi. Sayfayı yenileyip tekrar deneyin."
	}
	if errors.Is(err, pages.ErrRouteConflict) {
		return "Bu sayfa bağlantısı kullanımda. Farklı bir bağlantı adı seçin."
	}
	return "İşlem tamamlanamadı. Lütfen yeniden deneyin."
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

func editorAllowed(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return auth.Can(user, auth.PermissionContentWrite)
}
