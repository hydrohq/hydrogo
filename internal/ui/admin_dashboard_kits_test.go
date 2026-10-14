package ui

import (
	"bytes"
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

func TestRenderAdminDashboardUsesSessionPermissions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		user  *auth.User
		write bool
	}{
		{name: "content write denied", user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: false}}},
		{name: "content write allowed", user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: true}}, write: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := RenderAdminDashboard(&out, AdminDashboardData{User: tc.user, Active: "dashboard"}); err != nil {
				t.Fatal(err)
			}
			markup := out.String()
			start := strings.Index(markup, `data-admin-permissions="`)
			if start < 0 {
				t.Fatal("dashboard omitted effective permission list")
			}
			start += len(`data-admin-permissions="`)
			end := strings.IndexByte(markup[start:], '"')
			if end < 0 {
				t.Fatal("dashboard permission list was not closed")
			}
			hasWrite := slices.Contains(strings.Fields(markup[start:start+end]), auth.PermissionContentWrite)
			if hasWrite != tc.write {
				t.Errorf("dashboard content write permission = %v, want %v", hasWrite, tc.write)
			}
		})
	}
}

func TestRenderAdminDashboardHidesCommerceActionsWithoutPermission(t *testing.T) {
	var out bytes.Buffer
	user := &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{
		auth.PermissionCommerceRead: false, auth.PermissionCommerceWrite: false,
	}}
	if err := RenderAdminDashboard(&out, AdminDashboardData{User: user, Active: "dashboard", CommerceModuleEnabled: true}); err != nil {
		t.Fatal(err)
	}
	visibleMarkup := inlineAssets.ReplaceAllString(out.String(), "")
	for _, href := range []string{`href="/admin/commerce/products"`, `href="/admin/commerce/orders"`, `href="/admin/commerce/products/new"`} {
		if strings.Contains(visibleMarkup, href) {
			t.Errorf("dashboard still renders denied commerce action %s", href)
		}
	}
}

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

func TestMediaFormsCSSHasNoLeftoverHeaderColumnRules(t *testing.T) {
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

func TestMobileMediaLibraryUsesSingleColumnAndTouchSizedControls(t *testing.T) {
	for _, want := range []string{
		"@media(max-width:520px){.media-grid{grid-template-columns:minmax(0,1fr)!important",
		".media-card button,.media-card select{height:auto!important;min-height:44px!important}",
	} {
		if !strings.Contains(AdminResponsiveCSS, want) {
			t.Errorf("mobile media library responsive CSS is missing %q", want)
		}
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
	// var(--brand)/var(--brand-hover) background, which becomes unreadable
	// white-on-white whenever a palette makes --brand a near-white tone -
	// e.g. a plain <a> row action like "Düzenle" (internal/adminstudio/list.go
	// renders it with no class at all, so only the global
	// `.studio-card td a:not(.button):hover` rule applied). Every such rule
	// must use var(--brand-ink) instead, which is #fff by default.
	if !strings.Contains(AdminCSS, "--brand-ink:#fff") {
		t.Fatal("base :root is missing the --brand-ink default")
	}
	for _, css := range []string{AdminCSS, AdminHydroGoCSS} {
		// `background:var(--brand)` became `background-color:var(--brand)`
		// under the Tailwind `bg-[...]` utility (ADR-0341); match both
		// spellings so this regression guard still catches the real bug.
		re := regexp.MustCompile(`background(?:-color)?:var\(--brand(?:-hover)?\)[^}]*color:#fff|color:#fff[^}]*background(?:-color)?:var\(--brand(?:-hover)?\)`)
		if re.MatchString(css) {
			t.Fatalf("found a brand-background rule still hardcoding color:#fff instead of var(--brand-ink): %q", re.FindString(css))
		}
	}
}

func TestDashboardCommerceShortcutFollowsCommerceModule(t *testing.T) {
	var without bytes.Buffer
	if err := RenderAdminDashboard(&without, AdminDashboardData{
		Collections:           []adminstudio.Collection{{ID: "news", Label: "Haberler", Description: "Content"}},
		Active:                "dashboard",
		CommerceModuleEnabled: false,
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(without.String(), "Ürün oluştur") {
		t.Fatal("dashboard quick actions show the commerce 'Ürün oluştur' shortcut with CommerceModuleEnabled=false")
	}
	var withCommerce bytes.Buffer
	if err := RenderAdminDashboard(&withCommerce, AdminDashboardData{
		Collections:           []adminstudio.Collection{{ID: "news", Label: "Haberler", Description: "Content"}},
		Active:                "dashboard",
		CommerceModuleEnabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(withCommerce.String(), "Ürün oluştur") {
		t.Fatal("dashboard quick actions should show 'Ürün oluştur' when CommerceModuleEnabled=true")
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
