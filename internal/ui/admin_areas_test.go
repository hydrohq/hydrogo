package ui

import (
	"bytes"
	"strings"
	"testing"

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
