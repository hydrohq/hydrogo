package ui

import (
	"bytes"
	"context"
	"regexp"
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
	// The old workbench dashboard widget assertions (wb-hero, wb-metric-strip,
	// etc.) were removed along with workbench itself (ADR-0341); hydrogo's
	// own flagship dashboard widgets are asserted in
	// TestHydroGoDashboardRendersFlagshipWidgets below.
}

func TestHydroGoDashboardRendersFlagshipWidgets(t *testing.T) {
	var out bytes.Buffer
	err := RenderAdminDashboard(&out, AdminDashboardData{Collections: []adminstudio.Collection{{ID: "articles", Label: "Articles", Description: "Content", Actions: []adminstudio.Action{{Label: "Open", Href: "/admin/articles"}}, Kits: []adminstudio.Kit{adminstudio.KitSEO}}}, Active: "dashboard"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"hydrogo-dashboard", "Merhaba", "Genel Bakış", "Sistem Durumu", "Hızlı İşlemler", "Articles"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("hydrogo dashboard widget %q missing", want)
		}
	}
}

func TestShadcnDashboardRendersDataTableWidgets(t *testing.T) {
	var out bytes.Buffer
	err := RenderAdminDashboard(&out, AdminDashboardData{Collections: []adminstudio.Collection{{ID: "articles", Label: "Articles", Description: "Content", Actions: []adminstudio.Action{{Label: "Open", Href: "/admin/articles"}}, Kits: []adminstudio.Kit{adminstudio.KitSEO}}}, Active: "dashboard"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"shadcn-dashboard", "<table", "<thead", "Koleksiyonlar", "Articles"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("shadcn dashboard widget %q missing", want)
		}
	}
}

func TestMediaHeaderControlsWrapInAFlexToolbarUnderNova(t *testing.T) {
	// Two fixed-column layouts (a plain 2-col grid, then two independent
	// flex-columns) both left a dead empty block under whichever side had
	// less content, since the page's media-card grid can only start below
	// the taller side. A flex-wrap toolbar has no "columns" to go uneven:
	// items pack left-to-right and only take the height their own line
	// needs, so there's never a leftover empty block.
	// The component CSS is now compiled by the standalone Tailwind CLI
	// (ADR-0340): attribute selectors print unquoted, and @apply expands
	// each utility's declaration in Tailwind's own canonical order rather
	// than the source order, so `display:flex` and `flex-wrap:wrap` are no
	// longer guaranteed to be textually adjacent - check both are present
	// within the rule instead of one exact adjacent substring.
	if toolbarRule := firstRuleBody(AdminMediaFormsCSS, `html[data-ui-theme=nova] .media-toolbar-row{`); toolbarRule == "" ||
		!strings.Contains(toolbarRule, "display:flex") || !strings.Contains(toolbarRule, "flex-wrap:wrap") {
		t.Fatal("Nova is missing the flex-wrap toolbar row for the media page's secondary controls (view toggle, folder tree, create-folder, tag filter, cleanup link, settings)")
	}
	if strings.Contains(AdminMediaFormsCSS, `.media-header-grid`) || strings.Contains(AdminMediaFormsCSS, `.media-header-col`) {
		t.Fatal("leftover media-header-grid/media-header-col rules from the earlier column-based layout should be removed, not left dead in the CSS")
	}
}

func TestMediaInlineFormSubmitButtonDoesNotStretchFullWidth(t *testing.T) {
	// Regression test: .studio-form is a 2-column CSS grid with no
	// justify-items override, so a lone <button type="submit"> landing alone
	// in the grid's second row stretched to fill its whole column by CSS
	// Grid's default justify-items:stretch - "Uygula"/"Oluştur" rendered as a
	// near-full-width bar instead of a normal button.
	// Unquoted attribute selector: see firstRuleBody's doc comment (ADR-0340).
	if !strings.Contains(AdminMediaFormsCSS, `.media-inline-form button[type=submit]{justify-self:start`) {
		t.Fatal("media-inline-form submit buttons are missing the justify-self:start override that keeps them from stretching")
	}
}

