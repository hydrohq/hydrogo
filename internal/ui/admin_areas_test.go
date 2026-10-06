package ui

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
)

func TestRenderAdminResourceUsage(t *testing.T) {
	var out bytes.Buffer
	err := RenderAdminResourceUsage(&out, AdminResourceUsageData{
		Nonce: "nonce-test",
		Snapshot: observability.RuntimeSnapshot{
			ReadinessState: true, Requests: 12, AvgLatencyMS: 8,
			HeapAllocBytes: 1024 * 1024, HeapInUseBytes: 2 * 1024 * 1024,
			StartedAt: observability.NewMetrics().StartedAt(),
		},
	})
	if err != nil {
		t.Fatalf("RenderAdminResourceUsage() error = %v", err)
	}
	for _, want := range []string{"Kaynak tüketimi", "Sistem kaynakları", "Heap kullanımı", "Toplam istek", "/admin/developer/resources"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("rendered resource page missing %q", want)
		}
	}
}

func TestAdminShellAddsTrashNavigationOnlyWhenEnabled(t *testing.T) {
	request := httptest.NewRequest("GET", "/admin/trash", nil)
	var enabled bytes.Buffer
	if err := AdminStartForRequestWithTrash(&enabled, request, "Çöp kutusu", "", "trash"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(enabled.String(), `href="/admin/trash" aria-current="page"`) {
		t.Fatal("trash navigation link is missing")
	}
	var disabled bytes.Buffer
	if err := AdminStartForRequest(&disabled, request, "Dashboard", "", "dashboard"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(disabled.String(), `href="/admin/trash"`) {
		t.Fatal("trash navigation link appeared without the optional route")
	}
}

func TestAdminShellShowsMaintenanceBanner(t *testing.T) {
	request := httptest.NewRequest("GET", "/admin", nil)
	request = request.WithContext(adminstudio.WithSupportConfig(request.Context(), adminstudio.SupportConfig{MaintenanceActive: true}))
	var out bytes.Buffer
	if err := AdminStartForRequest(&out, request, "Dashboard", "", "dashboard"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Site bakım modunda") || !strings.Contains(out.String(), "/admin/site/maintenance") {
		t.Fatalf("maintenance banner missing from admin shell")
	}
}

func TestAdminShellShowsBrokenLinksWarning(t *testing.T) {
	request := httptest.NewRequest("GET", "/admin", nil)
	request = request.WithContext(adminstudio.WithSupportConfig(request.Context(), adminstudio.SupportConfig{BrokenLinksActive: true, BrokenLinksJobID: "report"}))
	var out bytes.Buffer
	if err := AdminStartForRequest(&out, request, "Dashboard", "", "dashboard"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "erişilemeyen bağlantılar") || !strings.Contains(out.String(), "/admin/site/link-check?job=report") {
		t.Fatalf("broken links warning missing from admin shell")
	}
}

func TestAdminShellShowsSafeModeWarning(t *testing.T) {
	request := httptest.NewRequest("GET", "/admin", nil)
	request = request.WithContext(adminstudio.WithSupportConfig(request.Context(), adminstudio.SupportConfig{SafeModeActive: true}))
	var out bytes.Buffer
	if err := AdminStartForRequest(&out, request, "Dashboard", "", "dashboard"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "System is operating in Safe Mode") {
		t.Fatalf("safe mode warning missing: %s", out.String())
	}
}

func TestAdminShellShowsDismissibleStaffNoticeAsEscapedTypedContent(t *testing.T) {
	request := httptest.NewRequest("GET", "/admin", nil)
	request = request.WithContext(adminstudio.WithSupportConfig(request.Context(), adminstudio.SupportConfig{StaffNoticeActive: true, StaffNoticeMessage: "<urgent> update", StaffNoticeSeverity: "critical", StaffNoticeDismissKey: "notice-key"}))
	var out bytes.Buffer
	if err := AdminStartForRequest(&out, request, "Dashboard", "", "dashboard"); err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"&lt;urgent&gt; update", `data-staff-notice="notice-key"`, `data-staff-notice-dismiss`, "hydro-staff-notice-dismissed"} {
		if !strings.Contains(out.String(), marker) {
			t.Errorf("staff notice shell missing %q", marker)
		}
	}
}

func TestAdminUserPermissionEditorUsesEditionCatalog(t *testing.T) {
	var out bytes.Buffer
	err := RenderAdminArea(&out, AdminAreaData{
		Area:        "users",
		CSRF:        "csrf-token",
		Permissions: []string{"article.read", "article.write"},
		Users:       []*auth.User{{ID: "user-1", Email: "editor@example.test", Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{"article.read": true}}},
	})
	if err != nil {
		t.Fatalf("RenderAdminArea(users): %v", err)
	}
	html := out.String()
	for _, marker := range []string{`name="perm_article.read"`, `name="perm_article.write"`, "article.read", "article.write"} {
		if !strings.Contains(html, marker) {
			t.Errorf("dynamic permission editor missing %q", marker)
		}
	}
	if strings.Contains(html, `name="perm_admin.users.manage"`) {
		t.Fatal("editor rendered a permission that the active Edition did not declare")
	}
}

func TestAdminUsersShowVerificationStateAndResendOnlyForUnverified(t *testing.T) {
	now := time.Now().UTC()
	var out bytes.Buffer
	err := RenderAdminArea(&out, AdminAreaData{Area: "users", CSRF: "csrf", EmailVerificationEnabled: true, Users: []*auth.User{
		{ID: "pending-user", Email: "pending@example.test", Role: auth.RoleEditor, Active: true},
		{ID: "verified-user", Email: "verified@example.test", Role: auth.RoleAdmin, Active: true, EmailVerifiedAt: &now},
	}})
	if err != nil {
		t.Fatal(err)
	}
	html := out.String()
	if !strings.Contains(html, "Doğrulama bekliyor") || !strings.Contains(html, "/auth/users/pending-user/email/verification/resend") || !strings.Contains(html, "Doğrulanmış") {
		t.Fatalf("verification state or resend action missing: %s", html)
	}
	verifiedRowStart := strings.Index(html, "verified@example.test")
	if verifiedRowStart < 0 || strings.Contains(html[verifiedRowStart:], "/auth/users/verified-user/email/verification/resend") {
		t.Fatalf("verified account offered a resend action: %s", html)
	}
	var disabled bytes.Buffer
	if err := RenderAdminArea(&disabled, AdminAreaData{Area: "users", Users: []*auth.User{{ID: "pending-user", Email: "pending@example.test"}}}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(disabled.String(), "/email/verification/resend") {
		t.Fatal("resend action displayed without mail delivery")
	}
}

func TestAdminUsersOfferPasswordResetLinkOnlyWhenConfigured(t *testing.T) {
	var out bytes.Buffer
	err := RenderAdminArea(&out, AdminAreaData{Area: "users", PasswordResetLinkEnabled: true, Users: []*auth.User{{ID: "active-user", Email: "active@example.test", Role: auth.RoleAdmin, Active: true}, {ID: "inactive-user", Email: "inactive@example.test", Active: false}}})
	if err != nil {
		t.Fatal(err)
	}
	html := out.String()
	if !strings.Contains(html, `data-password-reset-link="/auth/users/active-user/password-reset-link"`) || strings.Contains(html, `data-password-reset-link="/auth/users/inactive-user/password-reset-link"`) {
		t.Fatalf("reset-link controls incorrect: %s", html)
	}
	var disabled bytes.Buffer
	if err := RenderAdminArea(&disabled, AdminAreaData{Area: "users", Users: []*auth.User{{ID: "active-user", Active: true}}}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(disabled.String(), `data-password-reset-link="/auth/users/active-user/password-reset-link"`) {
		t.Fatal("reset link exposed without recovery configuration")
	}
}

func TestAdminSiteAreaLinksToSEOAndRedirectTools(t *testing.T) {
	var out bytes.Buffer
	if err := RenderAdminArea(&out, AdminAreaData{Area: "site", FontToolsEnabled: true}); err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`href="/admin/site/seo"`, `href="/admin/site/redirect-rules"`, `href="/admin/site/navigation/editor"`, `href="/admin/translations"`, `href="/admin/site/fonts"`, "SEO ayarları", "Yönlendirmeler", "Menüler", "Çeviriler", "Yazı tipi"} {
		if !strings.Contains(out.String(), marker) {
			t.Errorf("site admin area missing %q", marker)
		}
	}
}

func TestAdminPermissionCatalogEscapesEditionNames(t *testing.T) {
	var out bytes.Buffer
	if err := RenderAdminArea(&out, AdminAreaData{Area: "permissions", Permissions: []string{`article<script>`}}); err != nil {
		t.Fatalf("RenderAdminArea(permissions): %v", err)
	}
	if strings.Contains(out.String(), "article<script>") || !strings.Contains(out.String(), "article&lt;script&gt;") {
		t.Fatal("Edition permission catalog rendered an unescaped permission name")
	}
}

func TestRenderAdminAccountEnglishLocaleSelector(t *testing.T) {
	var out bytes.Buffer
	if err := RenderAdminArea(&out, AdminAreaData{Email: "admin@example.test",  CSRF: "csrf", Area: "account", Locale: "en", SoundAlertsEnabled: true}); err != nil {
		t.Fatalf("RenderAdminArea(account) error = %v", err)
	}
	markup := out.String()
	for _, want := range []string{`lang="en"`, "The operation could not be completed.", "Administrator profile", "Secure session", `action="/admin/account/locale"`, `name="locale"`, "English (EN)", `name="_csrf"`, `data-password-strength`, `data-password-meter`, `data-password-feedback`, `data-sound-alert-preference`, "Play a sound for new inquiries"} {
		if !strings.Contains(markup, want) {
			t.Fatalf("English account markup missing %q", want)
		}
	}
}

func TestAdminRealtimeChimeUsesOptInSettingAndFormEvent(t *testing.T) {
	for _, want := range []string{"hydro-inquiry-sound-v1", "forms%3Asubmission_created", `source.addEventListener('forms:submission_created',play)`, `window.AudioContext||window.webkitAudioContext`, `preference.checked=enabled()`} {
		if !strings.Contains(AdminRealtimeChimeJS, want) {
			t.Fatalf("realtime chime script missing %q", want)
		}
	}
}

func TestAdminShellIncludesDevicePreviewBehavior(t *testing.T) {
	for _, want := range []string{"data-device-preview-dialog", "data-device-preview-open", "data-device-preview-size", "data-device-preview-frame", "X-CSRF-Token", "credentials:'same-origin'"} {
		if !strings.Contains(AdminDevicePreviewJS, want) {
			t.Fatalf("device preview script missing %q", want)
		}
	}
	var out bytes.Buffer
	if err := AdminStart(&out, "Preview", "", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "data-device-preview-dialog") {
		t.Fatal("admin shell did not include device preview script")
	}
}

func TestAdminShellSPARouterAdoptsFragmentAssetsWithDocumentNonce(t *testing.T) {
	for _, want := range []string{"currentDocumentNonce", "adoptFragmentStyles", "Normalize while the fragment is still inert", "oldStyle.setAttribute('nonce', nonce)", "replaceChildren(...Array.from(newMain.childNodes))", "adoptFragmentStyles(newMain)", "searchGlobalBound"} {
		if !strings.Contains(AdminSPARouterJS, want) {
			t.Fatalf("AdminSPARouterJS missing fragment asset handling %q", want)
		}
	}
}

func TestAdminShellSpotlightFetchesProtectedCategorizedResults(t *testing.T) {
	for _, want := range []string{"/admin/api/search?q=", "new AbortController()", "İçerik ve kayıtlar", "item.badge", "item.href", "ArrowDown", "ArrowUp", "searchGlobalBound"} {
		if !strings.Contains(AdminSPARouterJS, want) {
			t.Fatalf("AdminSPARouterJS missing Spotlight behavior %q", want)
		}
	}
}

func TestAdminShellHardensBlankTargetLinks(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Güvenli bağlantılar", "", ""); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"target=\"_blank\"", "noopener noreferrer"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("admin shell missing blank-target hardening %q", want)
		}
	}
	if strings.Contains(out.String(), "/public/news/tr/html") {
		t.Fatal("admin preview link must not hard-code the legacy locale")
	}
	if !strings.Contains(out.String(), `href="/" target="_blank"`) {
		t.Fatal("admin preview link must use locale-neutral root")
	}
}

