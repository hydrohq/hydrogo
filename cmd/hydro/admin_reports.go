package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	analyticsstore "github.com/ardaozyildirim/hydrogo/internal/analytics/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

// adminDashboard serves /admin (business-owner or admin dashboard) and
// /admin/reports.
func (a *serveApp) adminDashboard(w http.ResponseWriter, r *http.Request) {
	cfg, modules, userStore, linkScanManager := a.cfg, a.modules, a.userStore, a.linkScanManager
	analyticsStore, memberService, runtimeMetrics := a.analyticsStore, a.memberService, a.metrics
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "Bu alana erişmek için giriş yapın.", http.StatusUnauthorized)
		return
	}
	if linkScanManager != nil {
		tenantKey := "default"
		if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
			tenantKey = scope.TenantID
		}
		if job, ok := linkScanManager.Latest(tenantKey); ok && len(job.Findings) > 0 {
			support := adminstudio.SupportConfigFromContext(r.Context())
			support.BrokenLinksActive, support.BrokenLinksJobID = true, job.ID
			r = r.WithContext(adminstudio.WithSupportConfig(r.Context(), support))
		}
	}
	if user.Role == auth.RoleEditor && r.URL.Path == "/admin" {
		securityNotice, noticeErr := userStore.LoginSecuritySummary(r.Context(), user.ID)
		if noticeErr != nil {
			http.Error(w, "Yönetim özeti şu anda açılamıyor.", http.StatusInternalServerError)
			return
		}
		previousLogin := ""
		if securityNotice.PreviousLoginAt != nil {
			previousLogin = securityNotice.PreviousLoginAt.Local().Format("02 Jan 2006 15:04 MST")
		}
		if err := ui.RenderBusinessOwnerDashboardWithSecurityNotice(w, r, modules, previousLogin, securityNotice.PreviousLoginIP, securityNotice.FailedAttempts); err != nil {
			http.Error(w, "Bu sayfa şu anda açılamıyor.", http.StatusInternalServerError)
		}
		return
	}
	if user.Role != auth.RoleAdmin {
		http.Error(w, "Bu alana erişim yetkiniz yok.", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.URL.Path == "/admin/reports" {
		var stats members.MemberStats
		if modules["members"] {
			stats, _ = memberService.Stats(r.Context())
		}
		dataTime := time.Now().Format("02.01.2006 15:04:05")
		dateRange := r.URL.Query().Get("range")
		dateFrom, dateTo := "", ""
		if dateRange == "custom" {
			var valid bool
			dateFrom, dateTo, valid = customAnalyticsDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
			if !valid {
				http.Error(w, "Choose a valid date range within the past year.", http.StatusBadRequest)
				return
			}
		}
		contentViews := []analyticsstore.ContentViewAggregate(nil)
		var noCookieStats []ui.NoCookieAnalyticsRow
		var uniqueVisitors int64
		if modules["analytics"] {
			contentViews, _ = analyticsStore.ListContentViews(r.Context(), contentViewFilterFromRequest(r, 100))
			tenantID := ""
			if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
				tenantID = scope.TenantID
			}
			noCookieStats, uniqueVisitors, _ = dailyNoCookieAnalytics(r.Context(), analyticsStore, tenantID, time.Now().UTC().Format("2006-01-02"))
		}
		contentViewRows := make([]ui.ContentViewReportRow, 0, len(contentViews))
		for _, row := range contentViews {
			contentViewRows = append(contentViewRows, ui.ContentViewReportRow{Day: row.Day, TenantID: row.TenantID, ContentType: row.ContentType, ContentID: row.ContentID, Locale: row.Locale, ViewCount: row.ViewCount})
		}
		if err := ui.RenderAdminReports(w, ui.AdminDashboardData{
			User:        user,
			Email:       user.Email,
			Collections: a.activeStudioCollections(),

			Active:                   "reports",
			Metrics:                  runtimeMetrics,
			MemberStats:              stats,
			DataTime:                 dataTime,
			DataSource:               "Proses İçi Bellek Telemetrisi",
			DateFilter:               dateRange,
			DateFrom:                 dateFrom,
			DateTo:                   dateTo,
			Locale:                   adminLocaleFromRequest(r),
			ContentViews:             contentViewRows,
			AnalyticsEnabled:         modules["analytics"] && cfg.Analytics.NoCookieEnabled,
			SearchInsightsEnabled:    modules["search"] && modules["analytics"],
			CollectionsModuleEnabled: modules["collections"],
			NoCookieStats:            noCookieStats,
			UniqueVisitors:           uniqueVisitors,
		}); err != nil {
			http.Error(w, "admin reports unavailable", http.StatusInternalServerError)
		}
		return
	}
	securityNotice, noticeErr := userStore.LoginSecuritySummary(r.Context(), user.ID)
	if noticeErr != nil {
		http.Error(w, "Yönetim özeti şu anda açılamıyor.", http.StatusInternalServerError)
		return
	}
	previousLogin := ""
	if securityNotice.PreviousLoginAt != nil {
		previousLogin = securityNotice.PreviousLoginAt.Local().Format("02 Jan 2006 15:04 MST")
	}
	if err := ui.RenderAdminDashboard(w, ui.AdminDashboardData{User: user, Email: user.Email, Collections: a.activeStudioCollections(), Active: "dashboard", Locale: adminLocaleFromRequest(r), PushEnabled: modules["push"], EmailPreviewEnabled: a.smtpSender != nil, CommerceModuleEnabled: modules["commerce"], CollectionsModuleEnabled: modules["collections"], PreviousLogin: previousLogin, PreviousLoginIP: securityNotice.PreviousLoginIP, FailedLoginAttempts: securityNotice.FailedAttempts}); err != nil {
		http.Error(w, "admin dashboard unavailable", http.StatusInternalServerError)
	}
}

