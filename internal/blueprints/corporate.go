package blueprints

import (
	"context"
	"fmt"
	"strings"

	"github.com/hydrohq/hydrogo/internal/collections"
	"github.com/hydrohq/hydrogo/internal/contenttypes"
	"github.com/hydrohq/hydrogo/internal/tenant"
)

// CorporateDefinitions returns the collections needed for a modern corporate / B2B agency site.
func CorporateDefinitions() []contenttypes.Definition {
	return []contenttypes.Definition{
		{
			ID:          "site_settings",
			Label:       "Company & Site Settings",
			Description: "Corporate identity, contact details, statistics and theme colors.",
			Group:       "Corporate Management",
			Icon:        "building",
			TitleField:  "brand_name",
			Public:      true,
			Singleton:   true,
			Fields: []contenttypes.Field{
				{ID: "brand_name", Label: "Company / Agency Name", Kind: "text", Required: true, List: true},
				{ID: "tagline", Label: "Slogan / Subtitle", Kind: "text", List: true},
				{ID: "primary_color", Label: "Primary Accent Color", Kind: "text"},
				{ID: "bg_tone", Label: "Background Tone", Kind: "text"},
				{ID: "font_preset", Label: "Typography Palette", Kind: "select", Options: []string{"v0_geist", "editorial_playfair", "artisan_garamond", "monumental_cinzel", "avantgarde_syne", "modern_grotesk"}},
				{ID: "phone", Label: "Phone", Kind: "text", List: true},
				{ID: "email", Label: "Email", Kind: "text", List: true},
				{ID: "address", Label: "Office Address", Kind: "text"},
				{ID: "hero_badge", Label: "Hero Badge", Kind: "text"},
				{ID: "hero_title", Label: "Hero Title", Kind: "text"},
				{ID: "hero_subtitle", Label: "Hero Subtext", Kind: "textarea"},
				{ID: "about_title", Label: "About Title", Kind: "text"},
				{ID: "about_text", Label: "About Text", Kind: "textarea"},
				{ID: "stat_1_val", Label: "Statistic 1 Value", Kind: "text"},
				{ID: "stat_1_lbl", Label: "Statistic 1 Label", Kind: "text"},
				{ID: "stat_2_val", Label: "Statistic 2 Value", Kind: "text"},
				{ID: "stat_2_lbl", Label: "Statistic 2 Label", Kind: "text"},
				{ID: "stat_3_val", Label: "Statistic 3 Value", Kind: "text"},
				{ID: "stat_3_lbl", Label: "Statistic 3 Label", Kind: "text"},
			},
		},
		{
			ID:          "services",
			Label:       "Our Services",
			Description: "The areas of expertise and consulting services the company offers.",
			Group:       "Corporate Management",
			Icon:        "briefcase",
			TitleField:  "title",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Service Title", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "slug", Label: "URL Slug", Kind: "text", Required: true, Unique: true, List: true},
				{ID: "icon", Label: "Icon / Badge", Kind: "text", List: true},
				{ID: "short_description", Label: "Short Description", Kind: "textarea", Required: true, List: true, Searchable: true},
				{ID: "detail_description", Label: "Detailed Description", Kind: "richtext"},
				{ID: "featured", Label: "Featured", Kind: "boolean", List: true, Filterable: true},
			},
		},
		{
			ID:          "projects",
			Label:       "Projects & Case Studies",
			Description: "Completed projects, customer success stories and portfolio.",
			Group:       "Corporate Management",
			Icon:        "folder",
			TitleField:  "title",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Project Name", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "slug", Label: "URL Slug", Kind: "text", Required: true, Unique: true, List: true},
				{ID: "client_name", Label: "Client / Brand", Kind: "text", List: true, Searchable: true},
				{ID: "category", Label: "Category", Kind: "select", Options: []string{"Financial Technology", "Logistics & Operations", "Health Technology", "E-Commerce & Retail", "AI & Data"}, List: true, Filterable: true},
				{ID: "image_url", Label: "Image", Kind: "text", Widget: "media", List: true},
				{ID: "summary", Label: "Summary", Kind: "textarea"},
				{ID: "metrics", Label: "Success Metric", Kind: "text", List: true},
				{ID: "featured", Label: "Featured", Kind: "boolean", List: true, Filterable: true},
			},
		},
		{
			ID:          "testimonials",
			Label:       "Client Testimonials",
			Description: "Feedback from the organizations and leaders we work with.",
			Group:       "Corporate Management",
			Icon:        "message-circle",
			TitleField:  "client_name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "client_name", Label: "Person Name", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "company", Label: "Company / Organization", Kind: "text", List: true, Searchable: true},
				{ID: "role", Label: "Position / Title", Kind: "text", List: true},
				{ID: "quote", Label: "Testimonial", Kind: "textarea", Required: true, Searchable: true},
				{ID: "avatar_url", Label: "Photo", Kind: "text", Widget: "media", List: true},
			},
		},
		{
			ID:          "team",
			Label:       "Leadership & Team",
			Description: "Our corporate team and leadership.",
			Group:       "Corporate Management",
			Icon:        "users",
			TitleField:  "name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "name", Label: "Full Name", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "role", Label: "Position / Expertise", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "bio", Label: "Bio Summary", Kind: "textarea"},
				{ID: "avatar_url", Label: "Profile Photo", Kind: "text", Widget: "media", List: true},
				{ID: "linkedin", Label: "LinkedIn Profile", Kind: "text"},
			},
		},
		{
			ID:          "inquiries",
			Label:       "Inquiries & Quote Requests",
			Description: "Leads from the website contact and quote request forms.",
			Group:       "Inbox",
			Icon:        "mail",
			TitleField:  "full_name",
			Public:      false,
			Fields: []contenttypes.Field{
				{ID: "full_name", Label: "Full Name", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "email", Label: "Email", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "phone", Label: "Phone", Kind: "text", List: true},
				{ID: "company", Label: "Company Name", Kind: "text", List: true, Searchable: true},
				{ID: "service_needed", Label: "Service of Interest", Kind: "text", List: true},
				{ID: "message", Label: "Message / Request Summary", Kind: "textarea", Required: true, Searchable: true},
				{ID: "status", Label: "Status", Kind: "select", Options: []string{"new", "in_discussion", "quoted", "completed", "archived"}, List: true, Filterable: true},
			},
		},
	}
}

