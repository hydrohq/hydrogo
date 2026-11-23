package blueprints

import (
	"github.com/hydrohq/hydrogo/internal/contenttypes"
)

// SaaSDefinitions returns collections for a modern SaaS product landing page.
func SaaSDefinitions() []contenttypes.Definition {
	return []contenttypes.Definition{
		{
			ID:          "site_settings",
			Label:       "SaaS Product & Page Settings",
			Description: "Product headlines, CTA buttons, primary colors and contact details.",
			Group:       "SaaS Management",
			Icon:        "cpu",
			TitleField:  "brand_name",
			Public:      true,
			Singleton:   true,
			Fields: []contenttypes.Field{
				{ID: "brand_name", Label: "Product / SaaS Name", Kind: "text", Required: true, List: true},
				{ID: "tagline", Label: "Product Tagline", Kind: "text", List: true},
				{ID: "primary_color", Label: "Primary Accent Color", Kind: "text"},
				{ID: "bg_tone", Label: "Background Tone", Kind: "text"},
				{ID: "font_preset", Label: "Typography Palette", Kind: "select", Options: []string{"v0_geist", "editorial_playfair", "artisan_garamond", "monumental_cinzel", "avantgarde_syne", "modern_grotesk"}},
				{ID: "contact_email", Label: "Contact Email", Kind: "text", List: true},
				{ID: "hero_badge", Label: "Hero Badge", Kind: "text"},
				{ID: "hero_title", Label: "Hero Title", Kind: "text"},
				{ID: "hero_subtitle", Label: "Hero Subtext", Kind: "textarea"},
				{ID: "cta_label", Label: "Primary CTA Button Text", Kind: "text"},
				{ID: "cta_link", Label: "Primary CTA Link", Kind: "text"},
				{ID: "hero_image", Label: "Hero Dashboard Image", Kind: "text", Widget: "media"},
			},
		},
		{
			ID:          "features",
			Label:       "Product Features (Bento Grid)",
			Description: "Core features and capabilities the platform offers.",
			Group:       "SaaS Management",
			Icon:        "zap",
			TitleField:  "title",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Feature Title", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "badge", Label: "Badge / Category", Kind: "text", List: true, Filterable: true},
				{ID: "description", Label: "Description", Kind: "textarea", Required: true, List: true, Searchable: true},
				{ID: "icon", Label: "Icon", Kind: "text"},
				{ID: "image_url", Label: "Image", Kind: "text", Widget: "media", List: true},
				{ID: "sort_order", Label: "Sort Order", Kind: "number", List: true},
			},
		},
		{
			ID:          "pricing_tiers",
			Label:       "Pricing Plans",
			Description: "Monthly and yearly subscription plans with feature lists.",
			Group:       "SaaS Management",
			Icon:        "credit-card",
			TitleField:  "name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "name", Label: "Plan Name", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "price_monthly", Label: "Monthly Price ($)", Kind: "number", Required: true, List: true},
				{ID: "price_annual", Label: "Yearly Price ($)", Kind: "number", Required: true, List: true},
				{ID: "description", Label: "Plan Description", Kind: "text"},
				{ID: "features_list", Label: "Included Features (one per line)", Kind: "textarea", Required: true},
				{ID: "is_highlighted", Label: "Featured Plan", Kind: "boolean", List: true, Filterable: true},
				{ID: "button_label", Label: "Button Text", Kind: "text"},
				{ID: "button_link", Label: "Button Link", Kind: "text"},
			},
		},
		{
			ID:          "faqs",
			Label:       "Frequently Asked Questions",
			Description: "FAQ on pricing, technical support and integrations.",
			Group:       "SaaS Management",
			Icon:        "help-circle",
			TitleField:  "question",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "question", Label: "Question", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "answer", Label: "Answer", Kind: "textarea", Required: true, Searchable: true},
				{ID: "category", Label: "Category", Kind: "select", Options: []string{"General", "Technical", "Pricing", "Security"}, List: true, Filterable: true},
				{ID: "sort_order", Label: "Sort Order", Kind: "number", List: true},
			},
		},
		{
			ID:          "waitlist",
			Label:       "Early Access & Waitlist",
			Description: "People who signed up before launch or for a free trial.",
			Group:       "Leads & Growth",
			Icon:        "user-plus",
			TitleField:  "email",
			Public:      false,
			Fields: []contenttypes.Field{
				{ID: "email", Label: "Email Address", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "full_name", Label: "Full Name", Kind: "text", List: true, Searchable: true},
				{ID: "company_name", Label: "Company Name", Kind: "text", List: true, Searchable: true},
				{ID: "use_case", Label: "Use Case", Kind: "textarea"},
			},
		},
	}
}
