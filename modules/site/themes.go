package site

import (
	"bytes"
	"encoding/json"
	"strings"
)

// ThemeSettingIDKey and ThemeSettingTokensKey store the active storefront theme
// and validated design-token overrides in site_settings.
const (
	ThemeSettingIDKey     = "storefront.theme_id"
	ThemeSettingTokensKey = "storefront.theme_tokens"
)

// ThemeTokens holds validated, injection-safe CSS design tokens for a storefront theme.
type ThemeTokens struct {
	Bg            string `json:"bg"`
	Surface       string `json:"surface"`
	Fg            string `json:"fg"`
	Muted         string `json:"muted"`
	Accent        string `json:"accent"`
	FontDisplay   string `json:"font_display"`
	FontBody      string `json:"font_body"`
	Radius        string `json:"radius"`
	HeaderVariant string `json:"header_variant"`
}

// ThemeBlockBlueprint defines a starter block compatible with the pages block editor.
type ThemeBlockBlueprint struct {
	Type  string         `json:"type"`
	Label string         `json:"label"`
	Props map[string]any `json:"props,omitempty"`
}

// ThemeManifest describes a built-in selectable storefront theme.
type ThemeManifest struct {
	ID                string                `json:"id"`
	Name              string                `json:"name"`
	Description       string                `json:"description"`
	SupportedPresets  []string              `json:"supported_presets"`
	RequiredModules   []string              `json:"required_modules,omitempty"`
	RecommendedLayout string                `json:"recommended_layout"`
	DefaultBlocks     []ThemeBlockBlueprint `json:"default_blocks"`
	DefaultTokens     ThemeTokens           `json:"default_tokens"`
}

var defaultThemes = []ThemeManifest{
	{
		ID:                "minimal",
		Name:              "Minimal",
		Description:       "Clean, fast-loading layout for corporate sites, portfolios, and SaaS landing pages.",
		SupportedPresets:  []string{"minimal", "corporate", "docs", "saas"},
		RecommendedLayout: "landing",
		DefaultBlocks: []ThemeBlockBlueprint{
			{Type: "hero", Label: "Hero", Props: map[string]any{"title": "Build faster with clarity", "subtitle": "Clean modular sections ready for your content."}},
			{Type: "feature_list", Label: "Feature list"},
			{Type: "stats", Label: "Statistics"},
			{Type: "pricing_table", Label: "Pricing table"},
			{Type: "cta", Label: "Call to action"},
		},
		DefaultTokens: ThemeTokens{
			Bg:            "#ffffff",
			Surface:       "#f8fafc",
			Fg:            "#0f172a",
			Muted:         "#64748b",
			Accent:        "#00add8",
			FontDisplay:   "sans",
			FontBody:      "sans",
			Radius:        "md",
			HeaderVariant: "split",
		},
	},
	{
		ID:                "boutique",
		Name:              "Boutique",
		Description:       "Visual-first commerce storefront with editorial hero, lookbook grid, and slide-over cart.",
		SupportedPresets:  []string{"boutique", "commerce"},
		RequiredModules:   []string{"commerce"},
		RecommendedLayout: "landing",
		DefaultBlocks: []ThemeBlockBlueprint{
			{Type: "hero", Label: "Hero", Props: map[string]any{"title": "New Season Collection", "subtitle": "Curated essentials crafted for everyday wear."}},
			{Type: "carousel", Label: "Image carousel"},
			{Type: "trust_bar", Label: "Trust badges"},
			{Type: "testimonials", Label: "Testimonials"},
			{Type: "cta", Label: "Call to action"},
		},
		DefaultTokens: ThemeTokens{
			Bg:            "#faf8f5",
			Surface:       "#f2ede4",
			Fg:            "#1c1917",
			Muted:         "#78716c",
			Accent:        "#9a3412",
			FontDisplay:   "serif",
			FontBody:      "sans",
			Radius:        "sm",
			HeaderVariant: "centered",
		},
	},
	{
		ID:                "editorial",
		Name:              "Editorial",
		Description:       "Typography-led newsroom and magazine layout with category rails and featured stories.",
		SupportedPresets:  []string{"news", "tr-news", "blog"},
		RecommendedLayout: "contained",
		DefaultBlocks: []ThemeBlockBlueprint{
			{Type: "hero", Label: "Hero", Props: map[string]any{"title": "Featured Dispatch", "subtitle": "Independent reporting and deep-dive essays."}},
			{Type: "columns", Label: "Columns"},
			{Type: "callout", Label: "Callout"},
			{Type: "faq", Label: "FAQ"},
		},
		DefaultTokens: ThemeTokens{
			Bg:            "#ffffff",
			Surface:       "#f4f4f5",
			Fg:            "#18181b",
			Muted:         "#52525b",
			Accent:        "#dc2626",
			FontDisplay:   "serif",
			FontBody:      "serif",
			Radius:        "none",
			HeaderVariant: "centered",
		},
	},
	{
		ID:                "b2b-catalog",
		Name:              "B2B Catalog",
		Description:       "High-density wholesale catalog with quick-order matrix, SKU search, and dealer pricing.",
		SupportedPresets:  []string{"erp-lite", "commerce"},
		RequiredModules:   []string{"commerce"},
		RecommendedLayout: "default",
		DefaultBlocks: []ThemeBlockBlueprint{
			{Type: "hero", Label: "Hero", Props: map[string]any{"title": "Wholesale & Dealer Portal", "subtitle": "Bulk ordering, tiered pricing, and fast dispatch."}},
			{Type: "trust_bar", Label: "Trust badges"},
			{Type: "feature_list", Label: "Feature list"},
			{Type: "pricing_table", Label: "Pricing table"},
			{Type: "contact_cta", Label: "Call / WhatsApp contact"},
		},
		DefaultTokens: ThemeTokens{
			Bg:            "#f8fafc",
			Surface:       "#ffffff",
			Fg:            "#0f172a",
			Muted:         "#475569",
			Accent:        "#2563eb",
			FontDisplay:   "sans",
			FontBody:      "sans",
			Radius:        "sm",
			HeaderVariant: "minimal",
		},
	},
}