func TestMediaFormsDoNotUseDeadTailwindClasses(t *testing.T) {
	// Regression test: internal/ui/picture.templ (folder tree, tag/color
	// filter, settings and per-asset metadata disclosures) was written with
	// Tailwind utility class names (rounded, border-slate-200, bg-slate-900,
	// block, w-full, ...) that have zero matching CSS anywhere in this
	// codebase - Studio never loads Tailwind (ADR-0041) - so the whole media
	// page rendered unstyled in every visual family. It must instead reuse
	// the same semantic classes (.studio-card, .studio-form,
	// .studio-form-field, .button) the rest of Studio already relies on.
	var out bytes.Buffer
	if err := MediaSettingsForm(MediaSettingsFormProps{}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if err := MediaAssetMetadataForm(MediaMetadataFormProps{}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if err := MediaCreateFolderForm(nil, "", "").Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if err := MediaTagFilter("", "").Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if err := MediaFolderTree(nil, "").Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	for _, dead := range []string{"border-slate", "bg-slate-900", "text-slate", "rounded-lg border ", `class="block `, `class="mt-`, `class="w-full`, `class="px-`} {
		if strings.Contains(out.String(), dead) {
			t.Fatalf("media forms still use a dead Tailwind class fragment %q", dead)
		}
	}
	for _, want := range []string{`class="studio-card`, `class="studio-form`, `class="studio-form-field`, `class="button"`, `class="media-folder-link`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("media forms missing expected Studio design-system class %q", want)
		}
	}
}

func TestBrandBackgroundsNeverPairWithHardcodedWhiteText(t *testing.T) {
	// Regression test: several rules painted white text (color:#fff) on a
	// var(--brand)/var(--brand-hover) background. That's fine for hydrogo and
	// workbench (brand stays a saturated color in both variants), but shadcn
	// and Nova's dark variant deliberately make --brand a near-white zinc
	// tone (matching the shadcn/Vercel reference), so the hardcoded white
	// text became unreadable white-on-white - e.g. a plain <a> row action
	// like "Düzenle" (internal/adminstudio/list.go renders it with no class
	// at all, so only the global `.studio-card td a:not(.button):hover` rule
	// applied). Every such rule must use var(--brand-ink) instead, which is
	// #fff by default and flips to a dark color for shadcn-dark/nova-dark.
	if !strings.Contains(AdminCSS, "--brand-ink:#fff") {
		t.Fatal("base :root is missing the --brand-ink default")
	}
	for _, css := range []string{AdminShadcnCSS, AdminNovaCSS} {
		// The component CSS is now compiled by the standalone Tailwind CLI
		// (ADR-0341), which prints attribute selectors without quotes when
		// not required - functionally identical to `][data-theme="dark"]{`.
		if !strings.Contains(css, `][data-theme=dark]{`) || !strings.Contains(css, "--brand-ink:#09090b") {
			t.Fatal("shadcn/nova dark variant is missing a --brand-ink override")
		}
	}
	for _, css := range []string{AdminCSS, AdminShadcnCSS, AdminNovaCSS} {
		// `background:var(--brand)` became `background-color:var(--brand)`
		// under the Tailwind `bg-[...]` utility (ADR-0341); match both
		// spellings so this regression guard still catches the real bug.
		re := regexp.MustCompile(`background(?:-color)?:var\(--brand(?:-hover)?\)[^}]*color:#fff|color:#fff[^}]*background(?:-color)?:var\(--brand(?:-hover)?\)`)
		if re.MatchString(css) {
			t.Fatalf("found a brand-background rule still hardcoding color:#fff instead of var(--brand-ink): %q", re.FindString(css))
		}
	}
}

func TestNovaSearchOpensAsDropdownNotModal(t *testing.T) {
	// Nova must not move the search box into the centered
	// studio-search-dialog <dialog> (a full command-palette modal); it should
	// keep it in place and show .studio-search-results as an anchored
	// dropdown instead, matching the reference design's inline search.
	if !strings.Contains(AdminSPARouterJS, `data-ui-theme') === 'nova'`) {
		t.Fatal("__openSpotlight is missing the Nova dropdown branch")
	}
	idx := strings.Index(AdminSPARouterJS, "__openSpotlight = ()")
	if idx == -1 {
		t.Fatal("__openSpotlight definition not found")
	}
	branch := AdminSPARouterJS[idx : idx+420]
	if !strings.Contains(branch, "data-ui-theme') === 'nova'") || !strings.Contains(branch, "return;") {
		t.Fatal("__openSpotlight must check for Nova and return before opening the modal dialog")
	}
}

// Note: TestDashboardLayoutCSSIsNotScopedToWorkbenchOnly used to guard
// AdminWorkbenchDashboardCSS/AdminWorkbenchComponentsCSS staying unscoped so
// every family could render the one shared dashboard markup correctly. That
// invariant no longer applies: ADR-0341 removed workbench and gave hydrogo
// and shadcn their own isolated, per-family dashboard markup and compiled
// stylesheets (the same pattern Aurora already used), so there is no more
// shared dashboard markup/CSS to guard here. See
// internal/ui/hydrogoui and internal/ui/shadcnui's own contract tests
// (embed_test.go in each package) for the equivalent dead-class guard.

func TestRenderAdminDashboardIncludesNovaVariant(t *testing.T) {
	var out bytes.Buffer
	err := RenderAdminDashboard(&out, AdminDashboardData{Collections: []adminstudio.Collection{{ID: "articles", Label: "Articles", Description: "Content", Actions: []adminstudio.Action{{Label: "Open", Href: "/admin/articles"}}, Fields: []adminstudio.Field{{ID: "title", Label: "Title", Kind: adminstudio.FieldText}}, Kits: []adminstudio.Kit{adminstudio.KitSEO}}}, Active: "dashboard"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"nova-dashboard", "nova-stat-grid", "nova-table-card", "nova-activity-card", "nova-side-stack", "Sistem Durumu", "Articles", "Yönet"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("nova dashboard widget %q missing", want)
		}
	}
	// All dashboard variants must render server-side (family selection is
	// client-only); visibility is toggled purely by CSS.
	for _, want := range []string{"nova-dashboard"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("expected the nova dashboard variant to render, missing %q", want)
		}
	}
}

