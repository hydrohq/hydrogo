package hydrogoui

import (
	"regexp"
	"strings"
	"testing"
)

// TestEmbeddedHydroGoCSSIsPresent is the contract test required by
// ADR-0341: the compiled,
// Tailwind-prefixed stylesheet must ship inside the binary and must not be
// an empty placeholder.
func TestEmbeddedHydroGoCSSIsPresent(t *testing.T) {
	if strings.TrimSpace(CSSText) == "" {
		t.Fatal("embedded hydrogo dashboard CSS is empty - run scripts/build-hydrogoui-tailwind.ps1 to regenerate static/admin/css/hydrogo-dashboard.css")
	}
}

// TestDashboardClassesAreCompiled guards against Tailwind utility classes
// used with zero matching compiled CSS.
func TestDashboardClassesAreCompiled(t *testing.T) {
	wantSelectors := []string{
		`.tw\:flex{`, `.tw\:grid{`, `.tw\:rounded-2xl{`, `.tw\:font-bold{`,
		`.tw\:tabular-nums{`, `.tw\:border{`,
	}
	for _, sel := range wantSelectors {
		if !strings.Contains(CSSText, sel) {
			t.Errorf("compiled hydrogo CSS is missing selector %q used by dashboard.tmpl.html - class is dead/unmatched", sel)
		}
	}
}

// TestHydroGoThemeVariablesAreNamespaced is the collision guard
// ADR-0339/0341 document: Tailwind's generated theme CSS custom properties
// must all be prefixed with "--tw-" so they cannot collide with an
// ADR-0053 semantic token name when this stylesheet is concatenated into
// the same page as AdminCSS.
var unprefixedTokenDefinition = regexp.MustCompile(`--(?:radius|shadow)-[a-zA-Z0-9]+:`)

func TestHydroGoThemeVariablesAreNamespaced(t *testing.T) {
	if loc := unprefixedTokenDefinition.FindString(CSSText); loc != "" {
		t.Fatalf("compiled hydrogo CSS defines an unprefixed theme variable (%q) that could collide with the ADR-0053 semantic token contract", loc)
	}
	if !strings.Contains(CSSText, "--tw-") {
		t.Fatal("compiled hydrogo CSS does not use the expected Tailwind \"--tw-\" variable prefix")
	}
}

// literalColorUtility matches a color-bearing Tailwind utility given a
// literal hex value instead of a var(--token) reference. Every
// color-bearing class in the markup must reference an ADR-0053 token (or
// the hydrogo-specific override in admin_hydrogo.go) via var(...) so it
// re-skins correctly across light/dark.
var literalColorUtility = regexp.MustCompile(`tw:(?:bg|text|border|shadow)-\[#[0-9a-fA-F]`)

func TestDashboardMarkupUsesTokensNotLiteralColors(t *testing.T) {
	b, err := markupFS.ReadFile("dashboard.tmpl.html")
	if err != nil {
		t.Fatal(err)
	}
	if m := literalColorUtility.FindString(string(b)); m != "" {
		t.Fatalf("dashboard.tmpl.html uses a literal hex color (%q) instead of a var(--token) reference - it will not respond to data-theme=\"dark\"", m)
	}
}

// TestHydroGoCSSExcludesPreflight is the second collision guard
// ADR-0339/0341 document: Tailwind's global reset layer must never ship
// here, since it would apply to every element on the admin shell page, not
// just .hydrogo-dashboard.
func TestHydroGoCSSExcludesPreflight(t *testing.T) {
	if strings.Contains(CSSText, "box-sizing:border-box}html{") {
		t.Fatal("compiled hydrogo CSS appears to include Tailwind's preflight reset layer - it must be built from tailwindcss/theme.css + tailwindcss/utilities.css only (see internal/ui/hydrogoui/input.css)")
	}
}

// TestHydroGoDashboardIsRestructuredNotAReskin guards the actual design
// intent of this package (ADR-0341): the flagship redesign must genuinely
// restructure the dashboard body (icon-badge stat cards, a two-column
// collections/system-status layout) rather than reuse the old shared
// wb-*/dashboard-* class names with new colors.
func TestHydroGoDashboardIsRestructuredNotAReskin(t *testing.T) {
	b, err := markupFS.ReadFile("dashboard.tmpl.html")
	if err != nil {
		t.Fatal(err)
	}
	markup := string(b)
	for _, dead := range []string{"wb-dashboard", "wb-hero", "wb-metric-strip", "dashboard-kpi", "dashboard-card"} {
		if strings.Contains(markup, dead) {
			t.Errorf("hydrogo dashboard markup still references the old shared class %q - ADR-0341 requires a genuinely new structure, not a reskin", dead)
		}
	}
	for _, want := range []string{"tw:rounded-2xl", "hydrogo-dashboard"} {
		if !strings.Contains(markup, want) {
			t.Errorf("hydrogo dashboard markup missing expected marker %q", want)
		}
	}
}