func TestAdminShellUsesDistinctBundledFavicon(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Admin", "", ""); err != nil {
		t.Fatal(err)
	}
	markup := out.String()
	headEnd := strings.Index(markup, "</head>")
	icon := strings.Index(markup, `rel="icon" href="data:image/svg+xml,`)
	if headEnd < 0 || icon < 0 || icon > headEnd {
		t.Fatalf("admin favicon was not emitted in the document head: %s", markup[:min(len(markup), 1200)])
	}
	if !strings.Contains(markup[icon:headEnd], "%23ef4444") {
		t.Fatal("admin favicon has no distinct badge mark")
	}
}

func TestAdminShellIncludesSessionExpiryWarning(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Session", "", ""); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/auth/me/session/extend", "Oturumunuz yakında sona erecek", "setAttribute('role','alertdialog')"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("admin shell session warning missing %q", want)
		}
	}
}

func TestAdminShellIncludesMobileNavigationContract(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Mobil navigasyon", "", ""); err != nil {
		t.Fatal(err)
	}
	markup := out.String()
	for _, want := range []string{
		`class="studio-mobile-menu-toggle"`,
		`aria-expanded="false"`,
		`studio-mobile-nav-open`,
		`pointerdown`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("admin shell missing mobile navigation contract %q", want)
		}
	}
}

