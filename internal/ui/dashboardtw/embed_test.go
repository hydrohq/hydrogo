package dashboardtw

import (
	"strings"
	"testing"
)

// TestEmbeddedCSSIsPresent is the contract test required by ADR-0338/0340:
// the compiled, Tailwind-prefixed stylesheet must ship inside the binary and
// must not be an empty placeholder.
func TestEmbeddedCSSIsPresent(t *testing.T) {
	if strings.TrimSpace(CSSText) == "" {
		t.Fatal("embedded dashboard CSS is empty - run scripts/build-dashboard-tailwind.ps1 to regenerate dist/dashboard.css")
	}
}

// TestDashboardMarkupClassesAreCompiled guards against the exact bug class
// documented at internal/ui/admin_shell.go:2747-2756 (Tailwind utility
// classes used with zero matching compiled CSS). A representative subset of
// the "tw:"-prefixed utility classes used by adminWorkbenchDashboardTemplate
// and adminNovaDashboardTemplate must have a real rule in the compiled,
// embedded stylesheet.
func TestDashboardMarkupClassesAreCompiled(t *testing.T) {
	wantSelectors := []string{
		`.tw\:grid{`, `.tw\:flex{`, `.tw\:gap-\[16px\]{`, `.tw\:rounded-\[var\(--radius-lg\)\]{`,
		`.tw\:bg-\[var\(--surface\)\]{`, `.tw\:shadow-\[var\(--card-highlight\)\]{`,
		`.tw\:grid-cols-\[repeat\(4\,minmax\(0\,1fr\)\)\]{`, `.tw\:text-\[var\(--ink\)\]{`,
	}
	for _, sel := range wantSelectors {
		if !strings.Contains(CSSText, sel) {
			t.Errorf("compiled dashboard CSS is missing selector %q used by the workbench/nova dashboard templates - class is dead/unmatched", sel)
		}
	}
}

// TestCSSExcludesPreflight is the collision guard ADR-0338/0339/0340
// document: Tailwind's global reset layer must never ship here, since it
// would apply to every element on the admin shell page, not just the
// dashboard markup.
func TestCSSExcludesPreflight(t *testing.T) {
	if strings.Contains(CSSText, "box-sizing:border-box}html{") {
		t.Fatal("compiled dashboard CSS appears to include Tailwind's preflight reset layer - it must be built from tailwindcss/theme.css + tailwindcss/utilities.css only (see internal/ui/dashboardtw/input.css)")
	}
}

// TestCSSDefinesNoUnprefixedSemanticToken is the second collision guard:
// this package's compiled output may *reference* ADR-0053 semantic tokens
// via var(--radius-lg) etc. (expected, that's how it re-skins per family),
// but it must never *define* one (e.g. "--radius-lg:0.5rem"), which would
// collide with the ADR-0053 contract.
func TestCSSDefinesNoUnprefixedSemanticToken(t *testing.T) {
	for _, def := range []string{"--radius-lg:", "--radius-md:", "--radius-sm:", "--shadow-sm:", "--shadow-md:"} {
		if strings.Contains(CSSText, def) {
			t.Fatalf("compiled dashboard CSS defines %q - this must stay a var() reference, never a definition, or it collides with the ADR-0053 semantic token contract", def)
		}
	}
}
