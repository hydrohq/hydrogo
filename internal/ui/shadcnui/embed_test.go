package shadcnui

import (
	"regexp"
	"strings"
	"testing"
)

// TestEmbeddedShadcnCSSIsPresent is the contract test required by ADR-0341
// (mirroring ADR-0339's Aurora precedent): the compiled, Tailwind-prefixed
// stylesheet must ship inside the binary and must not be an empty
// placeholder.
func TestEmbeddedShadcnCSSIsPresent(t *testing.T) {
	if strings.TrimSpace(CSSText) == "" {
		t.Fatal("embedded shadcn dashboard CSS is empty - run scripts/build-shadcnui-tailwind.ps1 to regenerate dist/shadcn.css")
	}
}

// TestDashboardClassesAreCompiled guards against the exact bug class
// documented in internal/ui/aurora: Tailwind utility classes used with zero
// matching compiled CSS. A representative subset of the "tw:"-prefixed
// utility classes used by dashboard.tmpl.html must have a real rule in the
// compiled, embedded stylesheet.
func TestDashboardClassesAreCompiled(t *testing.T) {
	wantSelectors := []string{
		`.tw\:flex{`, `.tw\:grid{`, `.tw\:rounded-xl{`, `.tw\:shadow-sm{`,
		`.tw\:font-semibold{`, `.tw\:border{`,
	}
	for _, sel := range wantSelectors {
		if !strings.Contains(CSSText, sel) {
			t.Errorf("compiled shadcn CSS is missing selector %q used by dashboard.tmpl.html - class is dead/unmatched", sel)
		}
	}
}

// TestShadcnThemeVariablesAreNamespaced is the collision guard ADR-0339/0341
// document: Tailwind's generated theme CSS custom properties must all be
// prefixed with "--tw-" so they cannot collide with an ADR-0053 semantic
// token name (e.g. "--radius-lg", "--shadow-sm") when this stylesheet is
// concatenated into the same page as AdminCSS.
var unprefixedTokenDefinition = regexp.MustCompile(`--(?:radius|shadow)-[a-zA-Z0-9]+:`)

func TestShadcnThemeVariablesAreNamespaced(t *testing.T) {
	if loc := unprefixedTokenDefinition.FindString(CSSText); loc != "" {
		t.Fatalf("compiled shadcn CSS defines an unprefixed theme variable (%q) that could collide with the ADR-0053 semantic token contract", loc)
	}
	if !strings.Contains(CSSText, "--tw-") {
		t.Fatal("compiled shadcn CSS does not use the expected Tailwind \"--tw-\" variable prefix")
	}
}

// literalColorUtility matches a color-bearing Tailwind utility
// (bg-/text-/border-/shadow-) given a literal hex value instead of a
// var(--token) reference. Every color-bearing class in the markup must
// reference an ADR-0053 token via var(...) so it re-skins correctly across
// light/dark.
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

// TestShadcnCSSExcludesPreflight is the second collision guard ADR-0339/0341
// document: Tailwind's global reset layer must never ship here, since it
// would apply to every element on the admin shell page, not just
// .shadcn-dashboard.
func TestShadcnCSSExcludesPreflight(t *testing.T) {
	if strings.Contains(CSSText, "box-sizing:border-box}html{") {
		t.Fatal("compiled shadcn CSS appears to include Tailwind's preflight reset layer - it must be built from tailwindcss/theme.css + tailwindcss/utilities.css only (see internal/ui/shadcnui/input.css)")
	}
}

// TestShadcnDashboardUsesCardRecipe guards the actual design intent of this
// package: the markup must use shadcn/ui's real card recipe
// (rounded-xl border ... shadow-sm) and real table/badge conventions, not
// the generic zinc-colored shared markup it used to reuse.
func TestShadcnDashboardUsesCardRecipe(t *testing.T) {
	b, err := markupFS.ReadFile("dashboard.tmpl.html")
	if err != nil {
		t.Fatal(err)
	}
	markup := string(b)
	for _, want := range []string{"tw:rounded-xl", "tw:shadow-sm", "tw:rounded-full", "<table", "<thead"} {
		if !strings.Contains(markup, want) {
			t.Errorf("shadcn dashboard markup missing expected shadcn/ui convention %q", want)
		}
	}
}
