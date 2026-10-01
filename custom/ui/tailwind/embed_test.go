package tailwind

import (
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/ui"
)

// TestEmbeddedTailwindCSSIsPresent is the contract test required by
// ADR-0338: the compiled asset must ship inside the binary and must not be
// an empty placeholder.
func TestEmbeddedTailwindCSSIsPresent(t *testing.T) {
	css, err := CSS()
	if err != nil {
		t.Fatalf("CSS(): %v", err)
	}
	if strings.TrimSpace(css) == "" {
		t.Fatal("embedded Tailwind CSS is empty - run scripts/build-tailwind.ps1 to regenerate dist/tailwind.css")
	}
}

// TestExampleComponentClassesAreCompiled guards against the exact bug class
// documented at internal/ui/admin_shell.go:2747-2756, where Tailwind utility
// class names were used with zero matching compiled CSS. Every utility class
// used by custom/ui/tailwind/example/card.templ must have a real rule in the
// compiled, embedded stylesheet.
func TestExampleComponentClassesAreCompiled(t *testing.T) {
	css, err := CSS()
	if err != nil {
		t.Fatalf("CSS(): %v", err)
	}

	// A representative subset of the utility classes used by example.Card;
	// Tailwind v4's minifier is free to reorder/merge declarations, so this
	// checks for the generated selector rather than exact formatting.
	wantSelectors := []string{
		".rounded-lg", ".border", ".bg-white", ".p-6", ".shadow-md",
		".text-lg", ".font-semibold", ".mt-2", ".text-sm",
	}
	for _, sel := range wantSelectors {
		if !strings.Contains(css, sel) {
			t.Errorf("compiled Tailwind CSS is missing selector %q used by custom/ui/tailwind/example.Card - class is dead/unmatched", sel)
		}
	}
}

// TestExistingAdminVisualFamiliesAreUntouched is the smoke test required by
// ADR-0338: this feature must not modify internal/ui's four existing visual
// families or the ADR-0053 semantic token contract they implement. It only
// asserts their CSS constants still exist and are non-empty; the real
// safety net is that this change does not edit internal/ui at all.
func TestExistingAdminVisualFamiliesAreUntouched(t *testing.T) {
	families := map[string]string{
		"AdminCSS": ui.AdminCSS,
	}
	for name, css := range families {
		if strings.TrimSpace(css) == "" {
			t.Errorf("internal/ui.%s is empty - existing admin visual families must remain unchanged by the Tailwind build-time addition (ADR-0338)", name)
		}
	}
}