// GenerateCorporateCatalog generates default corporate site data.
func GenerateCorporateCatalog(name, phone, prompt string) (map[string]any, []map[string]any, []map[string]any, []map[string]any, []map[string]any) {
	brandName := strings.TrimSpace(name)
	if brandName == "" {
		brandName = "Acme Strategy & Consulting"
	}
	cleanPhone := strings.TrimSpace(phone)
	if cleanPhone == "" {
		cleanPhone = "+90 212 555 0199"
	}

	settings := map[string]any{
		"brand_name":    brandName,
		"tagline":       "Future-Ready Digital and Strategic Solutions",
		"primary_color": "#1E3A8A", // Deep Navy
		"bg_tone":       "#F8FAFC", // Clean Slate
		"font_preset":   "v0_geist",
		"phone":         cleanPhone,
		"email":         "iletisim@" + strings.ToLower(strings.ReplaceAll(brandName, " ", "")) + ".com",
		"address":       "100 Example Street, Suite 200, Springfield",
		"hero_badge":    "Corporate Transformation & Engineering",
		"hero_title":    "Grow Your Company with Next-Generation Strategies",
		"hero_subtitle": "We deliver scalable success through data-driven consulting, digital product architecture and corporate strategy.",
		"about_title":   "Over 10 Years of Industry Leadership and Trust",
		"about_text":    "With a vision of engineering and strategic consulting at global standards, we add value to more than 100 organizations worldwide.",
		"stat_1_val":    "140+",
		"stat_1_lbl":    "Completed Global Projects",
		"stat_2_val":    "%99.4",
		"stat_2_lbl":    "Client Retention",
		"stat_3_val":    "12 Years",
		"stat_3_lbl":    "Industry Experience",
	}

	services := []map[string]any{
		{
			"title":              "Strategic Management & Consulting",
			"slug":               "strategic-management",
			"icon":               "Compass",
			"short_description":  "End-to-end strategy planning that modernizes your business model and grows your market share.",
			"detail_description": "Detailed growth and reorganization strategies.",
			"featured":           true,
		},
		{
			"title":              "Enterprise Digital Transformation",
			"slug":               "digital-transformation",
			"icon":               "Layers",
			"short_description":  "Seamless migration from legacy systems to modern microservice and cloud architectures.",
			"detail_description": "High-performance, scalable cloud solutions.",
			"featured":           true,
		},
		{
			"title":              "Data Analytics & Business Intelligence",
			"slug":               "data-analytics",
			"icon":               "BarChart3",
			"short_description":  "We turn big data into live, actionable dashboards for decision makers.",
			"detail_description": "Predictive analytics and AI integrations.",
			"featured":           true,
		},
		{
			"title":              "Cybersecurity & Compliance",
			"slug":               "cybersecurity",
			"icon":               "ShieldCheck",
			"short_description":  "Enterprise security audits fully aligned with GDPR and ISO 27001 requirements.",
			"detail_description": "Infrastructure protection built on zero-trust principles.",
			"featured":           false,
		},
	}

	projects := []map[string]any{
		{
			"title":       "Fintech Payment Infrastructure Modernization",
			"slug":        "fintech-payment-modernization",
			"client_name": "Global Pay Solutions",
			"category":    "Financial Technology",
			"image_url":   "https://images.unsplash.com/photo-1551836022-d5d88e9218df?auto=format&fit=crop&w=800&q=80",
			"summary":     "A next-generation distributed payment engine handling 2M+ transactions a day with zero downtime.",
			"metrics":     "340% Speed Increase",
			"featured":    true,
		},
		{
			"title":       "B2B Supply Chain Automation",
			"slug":        "supply-chain-automation",
			"client_name": "Nordic Logistics Ltd.",
			"category":    "Logistics & Operations",
			"image_url":   "https://images.unsplash.com/photo-1586528116311-ad8dd3c8310d?auto=format&fit=crop&w=800&q=80",
			"summary":     "We cut costs by automating warehouse management with real-time IoT sensors.",
			"metrics":     "42% Cost Savings",
			"featured":    true,
		},
		{
			"title":       "Health Group Patient Portal & Data Integration",
			"slug":        "health-portal-integration",
			"client_name": "Meridian Health Group",
			"category":    "Health Technology",
			"image_url":   "https://images.unsplash.com/photo-1576091160399-112ba8d25d1d?auto=format&fit=crop&w=800&q=80",
			"summary":     "We unified appointments and lab results for 300,000 active patients in a single mobile experience.",
			"metrics":     "4.9/5 Mobile Rating",
			"featured":    true,
		},
	}

	testimonials := []map[string]any{
		{
			"client_name": "Kemal Aksoy",
			"company":     "Atlas Finans",
			"role":        "Chief Technology Officer (CTO)",
			"quote":       "The Acme team delivered our complex banking integrations two months ahead of schedule with zero defects. A partnership we can trust.",
			"avatar_url":  "https://images.unsplash.com/photo-1534528741775-53994a69daeb?auto=format&fit=crop&w=200&q=80",
		},
		{
			"client_name": "Selin Varol",
			"company":     "Vortex Holding",
			"role":        "Chief Operating Officer (COO)",
			"quote":       "On our digitalization journey they delivered strategies that doubled our institutional memory and operational speed.",
			"avatar_url":  "https://images.unsplash.com/photo-1580489944761-15a19d654956?auto=format&fit=crop&w=200&q=80",
		},
	}

	team := []map[string]any{
		{
			"name":       "Dr. Emre Can",
			"role":       "Co-Founder & Chief Architect",
			"bio":        "15+ years of experience in enterprise software and organizational design.",
			"avatar_url": "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?auto=format&fit=crop&w=300&q=80",
			"linkedin":   "https://linkedin.com",
		},
		{
			"name":       "Derya Yılmaz",
			"role":       "Director of Strategy & Growth",
			"bio":        "Former top-tier strategy consultant who led 30+ M&A and transformation projects at multinational companies.",
			"avatar_url": "https://images.unsplash.com/photo-1573496359142-b8d87734a5a2?auto=format&fit=crop&w=300&q=80",
			"linkedin":   "https://linkedin.com",
		},
	}

	return settings, services, projects, testimonials, team
}

