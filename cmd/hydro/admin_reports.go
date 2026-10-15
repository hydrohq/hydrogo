package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	analyticsstore "github.com/ardaozyildirim/hydrogo/internal/analytics/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

type noCookieAnalyticsRow struct {
	Day, Metric, Dimension string
	EventCount             int64
}

type contentViewReportRow struct {
	Day, TenantID, ContentType, ContentID string
	Views                                 int64
}

// adminReportsHandler serves request telemetry and analytics as JSON
// (default) or a CSV download. ?from=&to= (YYYY-MM-DD, at most a year apart)
// or ?range=7d|30d bound the content-view rows.
func (a *serveApp) adminReportsHandler(w http.ResponseWriter, r *http.Request) {
	cfg, modules := a.cfg, a.modules
	analyticsStore, memberService, runtimeMetrics := a.analyticsStore, a.memberService, a.metrics
	if !activePlatformAdmin(auth.UserFromContext(r.Context())) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	query := r.URL.Query()
	if query.Get("from") != "" || query.Get("to") != "" {
		query.Set("range", "custom")
		r.URL.RawQuery = query.Encode()
	}
	if query.Get("range") == "custom" {
		if _, _, valid := customAnalyticsDateRange(query.Get("from"), query.Get("to")); !valid {
			api.WriteValidation(w, r, map[string]string{"from": "invalid", "to": "invalid"})
			return
		}
	}
	format := query.Get("format")
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		api.WriteValidation(w, r, map[string]string{"format": "invalid"})
		return
	}
	var paths []observability.PathStatsSnapshot
	if runtimeMetrics != nil {
		paths = runtimeMetrics.PathStats()
	}
	if format == "json" {
		w.Header().Set("Cache-Control", "private, no-store")
		var stats members.MemberStats
		if modules["members"] {
			stats, _ = memberService.Stats(r.Context())
		}
		totalReq := uint64(0)
		totalViews := uint64(0)
		totalErrors := uint64(0)
		avgLatency := uint64(0)
		uniqueVisitors := int64(0)
		var noCookieStats []noCookieAnalyticsRow
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
		api.WriteData(w, http.StatusOK, out, nil)
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

func dailyNoCookieAnalytics(ctx context.Context, store analyticsstore.Store, tenantID, day string) ([]noCookieAnalyticsRow, int64, error) {
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
	result := make([]noCookieAnalyticsRow, 0, len(aggregates)+len(dailyViews))
	for date, count := range dailyViews {
		result = append(result, noCookieAnalyticsRow{Day: date, Metric: "daily_views", EventCount: count})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Day < result[j].Day })
	var dimensions []noCookieAnalyticsRow
	for key, count := range aggregates {
		parts := strings.SplitN(key, "\x00", 2)
		dimensions = append(dimensions, noCookieAnalyticsRow{Metric: parts[0], Dimension: parts[1], EventCount: count})
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