// adminReportsExport downloads request telemetry and analytics as CSV or JSON.
func (a *serveApp) adminReportsExport(w http.ResponseWriter, r *http.Request) {
	cfg, modules := a.cfg, a.modules
	analyticsStore, memberService, runtimeMetrics := a.analyticsStore, a.memberService, a.metrics
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.URL.Query().Get("range") == "custom" {
		if _, _, valid := customAnalyticsDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to")); !valid {
			http.Error(w, "Choose a valid date range within the past year.", http.StatusBadRequest)
			return
		}
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	var paths []observability.PathStatsSnapshot
	if runtimeMetrics != nil {
		paths = runtimeMetrics.PathStats()
	}
	if format == "json" {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"hydro-report-%s.json\"", time.Now().Format("20060102-150405")))
		var stats members.MemberStats
		if modules["members"] {
			stats, _ = memberService.Stats(r.Context())
		}
		totalReq := uint64(0)
		totalViews := uint64(0)
		totalErrors := uint64(0)
		avgLatency := uint64(0)
		uniqueVisitors := int64(0)
		var noCookieStats []ui.NoCookieAnalyticsRow
		var startedAt time.Time
		if runtimeMetrics != nil {
			totalReq = runtimeMetrics.TotalRequests()
			totalViews = runtimeMetrics.TotalViews()
			totalErrors = runtimeMetrics.TotalErrors()
			avgLatency = runtimeMetrics.AvgLatencyMS()
			startedAt = runtimeMetrics.StartedAt()
		}
		if modules["analytics"] && cfg.Analytics.NoCookieEnabled {
			tenantID := ""
			if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
				tenantID = scope.TenantID
			}
			noCookieStats, uniqueVisitors, _ = dailyNoCookieAnalytics(r.Context(), analyticsStore, tenantID, time.Now().UTC().Format("2006-01-02"))
		}
		out := map[string]any{
			"timestamp":       time.Now().UTC().Format(time.RFC3339),
			"source":          "HydroGo in-memory telemetry",
			"process_started": startedAt.Format(time.RFC3339),
			"summary": map[string]any{
				"total_requests":   totalReq,
				"total_views":      totalViews,
				"total_errors":     totalErrors,
				"avg_latency_ms":   avgLatency,
				"unique_visitors":  uniqueVisitors,
				"total_members":    stats.TotalMembers,
				"active_members":   stats.ActiveMembers,
				"verified_members": stats.VerifiedMembers,
			},
			"paths": paths,
			"dimensions": func() []observability.DimensionStatsSnapshot {
				if runtimeMetrics == nil {
					return nil
				}
				return runtimeMetrics.DimensionStats()
			}(),
			"content_views": func() []analyticsstore.ContentViewAggregate {
				if !modules["analytics"] {
					return nil
				}
				rows, _ := analyticsStore.ListContentViews(r.Context(), contentViewFilterFromRequest(r, 500))
				return rows
			}(),
			"analytics_daily": noCookieStats,
		}
		_ = json.NewEncoder(w).Encode(out)
		return
	}

	// CSV format
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"hydro-report-%s.csv\"", time.Now().Format("20060102-150405")))
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"Path", "Views", "Requests", "Errors", "AvgLatencyMS", "P50MS", "P95MS", "P99MS", "BytesIn", "BytesOut", "Status2xx", "Status3xx", "Status4xx", "Status5xx", "Timeouts", "RateLimited"})
	for _, p := range paths {
		_ = writer.Write([]string{p.Path, strconv.FormatUint(p.Views, 10), strconv.FormatUint(p.Requests, 10), strconv.FormatUint(p.Errors, 10), strconv.FormatUint(p.AvgMS, 10), strconv.FormatUint(p.P50MS, 10), strconv.FormatUint(p.P95MS, 10), strconv.FormatUint(p.P99MS, 10), strconv.FormatUint(p.BytesIn, 10), strconv.FormatUint(p.BytesOut, 10), strconv.FormatUint(p.Status2xx, 10), strconv.FormatUint(p.Status3xx, 10), strconv.FormatUint(p.Status4xx, 10), strconv.FormatUint(p.Status5xx, 10), strconv.FormatUint(p.Timeouts, 10), strconv.FormatUint(p.RateLimited, 10)})
	}
	if runtimeMetrics != nil {
		_ = writer.Write([]string{})
		_ = writer.Write([]string{"Dimensions", "TenantID", "Module", "Requests", "Errors", "AvgLatencyMS"})
		for _, d := range runtimeMetrics.DimensionStats() {
			_ = writer.Write([]string{"", d.TenantID, d.Module, strconv.FormatUint(d.Requests, 10), strconv.FormatUint(d.Errors, 10), strconv.FormatUint(d.AvgMS, 10)})
		}
	}
	if modules["analytics"] {
		if rows, err := analyticsStore.ListContentViews(r.Context(), contentViewFilterFromRequest(r, 500)); err == nil && len(rows) > 0 {
			_ = writer.Write([]string{})
			_ = writer.Write([]string{"ContentViews", "Day", "TenantID", "ContentType", "ContentID", "Locale", "ViewCount"})
			for _, row := range rows {
				_ = writer.Write([]string{"", row.Day, row.TenantID, row.ContentType, row.ContentID, row.Locale, strconv.FormatInt(row.ViewCount, 10)})
			}
		}
	}
	if modules["analytics"] && cfg.Analytics.NoCookieEnabled {
		tenantID := ""
		if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
			tenantID = scope.TenantID
		}
		if rows, err := analyticsStore.ListDailyStats(r.Context(), time.Now().UTC().Format("2006-01-02"), tenantID, 500); err == nil && len(rows) > 0 {
			_ = writer.Write([]string{})
			_ = writer.Write([]string{"DailyAnalytics", "Metric", "Dimension", "EventCount"})
			for _, row := range rows {
				_ = writer.Write([]string{"", row.Metric, row.Dimension, strconv.FormatInt(row.EventCount, 10)})
			}
		}
	}
	writer.Flush()
}

