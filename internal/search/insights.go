package search

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

type SearchInsightsReader interface {
	ListSearchInsights(context.Context, string, string, string, int) ([]analytics.SearchInsight, error)
	ListZeroResultInsights(context.Context, string, string, string, int) ([]analytics.SearchInsight, error)
}

func SearchInsightsHandler(store SearchInsightsReader, requireTenantScope bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || !user.Active || user.Role != auth.RoleAdmin {
			http.Error(w, "Bu alana erişim yetkiniz yok.", http.StatusForbidden)
			return
		}
		if store == nil {
			http.Error(w, "Arama raporu kullanılamıyor.", http.StatusServiceUnavailable)
			return
		}
		days := 30
		if parsed, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && (parsed == 7 || parsed == 30) {
			days = parsed
		}
		tenantID := "default"
		if scope, err := tenant.FromContext(r.Context()); err == nil {
			tenantID = scope.TenantID
		} else if requireTenantScope {
			http.Error(w, "Arama raporu kullanılamıyor.", http.StatusForbidden)
			return
		}
		to := time.Now().UTC()
		from := to.AddDate(0, 0, -days+1)
		items, err := store.ListSearchInsights(r.Context(), tenantID, from.Format("2006-01-02"), to.Format("2006-01-02"), 10)
		if err != nil {
			http.Error(w, "Arama raporu şu anda açılamıyor.", http.StatusServiceUnavailable)
			return
		}
		zeroItems, err := store.ListZeroResultInsights(r.Context(), tenantID, from.Format("2006-01-02"), to.Format("2006-01-02"), 10)
		if err != nil {
			http.Error(w, "Arama raporu şu anda açılamıyor.", http.StatusServiceUnavailable)
			return
		}
		rows := make([]designsystem.SearchInsightRow, 0, len(items))
		for _, item := range items {
			rows = append(rows, designsystem.SearchInsightRow{Term: item.Term, Locale: item.Locale, SearchCount: item.SearchCount, ZeroResultCount: item.ZeroResultCount})
		}
		zeroRows := make([]designsystem.SearchInsightRow, 0, len(zeroItems))
		for _, item := range zeroItems {
			zeroRows = append(zeroRows, designsystem.SearchInsightRow{Term: item.Term, Locale: item.Locale, SearchCount: item.SearchCount, ZeroResultCount: item.ZeroResultCount})
		}
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := ui.AdminStartForRequest(w, r, "Site arama içgörüleri", "Popüler aramalar ve sonuçsuz sorgular", "reports"); err != nil {
			return
		}
		if err := designsystem.SearchInsightsTable(designsystem.SearchInsightsProps{RangeDays: days, Items: rows, ZeroResultItems: zeroRows}).Render(r.Context(), w); err != nil {
			http.Error(w, "Arama raporu şu anda açılamıyor.", http.StatusServiceUnavailable)
			return
		}
		_ = ui.AdminEnd(w)
	})
}