// SeedCorporateStore provisions schemas and seeds initial content for corporate archetype.
func SeedCorporateStore(ctx context.Context, colSvc collections.Service, scope tenant.Scope, name, phone, prompt string) error {
	tenantCtx := tenant.WithScope(ctx, scope)
	settings, services, projects, testimonials, team := GenerateCorporateCatalog(name, phone, prompt)

	if _, err := colSvc.CreateRecord(tenantCtx, "site_settings", settings); err != nil {
		return fmt.Errorf("seed corporate site_settings: %w", err)
	}
	for _, s := range services {
		if _, err := colSvc.CreateRecord(tenantCtx, "services", s); err != nil {
			return fmt.Errorf("seed service %q: %w", s["title"], err)
		}
	}
	for _, p := range projects {
		if _, err := colSvc.CreateRecord(tenantCtx, "projects", p); err != nil {
			return fmt.Errorf("seed project %q: %w", p["title"], err)
		}
	}
	for _, t := range testimonials {
		if _, err := colSvc.CreateRecord(tenantCtx, "testimonials", t); err != nil {
			return fmt.Errorf("seed testimonial: %w", err)
		}
	}
	for _, m := range team {
		if _, err := colSvc.CreateRecord(tenantCtx, "team", m); err != nil {
			return fmt.Errorf("seed team member: %w", err)
		}
	}
	return nil
}
