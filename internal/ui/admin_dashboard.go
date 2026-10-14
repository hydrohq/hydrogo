package ui

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	"github.com/ardaozyildirim/hydrogo/internal/ui/hydrogoui"
	"github.com/ardaozyildirim/hydrogo/internal/ui/shell"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

type AdminDashboardData struct {
	// Request, when set, supplies the frame's runtime state (enabled modules,
	// banners, content types); the dashboard data itself does not depend on it.
	Request                  *http.Request
	User                     *auth.User
	SearchInsightsEnabled    bool
	Email                    string
	Collections              []adminstudio.Collection
	Nonce                    string
	Active                   string
	Metrics                  *observability.Metrics
	MemberStats              members.MemberStats
	DataTime                 string
	DataSource               string
	DateFilter               string
	DateFrom                 string
	DateTo                   string
	Locale                   string
	ContentViews             []ContentViewReportRow
	AnalyticsEnabled         bool
	NoCookieStats            []NoCookieAnalyticsRow
	UniqueVisitors           int64
	PushEnabled              bool
	EmailPreviewEnabled      bool
	CommerceModuleEnabled    bool
	CommerceWriteAllowed     bool
	CollectionsModuleEnabled bool
	PreviousLogin            string
	PreviousLoginIP          string
	FailedLoginAttempts      int
}

type ContentViewReportRow struct {
	Day, TenantID, ContentType, ContentID, Locale string
	ViewCount                                     int64
}

type NoCookieAnalyticsRow struct {
	Day, Metric, Dimension string
	EventCount             int64
}

func dashboardShellView(data AdminDashboardData, user *auth.User, title, eyebrow string) shell.View {
	if data.Request != nil {
		view := shellViewForRequest(data.Request, title, eyebrow, data.Active)
		view.Locale = data.Locale
		return view
	}
	view := shellView(user, nil, data.Locale, title, eyebrow, data.Active)
	view.User.Email = data.Email
	return view
}

func RenderAdminDashboard(w io.Writer, data AdminDashboardData) error {
	user := dashboardUser(data.User)
	data.CommerceWriteAllowed = data.CommerceModuleEnabled && auth.Can(user, auth.PermissionCommerceWrite)
	if err := writeShellStart(w, dashboardShellView(data, user, "Dashboard", "")); err != nil {
		return err
	}
	if err := AdminLoginSecurityNotice(data.PreviousLogin, data.PreviousLoginIP, data.FailedLoginAttempts).Render(context.Background(), w); err != nil {
		return err
	}
	if err := hydrogoui.DashboardTemplate.Execute(w, data); err != nil {
		return err
	}
	if err := dashboardKitSummaries(data.Collections).Render(context.Background(), w); err != nil {
		return err
	}
	return AdminEnd(w)
}

func RenderAdminReports(w io.Writer, data AdminDashboardData) error {
	user := dashboardUser(data.User)
	if err := writeShellStart(w, dashboardShellView(data, user, "Raporlar", "Analitik ve Performans")); err != nil {
		return err
	}
	ctx := context.Background()
	view := newReportsView(data)
	parts := []templ.Component{
		reportsHeader(view),
		designsystem.AnalyticsDateRange(designsystem.AnalyticsDateRangeProps{From: data.DateFrom, To: data.DateTo}),
	}
	if data.SearchInsightsEnabled {
		parts = append(parts, designsystem.SearchInsightsLink())
	}
	parts = append(parts, reportsBody(view))
	if len(data.ContentViews) > 0 {
		parts = append(parts, reportsContentViews(data.ContentViews))
	}
	if data.AnalyticsEnabled {
		parts = append(parts, noCookieAnalytics(data))
	}
	for _, part := range parts {
		if err := part.Render(ctx, w); err != nil {
			return err
		}
	}
	return AdminEnd(w)
}

// kitBadges renders an Edition collection's capability badges.
func kitBadges(kits []adminstudio.Kit) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		return adminstudio.RenderKitBadges(w, kits)
	})
}

func dashboardUser(user *auth.User) *auth.User {
	if user != nil {
		return user
	}
	return &auth.User{Role: auth.RoleAdmin, Active: true}
}
