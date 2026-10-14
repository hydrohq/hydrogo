package transport

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

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
	view := newsListView{List: writerComponent(func(out io.Writer) error {
		return adminstudio.RenderList(out, adminstudio.ListData{
			Collection:        collection,
			Rows:              rows,
			StatusFilter:      r.URL.Query().Get("status"),
			UILocale:          uiLocale,
			ListCaption:       fmt.Sprintf(i18n.T(uiLocale, "admin.generated.list.caption"), collection.Label),
			ActionColumnLabel: i18n.T(uiLocale, "admin.generated.list.actions"),
		})
	})}
	if viewsAvailable {
		view.ViewSort = &designsystem.ContentViewSortProps{Path: "/admin/news", Status: r.URL.Query().Get("status"), Selected: r.URL.Query().Get("sort")}
	}
	if r.URL.Query().Get("fragment") == "table" {
		var fragment bytes.Buffer
		if err := newsListFragment(view).Render(r.Context(), &fragment); err != nil {
			http.Error(w, "news admin unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fragment.WriteTo(w)
		return
	}
	renderAdminScreen(w, r, "Haberler", "", newsListScreen(view))
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