// DefaultThemes returns a copy of the built-in storefront theme catalog.
func DefaultThemes() []ThemeManifest {
	out := make([]ThemeManifest, len(defaultThemes))
	for i, t := range defaultThemes {
		t.SupportedPresets = append([]string(nil), t.SupportedPresets...)
		if len(t.RequiredModules) > 0 {
			t.RequiredModules = append([]string(nil), t.RequiredModules...)
		}
		out[i] = t
	}
	return out
}

// FindTheme looks up a built-in theme by ID.
func FindTheme(id string) (ThemeManifest, bool) {
	clean := strings.TrimSpace(id)
	for _, t := range defaultThemes {
		if t.ID == clean {
			return t, true
		}
	}
	return ThemeManifest{}, false
}

// ValidateThemeID accepts an empty string (default) or a known built-in theme ID.
func ValidateThemeID(value string) bool {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return true
	}
	_, ok := FindTheme(clean)
	return ok
}

// ValidateThemeTokensJSON accepts an empty string or a strict JSON object of ThemeTokens.
// Unknown fields, non-hex colors, and unapproved font/radius/header values are rejected
// to prevent CSS injection when storefronts render tokens as CSS custom properties.
func ValidateThemeTokensJSON(value string) bool {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return true
	}
	if len(clean) > 2048 {
		return false
	}
	var tokens ThemeTokens
	dec := json.NewDecoder(bytes.NewBufferString(clean))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&tokens); err != nil {
		return false
	}
	for _, color := range []string{tokens.Bg, tokens.Surface, tokens.Fg, tokens.Muted, tokens.Accent} {
		if !validHexColor(color) {
			return false
		}
	}
	if !validEnum(tokens.FontDisplay, "", "sans", "serif", "mono", "display") {
		return false
	}
	if !validEnum(tokens.FontBody, "", "sans", "serif", "mono") {
		return false
	}
	if !validEnum(tokens.Radius, "", "none", "sm", "md", "lg") {
		return false
	}
	if !validEnum(tokens.HeaderVariant, "", "split", "centered", "minimal") {
		return false
	}
	return true
}

// ResolveTheme returns the active (or previewed) ThemeManifest and merged ThemeTokens.
func ResolveTheme(activeID, rawTokensJSON, previewID string) (ThemeManifest, ThemeTokens) {
	if p := strings.TrimSpace(previewID); p != "" {
		if theme, ok := FindTheme(p); ok {
			return theme, theme.DefaultTokens
		}
	}
	theme, ok := FindTheme(activeID)
	if !ok {
		theme = defaultThemes[0]
	}
	merged := theme.DefaultTokens
	if ValidateThemeTokensJSON(rawTokensJSON) && strings.TrimSpace(rawTokensJSON) != "" {
		var override ThemeTokens
		if err := json.Unmarshal([]byte(rawTokensJSON), &override); err == nil {
			if override.Bg != "" {
				merged.Bg = override.Bg
			}
			if override.Surface != "" {
				merged.Surface = override.Surface
			}
			if override.Fg != "" {
				merged.Fg = override.Fg
			}
			if override.Muted != "" {
				merged.Muted = override.Muted
			}
			if override.Accent != "" {
				merged.Accent = override.Accent
			}
			if override.FontDisplay != "" {
				merged.FontDisplay = override.FontDisplay
			}
			if override.FontBody != "" {
				merged.FontBody = override.FontBody
			}
			if override.Radius != "" {
				merged.Radius = override.Radius
			}
			if override.HeaderVariant != "" {
				merged.HeaderVariant = override.HeaderVariant
			}
		}
	}
	return theme, merged
}

func validHexColor(v string) bool {
	if v == "" {
		return true
	}
	if len(v) != 7 || v[0] != '#' {
		return false
	}
	for _, c := range v[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func validEnum(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}
