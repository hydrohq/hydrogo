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

func TestAdminShellPageCreateActionFollowsContentWritePermission(t *testing.T) {
	tests := []struct {
		name    string
		user    *auth.User
		visible bool
	}{
		{name: "admin default", user: &auth.User{Role: auth.RoleAdmin, Active: true}, visible: true},
		{name: "editor default", user: &auth.User{Role: auth.RoleEditor, Active: true}, visible: true},
		{name: "viewer default", user: &auth.User{Role: auth.RoleViewer, Active: true}},
		{name: "viewer allow override", user: &auth.User{Role: auth.RoleViewer, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: true}}, visible: true},
		{name: "editor deny override", user: &auth.User{Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: false}}},
		{name: "admin deny override", user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: false}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/admin/pages", nil)
			request = request.WithContext(auth.WithUser(request.Context(), tc.user))
			var out bytes.Buffer
			if err := AdminStartForRequest(&out, request, "Sayfalar", "", "pages"); err != nil {
				t.Fatal(err)
			}
			got := strings.Contains(out.String(), `href="/admin/pages/new"`)
			if got != tc.visible {
				t.Fatalf("page create action visible = %t, want %t", got, tc.visible)
			}
		})
	}
}

func TestAdminShellHasNoCommerceRemnants(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Dashboard", "", "dashboard"); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{"shell": out.String(), "studio.js": adminAsset(t, "js/studio.js")} {
		for _, removed := range []string{"/admin/commerce", "data-commerce-enabled", "commerceEnabled"} {
			if strings.Contains(source, removed) {
				t.Errorf("%s still mentions the removed commerce module: %q", name, removed)
			}
		}
	}
}

