package main

import (
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/members"
	newsservice "github.com/ardaozyildirim/hydrogo/modules/news"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

type adminDashboardUser struct {
	ID    string    `json:"id"`
	Email string    `json:"email"`
	Role  auth.Role `json:"role"`
}

type adminDashboardRuntime struct {
	StartedAt     time.Time `json:"started_at"`
	UptimeSeconds int64     `json:"uptime_seconds"`
	Requests      uint64    `json:"requests"`
	Views         uint64    `json:"views"`
	Errors        uint64    `json:"errors"`
	AvgLatencyMS  uint64    `json:"avg_latency_ms"`
}

type adminDashboardContent struct {
	PagesPublished int `json:"pages_published"`
	NewsPublished  int `json:"news_published"`
}

type adminDashboardAnalytics struct {
	UniqueVisitorsToday int64                  `json:"unique_visitors_today"`
	Daily               []noCookieAnalyticsRow `json:"daily"`
}

type adminDashboardBrokenLinks struct {
	Active bool   `json:"active"`
	JobID  string `json:"job_id,omitempty"`
}

type adminDashboard struct {
	User                auth.User                  `json:"user"`
	Modules             map[string]bool            `json:"modules"`
	SecurityNotice      *auth.LoginSecurityNotice  `json:"security_notice,omitempty"`
	Members             *members.MemberStats       `json:"members,omitempty"`
	Content             *adminDashboardContent     `json:"content,omitempty"`
	Runtime             *adminDashboardRuntime     `json:"runtime,omitempty"`
	Analytics           *adminDashboardAnalytics   `json:"analytics,omitempty"`
	BrokenLinks         *adminDashboardBrokenLinks `json:"broken_links,omitempty"`
	EmailPreviewEnabled bool                       `json:"email_preview_enabled"`
	PushEnabled         bool                       `json:"push_enabled"`
	GeneratedAt         time.Time                  `json:"generated_at"`
}

// adminDashboardHandler serves the cards the admin home shows: who is signed
// in, what is enabled, published counts, runtime telemetry and, for admins,
// the privacy-preserving analytics summary.
func (a *serveApp) adminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	modules := a.modules
	dashboard := adminDashboard{User: *user, Modules: enabledOnly(modules), EmailPreviewEnabled: a.smtpSender != nil, PushEnabled: modules["push"], GeneratedAt: time.Now().UTC()}
	if a.userStore != nil {
		if notice, err := a.userStore.LoginSecuritySummary(r.Context(), user.ID); err == nil {
			dashboard.SecurityNotice = &notice
		}
	}
	if a.linkScanManager != nil {
		tenantKey := "default"
		if scope, err := tenant.FromContext(r.Context()); err == nil {
			tenantKey = scope.TenantID
		}
		if job, ok := a.linkScanManager.Latest(tenantKey); ok && len(job.Findings) > 0 {
			dashboard.BrokenLinks = &adminDashboardBrokenLinks{Active: true, JobID: job.ID}
		}
	}
	if user.Role != auth.RoleAdmin {
		api.WriteData(w, http.StatusOK, dashboard, nil)
		return
	}
	if modules["members"] {
		if stats, err := a.memberService.Stats(r.Context()); err == nil {
			dashboard.Members = &stats
		}
	}
	if a.db != nil && (modules["pages"] || modules["news"]) {
		content := adminDashboardContent{}
		if modules["pages"] {
			_ = a.pageStore.StreamPublishedRoutes(r.Context(), a.db, func(pages.Page) error { content.PagesPublished++; return nil })
		}
		if modules["news"] {
			_ = a.newsRouteStore.StreamPublishedRoutes(r.Context(), a.db, func(newsservice.Item) error { content.NewsPublished++; return nil })
		}
		dashboard.Content = &content
	}
	if a.metrics != nil {
		started := a.metrics.StartedAt()
		dashboard.Runtime = &adminDashboardRuntime{StartedAt: started, UptimeSeconds: int64(time.Since(started).Seconds()), Requests: a.metrics.TotalRequests(), Views: a.metrics.TotalViews(), Errors: a.metrics.TotalErrors(), AvgLatencyMS: a.metrics.AvgLatencyMS()}
	}
	if modules["analytics"] && a.cfg.Analytics.NoCookieEnabled {
		tenantID := ""
		if scope, err := tenant.FromContext(r.Context()); err == nil {
			tenantID = scope.TenantID
		}
		if daily, unique, err := dailyNoCookieAnalytics(r.Context(), a.analyticsStore, tenantID, time.Now().UTC().Format("2006-01-02")); err == nil {
			dashboard.Analytics = &adminDashboardAnalytics{UniqueVisitorsToday: unique, Daily: daily}
		}
	}
	api.WriteData(w, http.StatusOK, dashboard, nil)
}

func enabledOnly(modules map[string]bool) map[string]bool {
	result := make(map[string]bool, len(modules))
	for name, enabled := range modules {
		if enabled {
			result[name] = true
		}
	}
	return result
}