func TestAdminShellIncludesAccessibleUXDialogsAndQuickActions(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Studio", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := AdminEnd(&out); err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`data-shortcuts-open`, `data-notifications-open`, `data-notification-count`, `/admin/notifications/unread`, `admin-shortcuts-dialog`, `admin-onboarding-dialog`} {
		if !strings.Contains(out.String(), marker) {
			t.Errorf("Admin Studio shell is missing %q", marker)
		}
	}
}

func TestBulkToolbarBuildsContextualActionSelector(t *testing.T) {
	script := adminUXScript()
	for _, marker := range []string{`data-bulk-choice`, `new Option(action.textContent,action.dataset.bulkAction)`, `actions[choice?.selectedIndex||0]`} {
		if !strings.Contains(script, marker) {
			t.Errorf("bulk toolbar script missing %q", marker)
		}
	}
}

func TestListViewSwitcherIsAccessibleAndUsesPersistedTarget(t *testing.T) {
	var out bytes.Buffer
	if err := ListViewSwitcher("content-list", "hydro-view-content").Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`role="group"`, `aria-label="Liste görünümü"`, `data-list-view-toggle="content-list"`, `data-storage-key="hydro-view-content"`, `data-list-view-choice="cards"`, `data-list-view-choice="table"`} {
		if !strings.Contains(out.String(), marker) {
			t.Errorf("list view switch missing %q", marker)
		}
	}
	for _, marker := range []string{`localStorage.setItem(toggle.dataset.storageKey||'hydro-view-default',value)`, `target.dataset.viewMode=mode`, `DOMContentLoaded`, `aria-pressed`} {
		if !strings.Contains(AdminListViewJS, marker) {
			t.Errorf("list view behavior missing %q", marker)
		}
	}
}

