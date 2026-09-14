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