func TestAdminShellNavigationFollowsEffectivePermissions(t *testing.T) {
	tests := []struct {
		name string
		user *auth.User
		want []string
		omit []string
	}{
		{
			name: "viewer defaults",
			user: &auth.User{Role: auth.RoleViewer, Active: true},
			want: []string{`href="/admin/pages"`, `href="/admin/news"`},
			omit: []string{`href="/admin/users"`, `href="/admin/reports"`, `href="/admin/members"`, `href="/admin/media"`},
		},
		{
			name: "viewer member override",
			user: &auth.User{Role: auth.RoleViewer, Active: true, PermissionOverrides: map[string]bool{auth.PermissionMembersManage: true}},
			want: []string{`href="/admin/members"`},
			omit: []string{`href="/admin/users"`},
		},
		{
			name: "editor member override",
			user: &auth.User{Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionMembersManage: true}},
			want: []string{`href="/admin/members"`},
			omit: []string{`href="/admin/users"`, `href="/admin/reports"`},
		},
		{
			name: "editor site management override",
			user: &auth.User{Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}},
			want: []string{`href="/admin/site"`},
			omit: []string{`href="/admin/control"`},
		},
		{
			name: "editor media denied",
			user: &auth.User{Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionMediaManage: false}},
			want: []string{`href="/admin/pages"`, `href="/admin/news"`},
			omit: []string{`href="/admin/media"`, `href="/admin/reports"`},
		},
		{
			name: "admin content read denied",
			user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentRead: false}},
			omit: []string{`href="/admin/pages"`, `href="/admin/news"`},
		},
		{
			name: "admin commerce denied",
			user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionCommerceRead: false, auth.PermissionCommerceWrite: false}},
			omit: []string{`href="/admin/commerce/products"`, `href="/admin/commerce/orders"`, `href="/admin/commerce/products/new"`},
		},
		{
			name: "admin user management denied",
			user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionUsersManage: false}},
			omit: []string{`href="/admin/users"`, `href="/admin/permissions"`},
		},
		{
			name: "admin users permission independent of members",
			user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionMembersManage: false}},
			want: []string{`href="/admin/users"`},
			omit: []string{`href="/admin/members"`},
		},
		{
			name: "admin operations denied",
			user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionOperations: false}},
			omit: []string{`href="/admin/operations"`},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/admin", nil)
			request = request.WithContext(auth.WithUser(request.Context(), tc.user))
			var out bytes.Buffer
			if err := AdminStartForRequest(&out, request, "Dashboard", "", "dashboard"); err != nil {
				t.Fatal(err)
			}
			html := inlineAssets.ReplaceAllString(out.String(), "")
			for _, marker := range tc.want {
				if !strings.Contains(html, marker) {
					t.Errorf("navigation missing %q", marker)
				}
			}
			for _, marker := range tc.omit {
				if strings.Contains(html, marker) {
					t.Errorf("navigation still shows denied link %q", marker)
				}
			}
		})
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
	for _, marker := range []string{"&lt;urgent&gt; update", `data-staff-notice="notice-key"`, `data-staff-notice-dismiss`, `/admin-assets/js/staff-notice.js`} {
		if !strings.Contains(out.String(), marker) {
			t.Errorf("staff notice shell missing %q", marker)
		}
	}
	if !strings.Contains(adminAsset(t, "js/staff-notice.js"), "hydro-staff-notice-dismissed") {
		t.Error("staff notice dismissal is not remembered")
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

func TestRenderAdminAreaUsesRequestRoleAndPermissions(t *testing.T) {
	request := httptest.NewRequest("GET", "/admin/site", nil)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}}))
	var out bytes.Buffer
	if err := RenderAdminArea(&out, AdminAreaData{Request: request, Area: "site", Locale: "tr"}); err != nil {
		t.Fatal(err)
	}
	markup := inlineAssets.ReplaceAllString(out.String(), "")
	for _, marker := range []string{`data-studio-role="editor"`, `href="/admin/site"`, "Site ayarları"} {
		if !strings.Contains(markup, marker) {
			t.Errorf("permission-aware site page missing %q", marker)
		}
	}
	for _, marker := range []string{`href="/admin/users"`, `href="/admin/operations"`, `href="/admin/control"`} {
		if strings.Contains(markup, marker) {
			t.Errorf("editor site page leaked privileged navigation %q", marker)
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
	if err := RenderAdminArea(&out, AdminAreaData{Email: "admin@example.test", CSRF: "csrf", Area: "account", Locale: "en", SoundAlertsEnabled: true, MFAAvailable: true, MFAStatusLoaded: true}); err != nil {
		t.Fatalf("RenderAdminArea(account) error = %v", err)
	}
	markup := out.String()
	for _, want := range []string{`lang="en"`, `src="/admin-assets/js/english-ui.js"`, "Administrator profile", "Secure session", `action="/admin/account/locale"`, `name="locale"`, "English (EN)", `name="_csrf"`, `data-password-strength`, `data-password-meter`, `data-password-feedback`, `data-sound-alert-preference`, "Play a sound for new inquiries", `data-mfa-settings`, "Set up authenticator"} {
		if !strings.Contains(markup, want) {
			t.Fatalf("English account markup missing %q", want)
		}
	}
}

func TestAdminTourReadsRoleAfterDOMIsReady(t *testing.T) {
	tour := adminAsset(t, "js/tour.js")
	roleRead := strings.Index(tour, "const role = document.body?.dataset.studioRole")
	initializer := strings.Index(tour, "function initTour()")
	if roleRead < 0 || initializer < 0 || roleRead < initializer {
		t.Fatal("admin onboarding tour must read the body role from its DOM-ready initializer")
	}
}

func TestAdminRealtimeChimeUsesOptInSettingAndFormEvent(t *testing.T) {
	for _, want := range []string{"hydro-inquiry-sound-v1", "forms%3Asubmission_created", `source.addEventListener('forms:submission_created',play)`, `window.AudioContext||window.webkitAudioContext`, `preference.checked=enabled()`} {
		if !strings.Contains(adminAsset(t, "js/realtime-chime.js"), want) {
			t.Fatalf("realtime chime script missing %q", want)
		}
	}
}

func TestAdminShellIncludesDevicePreviewBehavior(t *testing.T) {
	for _, want := range []string{"data-device-preview-dialog", "data-device-preview-open", "data-device-preview-size", "data-device-preview-frame", "X-CSRF-Token", "credentials:'same-origin'"} {
		if !strings.Contains(adminAsset(t, "js/device-preview.js"), want) {
			t.Fatalf("device preview script missing %q", want)
		}
	}
	var out bytes.Buffer
	if err := AdminStart(&out, "Preview", "", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `src="/admin-assets/js/device-preview.js"`) {
		t.Fatal("admin shell did not include device preview script")
	}
}

func TestAdminShellSPARouterAdoptsFragmentAssetsWithDocumentNonce(t *testing.T) {
	for _, want := range []string{"currentDocumentNonce", "adoptFragmentStyles", "Normalize while the fragment is still inert", "oldStyle.setAttribute('nonce', nonce)", "replaceChildren(...Array.from(newMain.childNodes))", "adoptFragmentStyles(newMain)", "searchGlobalBound"} {
		if !strings.Contains(adminAsset(t, "js/studio.js"), want) {
			t.Fatalf("studio.js missing fragment asset handling %q", want)
		}
	}
}

func TestAdminShellSpotlightFetchesProtectedCategorizedResults(t *testing.T) {
	for _, want := range []string{"/admin/api/search?q=", "new AbortController()", "İçerik ve kayıtlar", "item.badge", "item.href", "ArrowDown", "ArrowUp", "searchGlobalBound"} {
		if !strings.Contains(adminAsset(t, "js/studio.js"), want) {
			t.Fatalf("studio.js missing Spotlight behavior %q", want)
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
	if !strings.Contains(out.String(), `<script defer src="/admin-assets/js/session-warning.js"></script>`) {
		t.Fatal("admin shell does not load the session expiry warning")
	}
	for _, want := range []string{"/auth/me/session/extend", "Oturumunuz yakında sona erecek", "setAttribute('role','alertdialog')"} {
		if !strings.Contains(adminAsset(t, "js/session-warning.js"), want) {
			t.Fatalf("admin session warning missing %q", want)
		}
	}
}

func TestAdminSessionWarningUsesAdminLocale(t *testing.T) {
	script := adminAsset(t, "js/session-warning.js")
	if !strings.Contains(script, "document.documentElement.lang==='en'") {
		t.Fatal("session warning does not follow the document language")
	}
	if !strings.Contains(script, "document.addEventListener('DOMContentLoaded',install,{once:true})") {
		t.Fatal("session warning does not defer its body-dependent startup until DOMContentLoaded")
	}
	for _, want := range []string{
		"Your session will expire soon", "Extend your session to keep working.", "Extend session", "Try again",
		"Oturumunuz yakında sona erecek", "Çalışmanızı sürdürmek için oturumunuzu uzatabilirsiniz.", "Oturumu uzat", "Tekrar dene",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("session warning missing %q", want)
		}
	}
	for _, locale := range []string{"en", "tr"} {
		var out bytes.Buffer
		if err := AdminStartLocale(&out, "Session", "", "", locale); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), `<html lang="`+locale+`"`) {
			t.Errorf("admin shell does not declare document language %q", locale)
		}
	}
}

