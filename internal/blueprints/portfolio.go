package blueprints

import (
	"github.com/hydrohq/hydrogo/internal/contenttypes"
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
