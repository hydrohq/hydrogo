package blueprints

import (
	"context"
	"fmt"
	"strings"

	"github.com/hydrohq/hydrogo/internal/collections"
	"github.com/hydrohq/hydrogo/internal/contenttypes"
	"github.com/hydrohq/hydrogo/internal/tenant"
)

// PortfolioDefinitions returns collections for a creative studio / architect / artist portfolio.
func PortfolioDefinitions() []contenttypes.Definition {
	return []contenttypes.Definition{
		{
			ID:          "site_settings",
			Label:       "Studio & Artist Settings",
			Description: "Portfolio identity, biography, contact details and theme colors.",
			Group:       "Portfolio Management",
			Icon:        "camera",
			TitleField:  "artist_name",
			Public:      true,
			Singleton:   true,
			Fields: []contenttypes.Field{
				{ID: "artist_name", Label: "Studio / Artist Name", Kind: "text", Required: true, List: true},
				{ID: "tagline", Label: "Discipline / Title", Kind: "text", List: true},
				{ID: "primary_color", Label: "Primary Accent Color", Kind: "text"},
				{ID: "bg_tone", Label: "Background Tone", Kind: "text"},
				{ID: "font_preset", Label: "Typography Palette", Kind: "select", Options: []string{"v0_geist", "editorial_playfair", "artisan_garamond", "monumental_cinzel", "avantgarde_syne", "modern_grotesk"}},
				{ID: "contact_email", Label: "Contact Email", Kind: "text", List: true},
				{ID: "phone", Label: "Phone", Kind: "text", List: true},
				{ID: "location", Label: "Location / City", Kind: "text"},
				{ID: "instagram", Label: "Instagram", Kind: "text"},
				{ID: "bio", Label: "Biography / Artist Statement", Kind: "textarea"},
				{ID: "hero_title", Label: "Hero Title", Kind: "text"},
				{ID: "hero_subtitle", Label: "Hero Subtext", Kind: "textarea"},
			},
		},
		{
			ID:          "works",
			Label:       "Projects & Works",
			Description: "Photo series, architectural designs and visual works.",
			Group:       "Portfolio Management",
			Icon:        "grid",
			TitleField:  "title",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Project Title", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "slug", Label: "URL Slug", Kind: "text", Required: true, Unique: true, List: true},
				{ID: "category", Label: "Category", Kind: "select", Options: []string{"Residential Architecture", "Interior Architecture & Restoration", "Cultural Space", "Commercial Space", "Photography & Visual Art"}, Required: true, List: true, Filterable: true},
				{ID: "year", Label: "Year", Kind: "text", List: true},
				{ID: "client", Label: "Client / Institution", Kind: "text", Searchable: true},
				{ID: "cover_image", Label: "Cover Image", Kind: "text", Widget: "media", Required: true, List: true},
				{ID: "gallery_json", Label: "Gallery Images (JSON)", Kind: "textarea"},
				{ID: "description", Label: "Project Story", Kind: "textarea", Searchable: true},
				{ID: "featured", Label: "Feature on Homepage", Kind: "boolean", List: true, Filterable: true},
			},
		},
		{
			ID:          "awards",
			Label:       "Awards & Publications",
			Description: "Awards received, exhibitions and press coverage.",
			Group:       "Portfolio Management",
			Icon:        "award",
			TitleField:  "title",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Award / Exhibition Name", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "organization", Label: "Institution / Festival", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "year", Label: "Year", Kind: "text", Required: true, List: true},
				{ID: "project_name", Label: "Related Work", Kind: "text"},
			},
		},
		{
			ID:          "inquiries",
			Label:       "Projects & Collaborations",
			Description: "Incoming exhibition, commission and project requests.",
			Group:       "Inbox",
			Icon:        "inbox",
			TitleField:  "client_name",
			Public:      false,
			Fields: []contenttypes.Field{
				{ID: "client_name", Label: "Name / Organization", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "email", Label: "Email", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "project_type", Label: "Project Type", Kind: "text", List: true},
				{ID: "timeline", Label: "Timeline", Kind: "text"},
				{ID: "message", Label: "Project Details", Kind: "textarea", Required: true, Searchable: true},
				{ID: "status", Label: "Status", Kind: "select", Options: []string{"new", "answered", "agreed", "archived"}, List: true, Filterable: true},
			},
		},
	}
}

