package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestAdminReportsRenderLocalAnalyticsAndAccessibleDailyChart(t *testing.T) {
	var output bytes.Buffer
	err := RenderAdminReports(&output, AdminDashboardData{Active: "reports", AnalyticsEnabled: true, UniqueVisitors: 12, NoCookieStats: []NoCookieAnalyticsRow{
		{Day: "2026-09-28", Metric: "daily_views", EventCount: 7},
		{Metric: "page", Dimension: "/home", EventCount: 7},
		{Metric: "referrer", Dimension: "(direct)", EventCount: 7},
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"12 tekil ziyaretçi", "Son 30 günün görüntülenmeleri", `aria-label="2026-09-28: 7 görüntülenme"`, "/home", "Ziyaret kaynakları"} {
		if !strings.Contains(output.String(), marker) {
			t.Errorf("analytics report missing %q", marker)
		}
	}
}