func TestAdminShellWriterPreservesInputLengthAfterRewrite(t *testing.T) {
	var out bytes.Buffer
	writer := adminShellWriter{Writer: &out}
	payload := []byte(`/public/news/tr/html`)
	n, err := writer.Write(payload)
	if err != nil || n != len(payload) {
		t.Fatalf("writer contract violated: n=%d len=%d err=%v", n, len(payload), err)
	}
	if out.String() != "/" {
		t.Fatalf("unexpected rewritten output: %q", out.String())
	}
}

func TestAdminShellDoesNotLoadThirdPartyFonts(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Test", "", ""); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"fonts.googleapis.com", "fonts.gstatic.com"} {
		if strings.Contains(out.String(), forbidden) {
			t.Fatalf("admin shell must not request third-party font host %q", forbidden)
		}
	}
}

func TestAdminShellShowsAuthenticatedUserInitials(t *testing.T) {
	user := &auth.User{Email: "arda.ozyildirim@example.test", Role: auth.RoleAdmin, Active: true}
	request := httptest.NewRequest("GET", "/admin", nil).WithContext(auth.WithUser(httptest.NewRequest("GET", "/admin", nil).Context(), user))
	var out bytes.Buffer
	if err := AdminStartForRequest(&out, request, "Dashboard", "", "dashboard"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `aria-label="AO"`) || !strings.Contains(out.String(), `ds-avatar--`) {
		t.Fatal("admin shell did not render the authenticated user's initials avatar")
	}
	if !strings.Contains(out.String(), `data-studio-user-scope="`) {
		t.Fatal("admin shell did not provide a user scoped draft recovery key")
	}
}