func TestAdminShellIncludesMobileNavigationContract(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Mobil navigasyon", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := AdminEnd(&out); err != nil {
		t.Fatal(err)
	}
	markup := out.String()
	for _, want := range []string{
		`class="studio-mobile-menu-toggle" aria-label="Menüyü aç" aria-expanded="false" x-data="navToggle"`,
		`x-bind:aria-expanded="expanded"`,
		`class="studio-mobile-backdrop" aria-label="Menüyü kapat" x-data="navBackdrop"`,
		`class="studio-sidebar" x-data="studioSidebar"`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("admin shell missing mobile navigation contract %q", want)
		}
	}
	shellJS := adminAsset(t, "shell.js")
	for _, want := range []string{`"studio-mobile-nav-open"`, `event.key !== "Escape"`, "MOBILE_BREAKPOINT"} {
		if !strings.Contains(shellJS, want) {
			t.Fatalf("shell.js missing mobile navigation behaviour %q", want)
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
	for _, marker := range []string{`data-notifications-open`, `data-notification-count`, `x-data="notifications"`, `admin-shortcuts-dialog`, `admin-onboarding-dialog`} {
		if !strings.Contains(out.String(), marker) {
			t.Errorf("Admin Studio shell is missing %q", marker)
		}
	}
	if !strings.Contains(adminAsset(t, "shell.js"), "/admin/notifications/unread") {
		t.Error("notification panel does not load the unread count")
	}
	if !strings.Contains(adminAsset(t, "js/ux.js"), "[data-shortcuts-open]") {
		t.Error("shortcut dialog trigger is not wired")
	}
}

func TestAdminQuillImageURLFallbackUsesAdminLanguage(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStartLocale(&out, "Studio", "", "", "en"); err != nil {
		t.Fatal(err)
	}
	html := out.String() + adminAsset(t, "js/studio.js")
	for _, want := range []string{`<html lang="en"`, `document.documentElement.lang === 'en' ? 'Enter an image URL:' : 'Görsel URL adresini girin:'`} {
		if !strings.Contains(html, want) {
			t.Errorf("English Admin shell is missing locale-aware Quill prompt %q", want)
		}
	}
}

func TestAdminUndoToastLabelUsesDocumentLanguage(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStartLocale(&out, "Studio", "", "", "en"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`document.documentElement.lang === 'en' ? 'Undo' : 'Geri Al'`, `studio-toast-undo`} {
		if !strings.Contains(adminAsset(t, "js/studio.js"), want) {
			t.Errorf("English Admin shell is missing localized undo toast contract %q", want)
		}
	}
}

func TestAdminVisualControlsFollowAdminLanguage(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStartLocale(&out, "Studio", "", "", "en"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `aria-label="Change theme"`) || !strings.Contains(out.String(), `x-bind:aria-label="label"`) {
		t.Error("English theme toggle is missing its label binding")
	}
	for _, want := range []string{"; click to change", "; değiştirmek için tıklayın", " theme applied.", " tema uygulandı."} {
		if !strings.Contains(adminAsset(t, "shell.js"), want) {
			t.Errorf("Admin visual controls are missing locale behavior %q", want)
		}
	}
}

func TestBulkToolbarBuildsContextualActionSelector(t *testing.T) {
	script := adminAsset(t, "js/ux.js")
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
		if !strings.Contains(adminAsset(t, "js/list-view.js"), marker) {
			t.Errorf("list view behavior missing %q", marker)
		}
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

func TestAdminShellLightDarkToggleContract(t *testing.T) {
	shellJS := adminAsset(t, "shell.js")
	for _, want := range []string{
		`Alpine.data("themeToggle"`,
		`değiştirmek için tıklayın`,
		`localStorage.setItem("hydro-theme", variant)`,
		`setAttribute("data-theme", variant)`,
	} {
		if !strings.Contains(shellJS, want) {
			t.Fatalf("admin shell missing light/dark toggle contract %q", want)
		}
	}
	if !strings.Contains(AdminHydroGoCSS, `html[data-ui-theme=hydrogo][data-theme=dark]`) {
		t.Fatal("hydrogo styles lost their dark-mode palette")
	}
	var out bytes.Buffer
	if err := AdminStart(&out, "Tema", "", ""); err != nil {
		t.Fatalf("render admin shell: %v", err)
	}
	if !strings.Contains(out.String(), `id="theme-toggle-btn"`) || !strings.Contains(out.String(), `x-data="themeToggle"`) {
		t.Fatal("rendered admin shell missing the light/dark toggle button")
	}
}

// TestRemovedVisualFamiliesStayRemoved guards the consolidation of Studio on
// the single hydrogo visual family: the former workbench, shadcn, nova and
// aurora families and the family picker must not reappear in the rendered
// shell or on the account page.
func TestRemovedVisualFamiliesStayRemoved(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Tema", "", ""); err != nil {
		t.Fatalf("render admin shell: %v", err)
	}
	var account bytes.Buffer
	if err := RenderAdminArea(&account, AdminAreaData{Email: "admin@example.test", CSRF: "csrf", Area: "account"}); err != nil {
		t.Fatalf("render account area: %v", err)
	}
	for name, markup := range map[string]string{"admin shell": out.String(), "account area": account.String()} {
		lower := strings.ToLower(markup)
		for _, forbidden := range []string{"workbench", "shadcn", "aurora", "data-ui-theme=nova", "nova-", "ui-family-select"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s still mentions removed visual family marker %q", name, forbidden)
			}
		}
	}
}