func TestNovaDashboardHidesPagesCollectionAndGatesCommerceAction(t *testing.T) {
	var out bytes.Buffer
	err := RenderAdminDashboard(&out, AdminDashboardData{
		Collections: []adminstudio.Collection{
			{ID: "news", Label: "Haberler", Description: "Content"},
			{ID: "pages", Label: "Sayfalar", Description: "Content"},
		},
		Active:                "dashboard",
		CommerceModuleEnabled: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Both dashboard variants render server-side (family selection is
	// client-only) and the Workbench variant intentionally still lists every
	// collection including pages, so scope the check to Nova's own table
	// markup instead of the whole page.
	// Markup now carries Tailwind utility classes alongside the hook class
	// (ADR-0340), so match the class attribute by prefix, not exact value.
	novaStart := strings.Index(out.String(), `class="nova-table-card `)
	novaEnd := strings.Index(out.String(), `class="nova-activity-card `)
	if novaStart == -1 || novaEnd == -1 || novaEnd < novaStart {
		t.Fatal("could not locate the Nova content-areas table in the rendered dashboard")
	}
	novaTable := out.String()[novaStart:novaEnd]
	if strings.Contains(novaTable, "Sayfalar") {
		t.Fatal("Nova dashboard's content-areas table still lists the pages collection; it should be hidden per the request to hide the pages feature from the Nova theme")
	}
	if !strings.Contains(novaTable, "Haberler") {
		t.Fatal("Nova dashboard's content-areas table lost the non-pages collection along with the filter")
	}
	// Workbench's own quick-action list (wb-action-list) is a separate,
	// pre-existing template that always includes "Ürün oluştur" regardless of
	// CommerceModuleEnabled; only Nova's activity card is in scope here.
	// Markup now carries Tailwind utility classes alongside the hook class
	// (ADR-0340), so match the class attribute by prefix, not exact value.
	novaActivityStart := strings.Index(out.String(), `class="nova-activity-card `)
	novaActivityEnd := strings.Index(out.String(), `class="nova-status-dot `)
	if novaActivityStart == -1 || novaActivityEnd == -1 || novaActivityEnd < novaActivityStart {
		t.Fatal("could not locate the Nova quick-actions card in the rendered dashboard")
	}
	if strings.Contains(out.String()[novaActivityStart:novaActivityEnd], "Ürün oluştur") {
		t.Fatal("Nova quick actions still show the commerce 'Ürün oluştur' shortcut with CommerceModuleEnabled=false")
	}

	var withCommerce bytes.Buffer
	err = RenderAdminDashboard(&withCommerce, AdminDashboardData{
		Collections:           []adminstudio.Collection{{ID: "news", Label: "Haberler", Description: "Content"}},
		Active:                "dashboard",
		CommerceModuleEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(withCommerce.String(), "Ürün oluştur") {
		t.Fatal("Nova quick actions should show 'Ürün oluştur' when CommerceModuleEnabled=true")
	}
}

func TestAdminShellIncludesNovaTopNav(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Tema", "", ""); err != nil {
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
	// Tailwind's compiler (ADR-0340) prints attribute selectors unquoted
	// and merges different selectors that end up with an identical
	// declaration body into one comma-separated rule, so `.studio-sidebar`
	// no longer has its own standalone "{display:none}" - it's grouped
	// with `.studio-mobile-menu-toggle`, which gets the same treatment.
	if !strings.Contains(AdminNovaDashboardLayoutCSS, `html[data-ui-theme=nova] .studio-sidebar,`) {
		t.Fatal(`admin shell missing nova sidebar-less layout rule for .studio-sidebar`)
	}
	if rule := firstRuleBody(AdminNovaDashboardLayoutCSS, `html[data-ui-theme=nova] .studio-sidebar,`); !strings.Contains(rule, "display:none") {
		t.Fatalf("nova .studio-sidebar rule is missing display:none: %q", rule)
	}
	if !strings.Contains(AdminNovaDashboardLayoutCSS, `html[data-ui-theme=nova] .studio-app{grid-template-columns:minmax(0,1fr)}`) {
		t.Fatal(`admin shell missing nova sidebar-less layout rule for .studio-app`)
	}
}

func TestNovaTopNavHidesPagesAndIncludesSidebarParity(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Tema", "", ""); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), `nova-topnav-link" href="/admin/pages"`) {
		t.Fatal("Nova top nav still links to /admin/pages; the pages feature was asked to be hidden from the Nova theme (endpoint stays, nav link should not)")
	}
	for _, want := range []string{
		// Raporlar/Üyeler/Geliştirici carry data-admin-only, same as their
		// sidebar counterparts, so editor-role accounts don't see them.
		`href="/admin/reports" data-admin-only>Raporlar</a>`,
		`href="/admin/developer" data-admin-only>Geliştirici &amp; API</a>`,
		`href="/admin/members" data-admin-only>Üyeler</a>`,
		`href="/admin/media">Medya</a>`,
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("Nova top nav missing sidebar-parity link %q", want)
		}
	}
}

func TestRemoveCommerceLinksStripsNovaTopNavCommerceLinks(t *testing.T) {
	// Regression test: the Nova top nav always rendered "Ürünler"/"Siparişler"
	// regardless of whether the commerce module was enabled, so clicking them
	// 404'd on any install without commerce active. removeCommerceLinks (the
	// same gate that already strips the sidebar's COMMERCE section) must also
	// strip the nova-commerce-start/end marked span in the Nova top nav.
	markup := `<div class="topbar-right"><nav class="nova-topnav-links">` + novaCommerceMarkerStart + `<a class="nova-topnav-link" href="/admin/commerce/products">Ürünler</a><a class="nova-topnav-link" href="/admin/commerce/orders">Siparişler</a>` + novaCommerceMarkerEnd + `<a class="nova-topnav-link" href="/admin/members">Üyeler</a></nav></div>`
	out := removeCommerceLinks(markup)
	if strings.Contains(out, "Ürünler") || strings.Contains(out, "Siparişler") || strings.Contains(out, "/admin/commerce/") {
		t.Fatalf("removeCommerceLinks did not strip the Nova top nav commerce links: %q", out)
	}
	if !strings.Contains(out, "Üyeler") {
		t.Fatal("removeCommerceLinks stripped more than the commerce-marked span")
	}
}

func TestAdminShellHidesInactiveDashboardVariantByCSS(t *testing.T) {
	// Tailwind's compiler (ADR-0340) merges `.nova-dashboard`, `.wb-dashboard`
	// and `.nova-topnav-wrap` into one comma-separated rule since they all
	// compile to the identical declaration body ("display:none"), and
	// prints attribute selectors unquoted.
	for _, want := range []string{
		`html:not([data-ui-theme=nova]) .nova-dashboard,`,
		`html[data-ui-theme=nova] .hydrogo-dashboard,`,
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

func TestRenderAdminDashboardShowsDynamicCollectionsOnlyWhenEnabled(t *testing.T) {
	var enabled, disabled bytes.Buffer
	if err := RenderAdminDashboard(&enabled, AdminDashboardData{Active: "dashboard", CollectionsModuleEnabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := RenderAdminDashboard(&disabled, AdminDashboardData{Active: "dashboard"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(enabled.String(), `href="/admin/collections"`) || strings.Contains(disabled.String(), `href="/admin/collections"`) {
		t.Fatal("dynamic collections navigation does not follow module activation")
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

func TestMediaTagFilterHasNoColorField(t *testing.T) {
	var out bytes.Buffer
	if err := MediaTagFilter("", "").Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), `name="color"`) || strings.Contains(out.String(), "Dominant renk") {
		t.Fatal("media tag filter should only filter by tag, not by dominant color")
	}
}

func TestMediaSettingsFormRendersAsADialogNotADisclosure(t *testing.T) {
	var out bytes.Buffer
	if err := MediaSettingsForm(MediaSettingsFormProps{}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `<dialog class="media-settings-dialog"`) {
		t.Fatal("media security/watermark settings must open as a dialog, not a <details> disclosure")
	}
	if !strings.Contains(out.String(), `data-media-settings-open`) || !strings.Contains(out.String(), `data-media-settings-close`) {
		t.Fatal("media settings dialog is missing its open/close triggers")
	}
	if strings.Contains(out.String(), `<details class="studio-card media-settings-disclosure">`) {
		t.Fatal("media settings form should no longer render as a <details> disclosure")
	}
}

func TestMediaToolbarRowButtonsShareTheSameHeightUnderNova(t *testing.T) {
	// The toolbar row mixes several different element types (button, <a>,
	// <summary>, a submit button) that each have their own default sizing;
	// they must all be pinned to the same explicit height so they line up,
	// even though their widths naturally differ by content.
	for _, want := range []string{
		// Unquoted attribute selector: see firstRuleBody's doc comment (ADR-0340).
		`html[data-ui-theme=nova] .media-toolbar-row .list-view-switch button,html[data-ui-theme=nova] .media-toolbar-row>.media-folder-tree>a,html[data-ui-theme=nova] .media-toolbar-row>.media-settings-disclosure>summary,html[data-ui-theme=nova] .media-toolbar-row>p>.button,html[data-ui-theme=nova] .media-toolbar-row>.media-settings-trigger,html[data-ui-theme=nova] .media-toolbar-row input,html[data-ui-theme=nova] .media-toolbar-row select,html[data-ui-theme=nova] .media-toolbar-row button{`,
	} {
		if !strings.Contains(AdminMediaFormsCSS, want) {
			t.Fatal("Nova's media toolbar row is missing the shared-height rule covering all its button-like children")
		}
	}
}

func TestNovaThemeToggleAndAvatarHaveVisibleBorders(t *testing.T) {
	if strings.Contains(AdminNovaDashboardLayoutCSS, `.ui-family-select,html[data-ui-theme=nova] .theme-toggle-btn{border-color:transparent`) {
		t.Fatal("Nova must not strip the theme toggle button's border - it should match the notification bell's border")
	}
	// Unquoted attribute selector, and `background` -> `background-color`
	// under the Tailwind `bg-` utility (ADR-0340); same declarations.
	if rule := firstRuleBody(AdminNovaDashboardLayoutCSS, `html[data-ui-theme=nova] .theme-toggle-btn{`); !strings.Contains(rule, "border:1px solid var(--line)") || !strings.Contains(rule, "background-color:var(--surface)") {
		t.Fatalf("Nova's theme toggle button is missing a visible border like the other topbar buttons: %q", rule)
	}
	// Declaration order and `background`->`background-color` changed under
	// Tailwind's compiler (ADR-0340); same declarations, checked by content.
	if rule := firstRuleBody(AdminNovaDashboardLayoutCSS, `.user-avatar{`); !strings.Contains(rule, "width:32px") || !strings.Contains(rule, "height:32px") ||
		!strings.Contains(rule, "border-radius:50%") || !strings.Contains(rule, "background-color:var(--surface)") ||
		!strings.Contains(rule, "color:var(--muted)") || !strings.Contains(rule, "border:1px solid var(--line)") {
		t.Fatalf("Nova's account avatar must use the same border/background treatment as the notification bell button: %q", rule)
	}
}

// firstRuleBody returns the declaration body (without the enclosing braces)
// of the first flat, non-nested CSS rule in css whose selector text starts
// with selectorPrefix (selectorPrefix must include the trailing "{"), or ""
// if not found. Since ADR-0340, component CSS is compiled by the standalone
// Tailwind CLI, which prints each @apply utility's declaration in its own
// canonical order rather than the original source order - tests that need
// to check several declarations landed in the same rule should use this
// instead of asserting one long, order-sensitive substring.
func firstRuleBody(css, selectorPrefix string) string {
	i := strings.Index(css, selectorPrefix)
	if i < 0 {
		return ""
	}
	start := i + len(selectorPrefix)
	end := strings.Index(css[start:], "}")
	if end < 0 {
		return ""
	}
	return css[start : start+end]
}

// TestDarkModeStudioMainKeepsGradientAndSolidBackground is a regression test
// for a real bug introduced by the ADR-0340 Tailwind conversion: the
// original hand-written rule was a single CSS `background` shorthand with
// TWO comma-separated layers - a decorative radial-gradient highlight, and
// a plain var(--bg) solid-color fallback
// (`background: radial-gradient(...), var(--bg)`). Tailwind's ambiguous
// `bg-[value]` arbitrary utility took the whole two-layer value as one
// `background-image`, producing
// `background-image: radial-gradient(...), var(--bg)` - but a bare
// var(--bg) resolving to a plain color is not a valid background-image
// layer, so per CSS's "invalid at computed-value time" rule the ENTIRE
// declaration was dropped by the browser, silently losing the gradient
// highlight (and the var(--bg) fallback) across EVERY admin family's dark
// mode, not just one - the rule has no [data-ui-theme="X"] qualifier. This
// is the root cause this test guards: the two layers must be two separate,
// individually-valid declarations (background-image + background-color).
func TestDarkModeStudioMainKeepsGradientAndSolidBackground(t *testing.T) {
	rule := firstRuleBody(AdminCSS, `html[data-theme=dark] .studio-main{`)
	if rule == "" {
		t.Fatal("AdminCSS is missing the html[data-theme=dark] .studio-main rule entirely")
	}
	if !strings.Contains(rule, "background-image:radial-gradient(") {
		t.Fatalf("dark-mode .studio-main is missing its radial-gradient background-image: %q", rule)
	}
	if !strings.Contains(rule, "background-color:var(--bg)") {
		t.Fatalf("dark-mode .studio-main is missing its background-color:var(--bg) fallback: %q", rule)
	}
	// The exact bug signature: a comma joining the gradient and var(--bg)
	// inside a single background-image value is invalid CSS and must never
	// reappear.
	if strings.Contains(rule, "),var(--bg)") || strings.Contains(rule, "), var(--bg)") {
		t.Fatalf("dark-mode .studio-main regressed: var(--bg) is joined into background-image by a comma again (invalid CSS, silently dropped by browsers): %q", rule)
	}
}