func TestAdminShellIncludesShadcnThemeModes(t *testing.T) {
	for _, want := range []string{
		// The CSS half is now compiled by the standalone Tailwind CLI
		// (ADR-0341), which prints attribute selectors without quotes
		// when they aren't required (`data-ui-theme=shadcn` instead of
		// `data-ui-theme="shadcn"`) - functionally identical CSS, so the
		// contract checks the unquoted form for selectors sourced from
		// AdminShadcnCSS.
		`data-ui-theme=shadcn`,
		`data-theme=dark`,
		`hydro-ui-theme`,
		`shadcn-light`,
		`shadcn-dark`,
		`updateVisualModeLabel`,
		`değiştirmek için tıklayın`,
	} {
		if !strings.Contains(AdminShadcnCSS+AdminSPARouterJS, want) {
			t.Fatalf("admin shell missing shadcn theme contract %q", want)
		}
	}
	var out bytes.Buffer
	if err := AdminStart(&out, "Tema", "", ""); err != nil {
		t.Fatalf("render admin shell: %v", err)
	}
	for _, want := range []string{`data-ui-theme=shadcn`, `hydro-ui-theme`, `shadcn-dark`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("rendered admin shell missing shadcn contract %q", want)
		}
	}
}

// TestWorkbenchFamilyIsFullyRemoved is a regression guard for ADR-0341's
// removal of the "workbench" admin visual family: it must never reappear in
// the rendered shell, the family-select allowlists, the migration/default
// fallback logic, or the account-page family picker. hydrogo took over
// workbench's former role as the default/fallback family.
func TestWorkbenchFamilyIsFullyRemoved(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Tema", "", ""); err != nil {
		t.Fatalf("render admin shell: %v", err)
	}
	if strings.Contains(strings.ToLower(out.String()), "workbench") {
		t.Fatal("rendered admin shell still mentions workbench - it must be fully removed (ADR-0341)")
	}
	var account bytes.Buffer
	if err := RenderAdminArea(&account, AdminAreaData{Email: "admin@example.test",  CSRF: "csrf", Area: "account"}); err != nil {
		t.Fatalf("render account area: %v", err)
	}
	if strings.Contains(strings.ToLower(account.String()), "workbench") {
		t.Fatal("account area's family selector still mentions workbench - it must be fully removed (ADR-0341)")
	}
}