func contentViewFilterFromRequest(r *http.Request, limit int) analyticsstore.ContentViewFilter {
	filter := analyticsstore.ContentViewFilter{Limit: limit}
	if r == nil {
		return filter
	}
	q := r.URL.Query()
	filter.TenantID, filter.ContentType, filter.ContentID, filter.Locale = strings.TrimSpace(q.Get("tenant_id")), strings.TrimSpace(q.Get("content_type")), strings.TrimSpace(q.Get("content_id")), strings.TrimSpace(q.Get("locale"))
	switch q.Get("range") {
	case "7d", "30d":
		days := 7
		if q.Get("range") == "30d" {
			days = 30
		}
		filter.FromDay = time.Now().UTC().AddDate(0, 0, -days+1).Format("2006-01-02")
		filter.ToDay = time.Now().UTC().Format("2006-01-02")
	case "custom":
		if from, to, ok := customAnalyticsDateRange(q.Get("from"), q.Get("to")); ok {
			filter.FromDay, filter.ToDay = from, to
		}
	}
	return filter
}

func customAnalyticsDateRange(from, to string) (string, string, bool) {
	start, startErr := time.Parse("2006-01-02", from)
	end, endErr := time.Parse("2006-01-02", to)
	today, _ := time.Parse("2006-01-02", time.Now().UTC().Format("2006-01-02"))
	if startErr != nil || endErr != nil || end.Before(start) || end.After(today) || end.Sub(start) > 365*24*time.Hour {
		return "", "", false
	}
	return start.Format("2006-01-02"), end.Format("2006-01-02"), true
}

