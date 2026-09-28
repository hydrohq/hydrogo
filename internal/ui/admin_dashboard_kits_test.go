package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
)

func TestRenderAdminDashboardShowsCapabilitySummary(t *testing.T) {
	var out bytes.Buffer
	err := RenderAdminDashboard(&out, AdminDashboardData{Collections: []adminstudio.Collection{{ID: "articles", Label: "Articles", Description: "Content", Actions: []adminstudio.Action{{Label: "Open", Href: "/admin/articles"}}, Fields: []adminstudio.Field{{ID: "title", Label: "Title", Kind: adminstudio.FieldText}}, Kits: []adminstudio.Kit{adminstudio.KitSEO}}}, Active: "dashboard"})
	if err != nil || !strings.Contains(out.String(), "Etkin yetenekler") || !strings.Contains(out.String(), "seo") {
		t.Fatalf("dashboard kit summary missing: %v", err)
	}
	for _, want := range []string{"wb-dashboard", "wb-hero", "wb-metric-strip", "wb-collection-grid", "wb-action-list", "wb-health-list"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("workbench dashboard widget %q missing", want)
		}
	}
}

func TestRenderAdminDashboardUsesCollectionActionRoute(t *testing.T) {
	var out bytes.Buffer
	err := RenderAdminDashboard(&out, AdminDashboardData{Collections: []adminstudio.Collection{{ID: "commerce", Label: "Products", Description: "Catalog", Actions: []adminstudio.Action{{Label: "Catalog", Href: "/admin/commerce/products"}}, Fields: []adminstudio.Field{{ID: "title", Label: "Title", Kind: adminstudio.FieldText}}}}, Active: "dashboard"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `href="/admin/commerce/products"`) {
		t.Fatal("dashboard collection did not use its declared action route")
	}
	if strings.Contains(out.String(), `href="/admin/commerce"`) {
		t.Fatal("dashboard collection used the raw collection id as a route")
	}
}

func TestPushQuickActionAppearsOnlyWhenModuleEnabled(t *testing.T) {
	var active, inactive bytes.Buffer
	if err := RenderAdminDashboard(&active, AdminDashboardData{Active: "dashboard", PushEnabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := RenderAdminDashboard(&inactive, AdminDashboardData{Active: "dashboard"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(active.String(), `href="/admin/push"`) || strings.Contains(inactive.String(), `href="/admin/push"`) {
		t.Fatal("push navigation does not follow module activation")
	}
}

func TestEmailPreviewActionAppearsOnlyWhenMailerConfigured(t *testing.T) {
	var active, inactive bytes.Buffer
	if err := RenderAdminDashboard(&active, AdminDashboardData{Active: "dashboard", EmailPreviewEnabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := RenderAdminDashboard(&inactive, AdminDashboardData{Active: "dashboard"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(active.String(), `href="/admin/mailer/test-preview"`) || strings.Contains(inactive.String(), `href="/admin/mailer/test-preview"`) {
		t.Fatal("email preview navigation does not follow mailer availability")
	}
}
