package transport

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	adminstudiosql "github.com/ardaozyildirim/hydrogo/internal/adminstudio/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

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
	if !contentReadAllowed(r) {
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
	uiLocale := ui.AdminLocaleFromRequest(r)
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
					title += fmt.Sprintf(" ⚠ "+i18n.T(uiLocale, "admin.content.missing_images"), missing)
				}
			}
		}
		label := i18n.T(uiLocale, "admin.generated.list.view")
		if editorAllowed(r) {
			label = i18n.T(uiLocale, "admin.generated.list.edit")
		}
		actions := []adminstudio.Action{{Label: label, Href: "/admin/pages/" + item.ID}}
		if item.Status == pages.StatusPublished {
			publicPath := "/public/pages/" + url.PathEscape(item.Locale) + "/" + url.PathEscape(item.Slug) + "/html"
			actions = append(actions, adminstudio.Action{Label: i18n.T(uiLocale, "admin.content.copy_live_url"), CopyText: strings.TrimRight(strings.TrimSpace(h.SEOOrigin), "/") + publicPath})
		}
		rows = append(rows, adminstudio.Row{ID: item.ID, Values: map[string]string{"locale": item.Locale, "slug": item.Slug, "title": title, "status": string(item.Status), "views": strconv.FormatInt(viewCounts[item.ID], 10)}, ValueHTML: map[string]template.HTML{"views": renderViewCountBadge(r.Context(), viewCounts[item.ID])}, Status: adminstudio.StatusInfo{Value: string(item.Status)}, Actions: actions})
	}
	collection := h.AdminCollection
	if viewsAvailable {
		collection.Fields = append(append([]adminstudio.Field(nil), collection.Fields...), adminstudio.Field{ID: "views", Label: "Views", Kind: adminstudio.FieldText, VisibleInList: true})
	}
	view := pageListView{List: writerComponent(func(out io.Writer) error {
		return adminstudio.RenderList(out, adminstudio.ListData{
			Collection:        collection,
			Rows:              rows,
			StatusFilter:      r.URL.Query().Get("status"),
			UILocale:          uiLocale,
			ListCaption:       fmt.Sprintf(i18n.T(uiLocale, "admin.generated.list.caption"), collection.Label),
			ActionColumnLabel: i18n.T(uiLocale, "admin.generated.list.actions"),
		})
	})}
	if editorAllowed(r) {
		view.ImportLabel = i18n.T(uiLocale, "admin.content.import_csv_excel")
	}
	if viewsAvailable {
		view.ViewSort = &designsystem.ContentViewSortProps{Path: "/admin/pages", Status: r.URL.Query().Get("status"), Selected: r.URL.Query().Get("sort")}
	}
	renderAdminScreen(w, r, "Sayfalar", "", pageListScreen(view))
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