func dailyNoCookieAnalytics(ctx context.Context, store analyticsstore.Store, tenantID, day string) ([]ui.NoCookieAnalyticsRow, int64, error) {
	end, err := time.Parse("2006-01-02", day)
	if err != nil {
		return nil, 0, err
	}
	start := end.AddDate(0, 0, -29).Format("2006-01-02")
	rows, err := store.ListDailyStatsBetween(ctx, start, day, tenantID, 5000)
	if err != nil {
		return nil, 0, err
	}
	aggregates := make(map[string]int64)
	dailyViews := make(map[string]int64)
	var unique int64
	todayRows, err := store.ListDailyStats(ctx, day, tenantID, 500)
	if err != nil {
		return nil, 0, err
	}
	for _, row := range todayRows {
		if row.Metric == "unique_visitors" {
			unique = row.EventCount
			break
		}
	}
	for _, row := range rows {
		if row.Metric == "page" {
			aggregates[row.Metric+"\x00"+row.Dimension] += row.EventCount
			dailyViews[row.Day] += row.EventCount
		} else if row.Metric != "unique_visitors" {
			aggregates[row.Metric+"\x00"+row.Dimension] += row.EventCount
		}
	}
	result := make([]ui.NoCookieAnalyticsRow, 0, len(aggregates)+len(dailyViews))
	for date, count := range dailyViews {
		result = append(result, ui.NoCookieAnalyticsRow{Day: date, Metric: "daily_views", EventCount: count})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Day < result[j].Day })
	var dimensions []ui.NoCookieAnalyticsRow
	for key, count := range aggregates {
		parts := strings.SplitN(key, "\x00", 2)
		dimensions = append(dimensions, ui.NoCookieAnalyticsRow{Metric: parts[0], Dimension: parts[1], EventCount: count})
	}
	sort.Slice(dimensions, func(i, j int) bool {
		if dimensions[i].Metric != dimensions[j].Metric {
			return dimensions[i].Metric < dimensions[j].Metric
		}
		if dimensions[i].EventCount != dimensions[j].EventCount {
			return dimensions[i].EventCount > dimensions[j].EventCount
		}
		return dimensions[i].Dimension < dimensions[j].Dimension
	})
	result = append(result, dimensions...)
	return result, unique, nil
}
