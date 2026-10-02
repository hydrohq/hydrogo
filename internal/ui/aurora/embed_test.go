package aurora

import (
	"regexp"
	"strings"
	"testing"
)

// TestEmbeddedAuroraCSSIsPresent is the contract test required by ADR-0339:
// the compiled, Tailwind-prefixed stylesheet must ship inside the binary and
// must not be an empty placeholder.
func TestEmbeddedAuroraCSSIsPresent(t *testing.T) {
	if strings.TrimSpace(CSSText) == "" {
		t.Fatal("embedded Aurora CSS is empty - run scripts/build-aurora-tailwind.ps1 to regenerate dist/aurora.css")
	}
}

// TestDashboardClassesAreCompiled guards against the exact bug class
// documented at internal/ui/admin_shell.go:2747-2756 (Tailwind utility
// classes used with zero matching compiled CSS). A representative subset of
// the "tw:"-prefixed utility classes used by dashboard.tmpl.html must have a
// real rule in the compiled, embedded stylesheet.
func TestDashboardClassesAreCompiled(t *testing.T) {
	wantSelectors := []string{
		`.tw\:flex{`, `.tw\:grid{`, `.tw\:font-semibold{`, `.tw\:rounded-lg{`,
		`.tw\:tabular-nums{`, `.tw\:border{`,
	}
	for _, sel := range wantSelectors {
		if !strings.Contains(CSSText, sel) {
			t.Errorf("compiled Aurora CSS is missing selector %q used by dashboard.tmpl.html - class is dead/unmatched", sel)
		}
	}
}

// TestAuroraThemeVariablesAreNamespaced is the collision guard ADR-0339
// documents: Tailwind's generated theme CSS custom properties must all be
// prefixed with "--tw-" so they cannot collide with an ADR-0053 semantic
// token name (e.g. "--radius-lg", "--shadow-md") when this stylesheet is
// concatenated into the same page as AdminCSS.
//
// The dashboard body deliberately *references* ADR-0053 tokens (e.g.
// `tw:bg-[var(--surface)]` compiles to `background-color:var(--surface)`)
// so it re-skins correctly per family and across light/dark - that is a
// reference, not a definition, and must stay allowed. What must never
// appear is Tailwind *defining* its own bare `--radius-*`/`--shadow-*`
// custom property (a definition is `--radius-sm:value`, colon directly
// after the name; a reference is always `var(--radius-sm)`, no colon).
var unprefixedTokenDefinition = regexp.MustCompile(`--(?:radius|shadow)-[a-zA-Z0-9]+:`)

func TestAuroraThemeVariablesAreNamespaced(t *testing.T) {
	if loc := unprefixedTokenDefinition.FindString(CSSText); loc != "" {
		t.Fatalf("compiled Aurora CSS defines an unprefixed theme variable (%q) that could collide with the ADR-0053 semantic token contract", loc)
	}
	if !strings.Contains(CSSText, "--tw-") {
		t.Fatal("compiled Aurora CSS does not use the expected Tailwind \"--tw-\" variable prefix")
	}
}

// literalColorUtility matches a color-bearing Tailwind utility
// (bg-/text-/border-/shadow-) given a literal hex value instead of a
// var(--token) reference, e.g. `tw:bg-[#18181b]`. A regression exactly like
// this shipped once already: the dashboard body was colored with literal
// light-mode hex values, so it never responded to data-theme="dark" at all
// (cards stayed white, text stayed near-black, on a dark page). Every
// color-bearing class in the markup must reference an ADR-0053 token via
// var(...) so it re-skins correctly per family and across light/dark.
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

// TestAuroraCSSExcludesPreflight is the second collision guard ADR-0339
// documents: Tailwind's global reset layer must never ship here, since it
// would apply to every element on the admin shell page, not just
// .aurora-dashboard.
func TestAuroraCSSExcludesPreflight(t *testing.T) {
	// Preflight resets common elements unscoped (e.g. a bare "*,::before"
	// margin/padding reset and unscoped "button,input" normalization). Its
	// signature is the unscoped "*,::before,::after{box-sizing" rule; the
	// prefixed theme/utilities-only build must not contain it.
	if strings.Contains(CSSText, "box-sizing:border-box}html{") {
		t.Fatal("compiled Aurora CSS appears to include Tailwind's preflight reset layer - it must be built from tailwindcss/theme.css + tailwindcss/utilities.css only (see internal/ui/aurora/input.css)")
	}
}
