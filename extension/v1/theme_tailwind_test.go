package v1

import (
	"os"
	"strings"
	"testing"
)

// TestCorporateThemeCSSIsTailwindCompiled is the contract test required by
// ADR-0338/0340: the corporate theme's static/theme.css is now compiled by
// the standalone Tailwind CLI (@apply against the theme's existing
// selectors, zero markup changes - see ADR-0340), not hand-written from
// scratch. It must not be empty, must not include Tailwind's preflight
// reset (which would silently change element defaults across the whole
// public site), and must contain real, matching rules for a representative
// sample of classes the theme's own block templates render.
func TestCorporateThemeCSSIsTailwindCompiled(t *testing.T) {
	data, err := os.ReadFile("../../themes/corporate/static/theme.css")
	if err != nil {
		t.Fatalf("read theme.css: %v", err)
	}
	css := string(data)

	if strings.TrimSpace(css) == "" {
		t.Fatal("themes/corporate/static/theme.css is empty")
	}
	if strings.Contains(css, "box-sizing:border-box}html{") {
		t.Fatal("theme.css appears to include Tailwind's preflight reset layer - it must be compiled from tailwindcss/theme.css + tailwindcss/utilities.css only, unprefixed and without preflight, since it is the theme's OWN top-level stylesheet (see scripts/build-theme-tailwind.ps1 / docs/archive/architecture/0340-tailwind-styling-migration.md)")
	}

	// Regression guard against the exact bug class documented at
	// internal/ui/admin_shell.go:2747-2756 (utility classes used with zero
	// matching compiled CSS) and, separately, against the Tailwind
	// `bg-`/`font-` arbitrary-value ambiguity bugs found and fixed while
	// converting this file (ADR-0340): spot-check that selectors actually
	// used by the block templates compiled to the right PROPERTY, not a
	// plausible-looking wrong one.
	for _, want := range []string{
		// .cta-inner-card (themes/corporate/blocks/cta.html)
		`.cta-inner-card{`,
		// .faq-item (themes/corporate/blocks/faq.html)
		`.faq-item{`,
		// .pricing-card (themes/corporate/blocks/pricing_table.html)
		`.pricing-card{`,
		// .testimonial-card (themes/corporate/blocks/testimonials.html)
		`.testimonial-card{`,
		// .theme-corporate-hero (themes/corporate/blocks/hero.html)
		`.theme-corporate-hero{`,
	} {
		if !strings.Contains(css, want) {
			t.Errorf("theme.css is missing selector %q used by a block template - class is dead/unmatched", want)
		}
	}

	// The font-family/font-weight ambiguity bug (ADR-0340): html{} must set
	// a real font-family from the theme's own --font-sans token, never a
	// font-weight using that same token (which is what the bug produced).
	if strings.Contains(css, "font-weight:var(--font-sans)") {
		t.Fatal("theme.css maps --font-sans through font-weight instead of font-family - Tailwind's ambiguous font-[] utility guessed wrong for this arbitrary value (see ADR-0340)")
	}
	if !strings.Contains(css, "font-family:var(--font-sans)") {
		t.Fatal("theme.css is missing the expected font-family:var(--font-sans) declaration on html{}")
	}

	// The background/background-image "none" ambiguity bug (ADR-0340):
	// must never compile to the invalid "background-color:none".
	if strings.Contains(css, "background-color:none") {
		t.Fatal("theme.css contains the invalid declaration background-color:none - Tailwind's ambiguous bg-[] utility guessed wrong for the keyword \"none\" (see ADR-0340)")
	}
}
