package blueprints

import (
	"context"
	"fmt"
	"strings"

	"github.com/hydrohq/hydrogo/internal/collections"
	"github.com/hydrohq/hydrogo/internal/contenttypes"
	"github.com/hydrohq/hydrogo/internal/tenant"
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

// GenerateSaaSCatalog generates default SaaS startup landing data.
func GenerateSaaSCatalog(name, phone, prompt string) (map[string]any, []map[string]any, []map[string]any, []map[string]any) {
	brandName := strings.TrimSpace(name)
	if brandName == "" {
		brandName = "NovaFlow AI"
	}

	settings := map[string]any{
		"brand_name":    brandName,
		"tagline":       "A Modern Platform That Automates Your Workflows with AI",
		"primary_color": "#2563EB", // Electric Royal Blue
		"bg_tone":       "#09090B", // Pure Zinc Dark
		"font_preset":   "v0_geist",
		"contact_email": "hello@" + strings.ToLower(strings.ReplaceAll(brandName, " ", "")) + ".io",
		"hero_badge":    "v3.0 Enterprise Distributed Workflows",
		"hero_title":    "Run Your Distributed Workflows with Autonomous Agents",
		"hero_subtitle": "Unify all your systems in a single central automation layer. Sub-3ms P99 latency, cryptographic SHA-256 auditing and zero-build deployment speed.",
		"cta_label":     "Try Free for 14 Days",
		"cta_link":      "#pricing",
		"hero_image":    "https://images.unsplash.com/photo-1551288049-bebda4e38f71?auto=format&fit=crop&w=1200&q=80",
	}

	features := []map[string]any{
		{
			"title":       "Real-Time Data Synchronization",
			"badge":       "Performance",
			"description": "Get two-way, millisecond-level data flow with 50+ popular SaaS services and databases.",
			"icon":        "RefreshCw",
			"sort_order":  1,
		},
		{
			"title":       "AI-Powered Automatic Analysis",
			"badge":       "Machine Learning",
			"description": "Automatically classify incoming data, detect anomalies and send proactive alerts to your team.",
			"icon":        "Cpu",
			"sort_order":  2,
		},
		{
			"title":       "End-to-End Enterprise Security",
			"badge":       "SOC2 & GDPR",
			"description": "All your data is encrypted with AES-256 at rest and in transit. Role-based access control (RBAC) is included.",
			"icon":        "Shield",
			"sort_order":  3,
		},
		{
			"title":       "Developer-Friendly Open API & Webhooks",
			"badge":       "Developer Experience",
			"description": "Full flexibility with TypeScript and Go SDKs, OpenAPI documentation and instant webhook notifications.",
			"icon":        "Code",
			"sort_order":  4,
		},
	}

	pricing := []map[string]any{
		{
			"name":           "Starter",
			"price_monthly":  29.0,
			"price_annual":   24.0,
			"description":    "Ideal starting point for growing small teams and solo founders.",
			"features_list":  "Up to 5 Users\n100,000 Operations / Month\nStandard Integrations\nCommunity Support",
			"is_highlighted": false,
			"button_label":   "Get Started",
			"button_link":    "#waitlist",
		},
		{
			"name":           "Pro Growth",
			"price_monthly":  79.0,
			"price_annual":   64.0,
			"description":    "Full power for fast-scaling startups and mid-sized companies.",
			"features_list":  "Unlimited Users\n2,500,000 Operations / Month\nAI Automation Engine\nPriority 24/7 Support\nCustom Webhooks & Integrations",
			"is_highlighted": true,
			"button_label":   "Try Free for 14 Days",
			"button_link":    "#waitlist",
		},
		{
			"name":           "Enterprise",
			"price_monthly":  249.0,
			"price_annual":   199.0,
			"description":    "For large organizations that need custom security, SLA guarantees and on-premise deployment.",
			"features_list":  "Unlimited Volume & Operations\nDedicated Server\n99.99% SLA Guarantee\nDedicated Customer Success Manager\nSSO / SAML Integration",
			"is_highlighted": false,
			"button_label":   "Talk to Sales",
			"button_link":    "#waitlist",
		},
	}

	faqs := []map[string]any{
		{
			"question":   "Is a credit card required for the free trial?",
			"answer":     "No! You can start the 14-day trial instantly without entering any card details.",
			"category":   "General",
			"sort_order": 1,
		},
		{
			"question":   "How long does integration with my existing systems take?",
			"answer":     "Thanks to our ready-made connectors, a typical integration takes less than 15 minutes. We provide ready SDKs for advanced API integrations.",
			"category":   "Technical",
			"sort_order": 2,
		},
		{
			"question":   "Can I cancel my subscription at any time?",
			"answer":     "Absolutely. There are no commitments or hidden fees. You can cancel with one click from your dashboard.",
			"category":   "Pricing",
			"sort_order": 3,
		},
	}

	return settings, features, pricing, faqs
}

// SeedSaaSStore provisions schemas and initial data for saas archetype.
func SeedSaaSStore(ctx context.Context, colSvc collections.Service, scope tenant.Scope, name, phone, prompt string) error {
	tenantCtx := tenant.WithScope(ctx, scope)
	settings, features, pricing, faqs := GenerateSaaSCatalog(name, phone, prompt)

	if _, err := colSvc.CreateRecord(tenantCtx, "site_settings", settings); err != nil {
		return fmt.Errorf("seed saas site_settings: %w", err)
	}
	for _, f := range features {
		if _, err := colSvc.CreateRecord(tenantCtx, "features", f); err != nil {
			return fmt.Errorf("seed feature %q: %w", f["title"], err)
		}
	}
	for _, p := range pricing {
		if _, err := colSvc.CreateRecord(tenantCtx, "pricing_tiers", p); err != nil {
			return fmt.Errorf("seed pricing tier %q: %w", p["name"], err)
		}
	}
	for _, q := range faqs {
		if _, err := colSvc.CreateRecord(tenantCtx, "faqs", q); err != nil {
			return fmt.Errorf("seed faq: %w", err)
		}
	}
	return nil
}