// TestAdminShellIncludesHydroGoThemeMode verifies hydrogo (not workbench) is
// now the default/fallback family everywhere a fallback is needed: the
// pre-paint bootstrap script, the legacy-session migration script, and the
// family-select change handler and theme-toggle handler in AdminSPARouterJS.
func TestAdminShellIncludesHydroGoThemeMode(t *testing.T) {
	for _, want := range []string{
		`includes(parts[0])?parts[0]:'hydrogo'`,
		`setAttribute('data-ui-theme','hydrogo')`,
		`['hydrogo', 'shadcn', 'nova', 'aurora'].includes(familySelect.value) ? familySelect.value : 'hydrogo'`,
		`getAttribute('data-ui-theme') || 'hydrogo'`,
	} {
		if !strings.Contains(AdminSPARouterJS, want) {
			t.Fatalf("admin shell missing hydrogo default-fallback contract %q", want)
		}
	}
	var out bytes.Buffer
	if err := AdminStart(&out, "Tema", "", ""); err != nil {
		t.Fatalf("render admin shell: %v", err)
	}
	if !strings.Contains(out.String(), `data-ui-theme`) || !strings.Contains(out.String(), "hydro-ui-theme-v2") {
		t.Fatal("rendered admin shell missing pre-paint theme bootstrap")
	}
	if !strings.Contains(AdminThemeMigrationJS, "hydrogo-'+variant") {
		t.Fatal("AdminThemeMigrationJS no longer seeds the hydrogo default for legacy sessions")
	}
}

func TestAdminThemeMigrationDoesNotOverrideExplicitFamilyChoice(t *testing.T) {
	// Regression test: an earlier version of AdminThemeMigrationJS ran on
	// every page load and forced data-ui-theme back to "workbench" unless the
	// stored preference literally started with "workbench-", silently
	// reverting any explicit hydrogo/shadcn/nova selection on the next
	// refresh. The migration must now only seed a default when no versioned
	// preference exists at all, and must never call setAttribute itself.
	if strings.Contains(AdminThemeMigrationJS, "setAttribute") {
		t.Fatal("AdminThemeMigrationJS must not set data-ui-theme/data-theme itself; that would override an explicit family choice on every reload")
	}
	if strings.Contains(AdminThemeMigrationJS, "startsWith('workbench-')") {
		t.Fatal("AdminThemeMigrationJS must not re-fire for every non-workbench saved family")
	}
	if !strings.Contains(AdminThemeMigrationJS, "!localStorage.getItem(key)") {
		t.Fatal("AdminThemeMigrationJS must only seed a default when no versioned preference exists yet")
	}
}

func TestHydroAndShadcnModesShareSemanticMarkupContract(t *testing.T) {
	for _, selector := range []string{".studio-card", ".studio-form", ".button", ".studio-alert", ".studio-search"} {
		if !strings.Contains(AdminCSS, selector) {
			t.Fatalf("Hydro mode missing semantic selector %q", selector)
		}
		if !strings.Contains(AdminShadcnCSS, selector) {
			t.Fatalf("Shadcn mode missing shared semantic selector %q", selector)
		}
	}
	if !strings.Contains(AdminShadcnCSS, `html[data-ui-theme=shadcn]`) {
		t.Fatal("Shadcn styles are not scoped to the visual mode")
	}
}

func TestAdminShellSelfHostsInterFont(t *testing.T) {
	for _, want := range []string{
		`@font-face{font-family:'Inter'`,
		`url(/vendor/inter/inter-latin.woff2)`,
		`url(/vendor/inter/inter-latin-ext.woff2)`,
	} {
		if !strings.Contains(AdminInterFontCSS, want) {
			t.Fatalf("AdminInterFontCSS missing %q", want)
		}
	}
	// Must never point at an external font host: the project's CSP only
	// allows font-src 'self' data:, and ADR-0041 rules out a CDN dependency.
	for _, forbidden := range []string{"fonts.googleapis.com", "fonts.gstatic.com", "http://", "https://"} {
		if strings.Contains(AdminInterFontCSS, forbidden) {
			t.Fatalf("AdminInterFontCSS must be self-hosted, found forbidden reference %q", forbidden)
		}
	}
	var out bytes.Buffer
	if err := AdminStart(&out, "Tema", "", ""); err != nil {
		t.Fatalf("render admin shell: %v", err)
	}
	if !strings.Contains(out.String(), `/vendor/inter/inter-latin.woff2`) {
		t.Fatal("rendered admin shell missing self-hosted Inter font reference")
	}
}

