package ui

import (
	"bytes"
	"strings"
	"testing"

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
	for _, want := range []string{"Kaynak tüketimi", "Sistem kaynakları", "Heap kullanımı", "Toplam istek", "nonce-test", "/admin/developer/resources"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("rendered resource page missing %q", want)
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

func TestAdminPermissionCatalogEscapesEditionNames(t *testing.T) {
	var out bytes.Buffer
	if err := RenderAdminArea(&out, AdminAreaData{Area: "permissions", Permissions: []string{`article<script>`}}); err != nil {
		t.Fatalf("RenderAdminArea(permissions): %v", err)
	}
	if strings.Contains(out.String(), "<script>") || !strings.Contains(out.String(), "&lt;script&gt;") {
		t.Fatal("Edition permission catalog rendered an unescaped permission name")
	}
}

func TestRenderAdminAccountEnglishLocaleSelector(t *testing.T) {
	var out bytes.Buffer
	if err := RenderAdminArea(&out, AdminAreaData{Email: "admin@example.test", Nonce: "nonce", CSRF: "csrf", Area: "account", Locale: "en"}); err != nil {
		t.Fatalf("RenderAdminArea(account) error = %v", err)
	}
	markup := out.String()
	for _, want := range []string{`lang="en"`, "The operation could not be completed.", "Administrator profile", "Secure session", `action="/admin/account/locale"`, `name="locale"`, "English (EN)", `name="_csrf"`} {
		if !strings.Contains(markup, want) {
			t.Fatalf("English account markup missing %q", want)
		}
	}
}

func TestAdminShellSPARouterAdoptsFragmentAssetsWithDocumentNonce(t *testing.T) {
	for _, want := range []string{"currentDocumentNonce", "adoptFragmentStyles", "Normalize while the fragment is still inert", "oldStyle.setAttribute('nonce', nonce)", "replaceChildren(...Array.from(newMain.childNodes))", "adoptFragmentStyles(newMain)", "searchGlobalBound"} {
		if !strings.Contains(AdminSPARouterJS, want) {
			t.Fatalf("AdminSPARouterJS missing fragment asset handling %q", want)
		}
	}
}

func TestAdminShellHardensBlankTargetLinks(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Güvenli bağlantılar", "", "nonce-test", ""); err != nil {
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

func TestAdminShellIncludesMobileNavigationContract(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Mobil navigasyon", "", "nonce-test", ""); err != nil {
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
	if err := AdminStart(&out, "Test", "", "nonce-test", ""); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"fonts.googleapis.com", "fonts.gstatic.com"} {
		if strings.Contains(out.String(), forbidden) {
			t.Fatalf("admin shell must not request third-party font host %q", forbidden)
		}
	}
}

func TestAdminShellIncludesShadcnThemeModes(t *testing.T) {
	for _, want := range []string{
		`data-ui-theme="shadcn"`,
		`data-theme="dark"`,
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
	if err := AdminStart(&out, "Tema", "", "nonce-test", ""); err != nil {
		t.Fatalf("render admin shell: %v", err)
	}
	for _, want := range []string{`data-ui-theme="shadcn"`, `hydro-ui-theme`, `shadcn-dark`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("rendered admin shell missing shadcn contract %q", want)
		}
	}
}

func TestAdminShellIncludesWorkbenchThemeMode(t *testing.T) {
	for _, want := range []string{"data-ui-theme=\"workbench\"", "workbench-light", "workbench-dark", "Workbench"} {
		if !strings.Contains(AdminWorkbenchCSS+AdminSPARouterJS, want) {
			t.Fatalf("admin shell missing workbench theme contract %q", want)
		}
	}
	var out bytes.Buffer
	if err := AdminStart(&out, "Tema", "", "nonce-test", ""); err != nil {
		t.Fatalf("render admin shell: %v", err)
	}
	if !strings.Contains(out.String(), `data-ui-theme="workbench"`) {
		t.Fatal("rendered admin shell missing workbench contract")
	}
	if !strings.Contains(out.String(), "hydro-ui-theme-v2") || !strings.Contains(out.String(), "startsWith('workbench-')") {
		t.Fatal("rendered admin shell missing pre-paint Workbench migration")
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
	if !strings.Contains(AdminShadcnCSS, `html[data-ui-theme="shadcn"]`) {
		t.Fatal("Shadcn styles are not scoped to the visual mode")
	}
}

func TestAdminControlDoesNotRenderRuntimeErrorDetails(t *testing.T) {
	var out bytes.Buffer
	if err := RenderAdminControl(&out, AdminControlData{Email: "admin@example.test", Nonce: "nonce", CSRF: "csrf"}); err != nil {
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
	if err := RenderAdminArea(&secOut, AdminAreaData{Email: "admin@example.test", Nonce: "nonce", CSRF: "csrf", Area: "security"}); err != nil {
		t.Fatalf("RenderAdminArea(security) error = %v", err)
	}
	if !strings.Contains(secOut.String(), "studio-tabs-nav") || !strings.Contains(secOut.String(), `href="/admin/security" class="studio-tab-item is-active"`) {
		t.Fatal("security page missing active developer tabs navigation")
	}

	// Verify control page also renders developer tabs
	var ctrlOut bytes.Buffer
	if err := RenderAdminControl(&ctrlOut, AdminControlData{Email: "admin@example.test", Nonce: "nonce", CSRF: "csrf", Active: "control"}); err != nil {
		t.Fatalf("RenderAdminControl error = %v", err)
	}
	if !strings.Contains(ctrlOut.String(), "studio-tabs-nav") || !strings.Contains(ctrlOut.String(), `href="/admin/control" class="studio-tab-item is-active"`) {
		t.Fatal("control page missing active developer tabs navigation")
	}
}