// GeneratePortfolioCatalog generates default portfolio data.
func GeneratePortfolioCatalog(name, phone, prompt string) (map[string]any, []map[string]any, []map[string]any) {
	brandName := strings.TrimSpace(name)
	if brandName == "" {
		brandName = "Apex Architecture & Design Studio"
	}

	settings := map[string]any{
		"artist_name":   brandName,
		"tagline":       "Minimalist Architecture, Spatial Experience and Sustainable Design",
		"primary_color": "#0F172A", // Deep Charcoal / Slate
		"bg_tone":       "#F8FAFC", // Museum White
		"font_preset":   "monumental_cinzel",
		"contact_email": "studio@" + strings.ToLower(strings.ReplaceAll(brandName, " ", "")) + ".com",
		"phone":         phone,
		"location":      "Lisbon / Berlin",
		"instagram":     "@apex_studio",
		"bio":           "We design human-centered, timeless buildings and interiors with light, material honesty and spatial fluidity at their core.",
		"hero_title":    "Contemporary Architecture That Shapes the Spirit of Place",
		"hero_subtitle": "Residential, commercial and cultural designs that blend brutalism with nature.",
	}

	works := []map[string]any{
		{
			"title":       "Coastal House: A Dance of Stone and Light",
			"slug":        "bodrum-evi-tas-ve-isik",
			"category":    "Residential Architecture",
			"year":        "2025",
			"client":      "Private Client",
			"cover_image": "https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&w=800&q=80",
			"description": "A zero-energy villa set into the coastal landscape with local slate stone and sustainable olive-wood canopies.",
			"featured":    true,
		},
		{
			"title":       "Harbor Loft & Artist Residence",
			"slug":        "galata-loft-rezidans",
			"category":    "Interior Architecture & Restoration",
			"year":        "2024",
			"client":      "Collector's Home",
			"cover_image": "https://images.unsplash.com/photo-1600607687939-ce8a6c25118c?auto=format&fit=crop&w=800&q=80",
			"description": "A 120-year-old historic brick vault reinterpreted with modern raw concrete and sculptural steel staircases.",
			"featured":    true,
		},
		{
			"title":       "Pavilion X: Contemporary Sculpture Pavilion",
			"slug":        "pavilion-x-sanat",
			"category":    "Cultural Space",
			"year":        "2024",
			"client":      "City Arts Foundation",
			"cover_image": "https://images.unsplash.com/photo-1513694203232-719a280e022f?auto=format&fit=crop&w=800&q=80",
			"description": "A semi-open exhibition space built with parametric timber beams for temporary installations.",
			"featured":    true,
		},
		{
			"title":       "Nordic Monolith Villa",
			"slug":        "nordic-monolith-villa",
			"category":    "Residential Architecture",
			"year":        "2023",
			"client":      "Signature Design",
			"cover_image": "https://images.unsplash.com/photo-1600566753376-12c8ab7fb75b?auto=format&fit=crop&w=800&q=80",
			"description": "A lakeside design with a monolithic black timber shell and a floor-to-ceiling panoramic glass facade.",
			"featured":    false,
		},
	}

	awards := []map[string]any{
		{
			"title":        "Residential Design of the Year",
			"organization": "National Architecture Awards",
			"year":         "2025",
			"project_name": "Coastal House",
		},
		{
			"title":        "Special Award for Adaptive Reuse of a Historic Building",
			"organization": "European Heritage Design Awards",
			"year":         "2024",
			"project_name": "Harbor Loft",
		},
		{
			"title":        "Sustainable Architecture Medal",
			"organization": "Sustainable Building Council",
			"year":         "2023",
			"project_name": "Pavilion X",
		},
	}

	return settings, works, awards
}

// SeedPortfolioStore provisions schemas and initial data for portfolio archetype.
func SeedPortfolioStore(ctx context.Context, colSvc collections.Service, scope tenant.Scope, name, phone, prompt string) error {
	tenantCtx := tenant.WithScope(ctx, scope)
	settings, works, awards := GeneratePortfolioCatalog(name, phone, prompt)

	if _, err := colSvc.CreateRecord(tenantCtx, "site_settings", settings); err != nil {
		return fmt.Errorf("seed portfolio site_settings: %w", err)
	}
	for _, w := range works {
		if _, err := colSvc.CreateRecord(tenantCtx, "works", w); err != nil {
			return fmt.Errorf("seed work %q: %w", w["title"], err)
		}
	}
	for _, a := range awards {
		if _, err := colSvc.CreateRecord(tenantCtx, "awards", a); err != nil {
			return fmt.Errorf("seed award: %w", err)
		}
	}
	return nil
}