// TestAdminShellIncludesHydroGoThemeMode verifies the pre-paint bootstrap
// always pins the hydrogo family and resolves light/dark from the saved
// preference or the OS setting.
func TestAdminShellIncludesHydroGoThemeMode(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Tema", "", ""); err != nil {
		t.Fatalf("render admin shell: %v", err)
	}
	markup := out.String()
	head := markup[:strings.Index(markup, "</head>")]
	for _, want := range []string{
		`<html lang="tr" data-ui-theme="hydrogo">`,
		`<script>(()=>{const root=document.documentElement;`,
		`setAttribute('data-ui-theme','hydrogo')`,
		`localStorage.getItem('hydro-theme')`,
		`prefers-color-scheme: dark`,
		`setAttribute('data-theme',variant)`,
	} {
		if !strings.Contains(head, want) {
			t.Fatalf("pre-paint theme bootstrap missing contract %q", want)
		}
	}
	if strings.Index(head, "data-theme',variant") > strings.Index(head, "<link rel=\"stylesheet\"") {
		t.Fatal("theme bootstrap must run before the stylesheets load")
	}
}

func TestAdminCSSKeepsSemanticMarkupContract(t *testing.T) {
	for _, selector := range []string{".studio-card", ".studio-form", ".button", ".studio-alert", ".studio-search"} {
		if !strings.Contains(AdminCSS, selector) {
			t.Fatalf("Hydro mode missing semantic selector %q", selector)
		}
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
	if !strings.Contains(out.String(), `<link rel="stylesheet" href="/admin-assets/css/fonts.css">`) || !strings.Contains(AdminInterFontCSS, `/vendor/inter/inter-latin.woff2`) {
		t.Fatal("rendered admin shell missing self-hosted Inter font reference")
	}
}

func TestAdminControlDoesNotRenderRuntimeErrorDetails(t *testing.T) {
	var out bytes.Buffer
	if err := RenderAdminControl(&out, AdminControlData{Email: "admin@example.test", CSRF: "csrf"}); err != nil {
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
	if err := RenderAdminArea(&secOut, AdminAreaData{Email: "admin@example.test", CSRF: "csrf", Area: "security"}); err != nil {
		t.Fatalf("RenderAdminArea(security) error = %v", err)
	}
	if !strings.Contains(secOut.String(), "studio-tabs-nav") || !strings.Contains(secOut.String(), `href="/admin/security" class="studio-tab-item is-active"`) {
		t.Fatal("security page missing active developer tabs navigation")
	}

	// Verify control page also renders developer tabs
	var ctrlOut bytes.Buffer
	if err := RenderAdminControl(&ctrlOut, AdminControlData{Email: "admin@example.test", CSRF: "csrf", Active: "control"}); err != nil {
		t.Fatalf("RenderAdminControl error = %v", err)
	}
	for _, want := range []string{`data-invite-share-dialog`, `data-invite-share-message`, `data-invite-share-copy`, `id="invite-form"`} {
		if !strings.Contains(ctrlOut.String(), want) {
			t.Fatalf("admin invite share UI missing %q", want)
		}
	}
	// The invite flow runs from a same-origin script, not an inline block.
	for _, want := range []string{`dialog.showModal()`, `clipboard.writeText`, `"invite-form"`, `"/admin/auth/invites"`} {
		if !strings.Contains(adminAsset(t, "js/admin-control.js"), want) {
			t.Fatalf("admin control script missing %q", want)
		}
	}
	if !strings.Contains(ctrlOut.String(), "studio-tabs-nav") || !strings.Contains(ctrlOut.String(), `href="/admin/control" class="studio-tab-item is-active"`) {
		t.Fatal("control page missing active developer tabs navigation")
	}
}