func TestAdminShellIncludesNovaThemeMode(t *testing.T) {
	for _, want := range []string{
		`data-ui-theme=nova`,
		`nova-light`,
		`nova-dark`,
		`nova:'Nova'`,
	} {
		if !strings.Contains(AdminNovaCSS+AdminSPARouterJS+AdminFamilySelectCSS, want) {
			t.Fatalf("admin shell missing nova theme contract %q", want)
		}
	}
	var out bytes.Buffer
	if err := AdminStart(&out, "Tema", "", ""); err != nil {
		t.Fatalf("render admin shell: %v", err)
	}
	if !strings.Contains(out.String(), `data-ui-theme=nova`) {
		t.Fatal(`rendered admin shell missing data-ui-theme="nova" contract`)
	}

	// The family selector itself lives on the account settings page, not the
	// shared shell (moved out of the header per customer feedback).
	var account bytes.Buffer
	if err := RenderAdminArea(&account, AdminAreaData{Email: "admin@example.test",  CSRF: "csrf", Area: "account"}); err != nil {
		t.Fatalf("render account area: %v", err)
	}
	for _, want := range []string{
		`id="ui-family-select"`,
		`<option value="hydrogo">HydroGo</option>`,
		`<option value="shadcn">shadcn</option>`,
		`<option value="nova">Nova</option>`,
	} {
		if !strings.Contains(account.String(), want) {
			t.Fatalf("account area missing family selector contract %q", want)
		}
	}
}

func TestNovaModeSharesSemanticMarkupContract(t *testing.T) {
	novaCSS := AdminNovaCSS + AdminNovaDensityCSS
	for _, selector := range []string{".studio-card", ".studio-form", ".button", ".studio-alert", ".studio-search"} {
		if !strings.Contains(novaCSS, selector) {
			t.Fatalf("Nova mode missing shared semantic selector %q", selector)
		}
	}
	if !strings.Contains(AdminNovaCSS, `html[data-ui-theme=nova]`) {
		t.Fatal("Nova styles are not scoped to the visual mode")
	}
}

func TestAdminShellIncludesAuroraThemeMode(t *testing.T) {
	for _, want := range []string{
		`data-ui-theme="aurora"`,
		`aurora-light`,
		`aurora-dark`,
		`aurora:'Aurora'`,
	} {
		if !strings.Contains(AdminAuroraDashboardLayoutCSS+AdminSPARouterJS, want) {
			t.Fatalf("admin shell missing aurora theme contract %q", want)
		}
	}
	var out bytes.Buffer
	if err := AdminStart(&out, "Tema", "", ""); err != nil {
		t.Fatalf("render admin shell: %v", err)
	}
	if !strings.Contains(out.String(), `data-ui-theme="aurora"`) {
		t.Fatal(`rendered admin shell missing data-ui-theme="aurora" contract`)
	}

	var account bytes.Buffer
	if err := RenderAdminArea(&account, AdminAreaData{Email: "admin@example.test",  CSRF: "csrf", Area: "account"}); err != nil {
		t.Fatalf("render account area: %v", err)
	}
	if !strings.Contains(account.String(), `<option value="aurora">Aurora</option>`) {
		t.Fatal(`account area missing family selector contract for "aurora"`)
	}
}

// TestAuroraChromeSharesSemanticMarkupContract mirrors
// TestNovaModeSharesSemanticMarkupContract: Aurora re-skins the shared shell
// chrome (sidebar/topbar/cards/buttons/inputs) the same way every other
// family does, reusing the ADR-0053 semantic selectors. Only Aurora's
// dashboard body (covered by the next test) is the documented exception.
func TestAuroraChromeSharesSemanticMarkupContract(t *testing.T) {
	auroraCSS := AdminAuroraCSS + AdminAuroraDensityCSS
	for _, selector := range []string{".studio-card", ".studio-form", ".button", ".studio-alert", ".studio-search", ".studio-sidebar", ".studio-topbar"} {
		if !strings.Contains(auroraCSS, selector) {
			t.Fatalf("Aurora chrome missing shared semantic selector %q", selector)
		}
	}
	if !strings.Contains(AdminAuroraCSS, `html[data-ui-theme="aurora"]`) {
		t.Fatal("Aurora chrome styles are not scoped to the visual mode")
	}
}

