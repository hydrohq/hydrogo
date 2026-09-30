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

func TestRenderAdminDashboardIncludesNovaVariant(t *testing.T) {
	var out bytes.Buffer
	err := RenderAdminDashboard(&out, AdminDashboardData{Collections: []adminstudio.Collection{{ID: "articles", Label: "Articles", Description: "Content", Actions: []adminstudio.Action{{Label: "Open", Href: "/admin/articles"}}, Fields: []adminstudio.Field{{ID: "title", Label: "Title", Kind: adminstudio.FieldText}}, Kits: []adminstudio.Kit{adminstudio.KitSEO}}}, Active: "dashboard"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"nova-dashboard", "nova-stat-grid", "nova-table-card", "nova-activity-card", "Articles", "Yönet"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("nova dashboard widget %q missing", want)
		}
	}
	// Both dashboard variants must render server-side (family selection is
	// client-only); visibility is toggled purely by CSS.
	for _, want := range []string{"wb-dashboard", "nova-dashboard"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("expected both dashboard variants to render, missing %q", want)
		}
	}
}

func TestAdminShellIncludesNovaTopNav(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Tema", "", "nonce-test", ""); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`class="nova-topnav-wrap"`,
		`class="nova-topnav-brand"`,
		`nova-topnav-link is-active" href="/admin"`,
		`href="/admin/news">Haberler</a>`,
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("admin shell missing nova top nav contract %q", want)
		}
	}
	for _, want := range []string{
		`html[data-ui-theme="nova"] .studio-sidebar{display:none}`,
		`html[data-ui-theme="nova"] .studio-app{grid-template-columns:minmax(0,1fr)}`,
	} {
		if !strings.Contains(AdminNovaDashboardLayoutCSS, want) {
			t.Fatalf("admin shell missing nova sidebar-less layout rule %q", want)
		}
	}
}

func TestAdminShellHidesInactiveDashboardVariantByCSS(t *testing.T) {
	for _, want := range []string{
		`html:not([data-ui-theme="nova"]) .nova-dashboard{display:none}`,
		`html[data-ui-theme="nova"] .wb-dashboard{display:none}`,
	} {
		if !strings.Contains(AdminNovaDashboardLayoutCSS, want) {
			t.Fatalf("admin shell missing dashboard variant toggle rule %q", want)
		}
	}
}

func TestAdminLoginSecurityNoticeShowsPreviousSessionAndFailures(t *testing.T) {
	var out bytes.Buffer
	if err := RenderAdminDashboard(&out, AdminDashboardData{Active: "dashboard", PreviousLogin: "28 Sep 2026 12:15 UTC", PreviousLoginIP: "192.0.2.4", FailedLoginAttempts: 2}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"data-login-security-notice", "28 Sep 2026 12:15 UTC", "192.0.2.4", "2 başarısız giriş denemesi"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("dashboard missing %q", want)
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
