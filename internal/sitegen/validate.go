package sitegen

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var allowedSectionTypes = map[string]bool{
	"hero":          true,
	"editorial":     true,
	"story":         true,
	"featured_grid": true,
	"collection":    true,
	"pillars":       true,
	"features":      true,
	"testimonials":  true,
	"reviews":       true,
	"faq":           true,
	"newsletter":    true,
	"contact_bar":   true,
}

var nonAlphanumericSlug = regexp.MustCompile(`[^a-z0-9]+`)

// Validate checks that a SitePlan conforms to integrity rules and safety boundaries.
func Validate(plan *SitePlan) error {
	if plan == nil {
		return errors.New("site plan cannot be nil")
	}

	if plan.StoreSettings == nil {
		plan.StoreSettings = map[string]any{}
	}

	name, _ := plan.StoreSettings["store_name"].(string)
	if strings.TrimSpace(name) == "" {
		if strings.TrimSpace(plan.Brief) != "" {
			plan.StoreSettings["store_name"] = "Boutique Atelier"
		} else {
			return errors.New("store_name is required in store_settings")
		}
	}

	// Validate theme preset
	if plan.Theme.Preset == "" {
		plan.Theme.Preset = "minimal_cream"
	}
	if plan.Theme.FontPreset == "" {
		plan.Theme.FontPreset = "v0_geist"
	}

	// Validate layout bounds
	if len(plan.Layout) > 30 {
		return fmt.Errorf("layout has %d sections; maximum allowed is 30", len(plan.Layout))
	}

	// Filter and validate sections
	validSections := make([]LayoutSection, 0, len(plan.Layout))
	for _, sec := range plan.Layout {
		t := strings.ToLower(strings.TrimSpace(sec.Type))
		if allowedSectionTypes[t] {
			sec.Type = t
			if sec.Props == nil {
				sec.Props = map[string]any{}
			}
			validSections = append(validSections, sec)
		}
	}
	plan.Layout = validSections

	// Validate products
	if len(plan.Products) > 200 {
		return fmt.Errorf("products count is %d; maximum allowed is 200", len(plan.Products))
	}

	for i := range plan.Products {
		p := &plan.Products[i]
		p.Title = strings.TrimSpace(p.Title)
		if p.Title == "" {
			p.Title = fmt.Sprintf("Ürün %d", i+1)
		}
		if p.Slug == "" {
			p.Slug = slugify(p.Title)
		}
		if p.Price < 0 {
			p.Price = 0
		}
	}

	return nil
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	repl := strings.NewReplacer("ı", "i", "ğ", "g", "ü", "u", "ş", "s", "ö", "o", "ç", "c")
	s = repl.Replace(s)
	s = nonAlphanumericSlug.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