// TestAuroraModeDoesNotShareSemanticMarkupContract documents the one
// intentional exception ADR-0339 carves out: unlike hydrogo/shadcn/workbench/
// nova, Aurora's dashboard is authored with Tailwind utility classes instead
// of the ADR-0053 semantic token contract, so it must NOT reuse the shared
// semantic selectors other families restyle.
func TestAuroraModeDoesNotShareSemanticMarkupContract(t *testing.T) {
	for _, selector := range []string{".studio-card", ".studio-form", ".button", ".studio-alert", ".studio-search"} {
		if strings.Contains(AdminAuroraDashboardLayoutCSS, selector) {
			t.Fatalf("Aurora layout CSS unexpectedly restyles shared semantic selector %q - it must stay isolated to its own compiled Tailwind stylesheet (ADR-0339)", selector)
		}
	}
	if !strings.Contains(AdminAuroraDashboardLayoutCSS, `html[data-ui-theme="aurora"]`) {
		t.Fatal("Aurora dashboard visibility toggle is not scoped to the visual mode")
	}
}

func TestAdminControlDoesNotRenderRuntimeErrorDetails(t *testing.T) {
	var out bytes.Buffer
	if err := RenderAdminControl(&out, AdminControlData{Email: "admin@example.test",  CSRF: "csrf"}); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	for _, forbidden := range []string{"err.message", "error.message"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("admin control exposes runtime error detail %q", forbidden)
		}
	}
}

func TestDeveloperTabsRendered(t *testing.T) {
	var out bytes.Buffer
	err := RenderAdminArea(&out, AdminAreaData{
		Email: "admin@example.test",
		Nonce: "nonce-dev",
		CSRF:  "csrf-dev",
		Area:  "developer",
	})
	if err != nil {
		t.Fatalf("RenderAdminArea(developer) error = %v", err)
	}
	rendered := out.String()
	for _, want := range []string{"studio-tabs-nav", "/admin/developer", "/admin/developer/resources", "/admin/control", "/admin/security", "is-active"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("developer page missing tab navigation element %q", want)
		}
	}
	if strings.Contains(rendered, "/admin/developer/components") {
		t.Fatal("developer tabs still contain component showcase")
	}

	// Verify security page also renders developer tabs
	var secOut bytes.Buffer
	if err := RenderAdminArea(&secOut, AdminAreaData{Email: "admin@example.test",  CSRF: "csrf", Area: "security"}); err != nil {
		t.Fatalf("RenderAdminArea(security) error = %v", err)
	}
	if !strings.Contains(secOut.String(), "studio-tabs-nav") || !strings.Contains(secOut.String(), `href="/admin/security" class="studio-tab-item is-active"`) {
		t.Fatal("security page missing active developer tabs navigation")
	}

	// Verify control page also renders developer tabs
	var ctrlOut bytes.Buffer
	if err := RenderAdminControl(&ctrlOut, AdminControlData{Email: "admin@example.test",  CSRF: "csrf", Active: "control"}); err != nil {
		t.Fatalf("RenderAdminControl error = %v", err)
	}
	for _, want := range []string{`data-invite-share-dialog`, `data-invite-share-message`, `data-invite-share-copy`, `dialog.showModal()`, `clipboard.writeText`} {
		if !strings.Contains(ctrlOut.String(), want) {
			t.Fatalf("admin invite share UI missing %q", want)
		}
	}
	if !strings.Contains(ctrlOut.String(), "studio-tabs-nav") || !strings.Contains(ctrlOut.String(), `href="/admin/control" class="studio-tab-item is-active"`) {
		t.Fatal("control page missing active developer tabs navigation")
	}
}
