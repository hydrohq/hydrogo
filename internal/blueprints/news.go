package blueprints

import (
	"context"
	"fmt"
	"strings"

	"github.com/hydrohq/hydrogo/internal/collections"
	"github.com/hydrohq/hydrogo/internal/contenttypes"
	"github.com/hydrohq/hydrogo/internal/tenant"
)

// NewsDefinitions returns the collection schemas for a modern digital news portal.
func NewsDefinitions() []contenttypes.Definition {
	return []contenttypes.Definition{
		{
			ID:          "site_settings",
			Label:       "Portal & Publication Settings",
			Description: "Newspaper/portal name, tagline, accent colors, masthead and announcements.",
			Group:       "Publication Management",
			Icon:        "newspaper",
			TitleField:  "brand_name",
			Public:      true,
			Singleton:   true,
			Fields: []contenttypes.Field{
				{ID: "brand_name", Label: "Portal Name", Kind: "text", Required: true, List: true},
				{ID: "tagline", Label: "Publication Motto / Tagline", Kind: "text", List: true},
				{ID: "primary_color", Label: "Primary Accent Color", Kind: "text"},
				{ID: "bg_tone", Label: "Background Tone", Kind: "text"},
				{ID: "font_preset", Label: "Typography Palette", Kind: "select", Options: []string{"v0_geist", "editorial_playfair", "artisan_garamond", "monumental_cinzel", "avantgarde_syne", "modern_grotesk"}},
				{ID: "breaking_news", Label: "Breaking News Headline", Kind: "text", List: true},
				{ID: "contact_email", Label: "Editorial Desk Email", Kind: "text", List: true},
			},
		},
		{
			ID:          "news_categories",
			Label:       "News Categories",
			Description: "Headlines, Technology, Economy, Culture, Science, etc.",
			Group:       "Publication Management",
			Icon:        "folder",
			TitleField:  "name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "name", Label: "Category Name", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "slug", Label: "URL Slug", Kind: "text", Required: true, Unique: true, List: true, Filterable: true},
				{ID: "sort_order", Label: "Sort Order", Kind: "number", List: true},
				{ID: "description", Label: "Category Description", Kind: "text"},
			},
		},
		{
			ID:          "news_articles",
			Label:       "News & Articles",
			Description: "Published headlines, analyses and opinion pieces.",
			Group:       "Publication Management",
			Icon:        "file-text",
			TitleField:  "title",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Headline", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "slug", Label: "URL Slug", Kind: "text", Required: true, Unique: true, List: true},
				{ID: "category_slug", Label: "Category", Kind: "text", Widget: "relation", To: "news_categories", ValueField: "slug", Required: true, List: true, Filterable: true},
				{ID: "excerpt", Label: "Summary / Standfirst", Kind: "textarea", Required: true, List: true, Searchable: true},
				{ID: "body", Label: "Article Body", Kind: "richtext", Required: true, Searchable: true},
				{ID: "image_url", Label: "Cover Image", Kind: "text", Widget: "media", List: true},
				{ID: "author_name", Label: "Author / Reporter", Kind: "text", List: true, Searchable: true},
				{ID: "reading_time", Label: "Reading Time (min)", Kind: "number", List: true},
				{ID: "published_at", Label: "Published At", Kind: "text", List: true},
				{ID: "is_lead", Label: "Top Story of the Day", Kind: "boolean", List: true, Filterable: true},
				{ID: "is_trending", Label: "Most Read", Kind: "boolean", List: true, Filterable: true},
			},
		},
	}
}

// GenerateNewsCatalog builds authentic, rich editorial news content.
func GenerateNewsCatalog(brandName, phone, prompt string) (map[string]any, []map[string]any, []map[string]any) {
	if brandName == "" {
		brandName = "Nexus News"
	}

	settings := map[string]any{
		"brand_name":    brandName,
		"tagline":       "Independent Digital Newspaper, In-Depth Analysis & Technology",
		"primary_color": "#DC2626", // Editorial Crimson
		"bg_tone":       "#09090B",
		"font_preset":   "editorial_playfair",
		"contact_email": "editor@" + strings.ToLower(strings.ReplaceAll(brandName, " ", "")) + ".com",
		"breaking_news": "A New Era in AI Regulation: Global Open-Source Standards Adopted",
	}

	categories := []map[string]any{
		{"name": "Headlines", "slug": "headlines", "sort_order": 1, "description": "Breaking developments from around the world"},
		{"name": "Technology", "slug": "technology", "sort_order": 2, "description": "AI, software and digital transformation"},
		{"name": "Economy", "slug": "economy", "sort_order": 3, "description": "Markets, entrepreneurship and macroeconomics"},
		{"name": "Culture & Arts", "slug": "culture", "sort_order": 4, "description": "Literature, architecture, film and ideas"},
		{"name": "Science", "slug": "science", "sort_order": 5, "description": "Space research, biotechnology and climate"},
	}

	articles := []map[string]any{
		{
			"title":         "The Age of Autonomous Software Agents: From Writing Code to Managing Processes",
			"slug":          "age-of-autonomous-software-agents",
			"category_slug": "technology",
			"excerpt":       "Development teams are becoming system architects who define goals and supervise autonomous loops rather than writing code line by line.",
			"body":          "As AI models evolve from simple text completion to multi-step planning and self-debugging, software engineering is being fundamentally transformed.\n\nNew-generation agentic frameworks can autonomously manage the entire lifecycle, from database schemas to end-to-end microservice integrations. Industry leaders predict that 70% of enterprise software will be built with autonomous agent coordination by 2027.",
			"image_url":     "https://images.unsplash.com/photo-1485827404703-89b55fcc595e?auto=format&fit=crop&w=1400&q=80",
			"author_name":   "Deniz Aksoy",
			"reading_time":  4,
			"published_at":  "Today 14:30",
			"is_lead":       true,
			"is_trending":   true,
		},
		{
			"title":         "Global Central Banks and Digital Currency Architectures: A New Financial Order",
			"slug":          "central-banks-digital-currency",
			"category_slug": "economy",
			"excerpt":       "New-generation CBDC and distributed reconciliation protocols that cut cross-border settlement times to seconds are coming into force.",
			"body":          "The high transaction costs and multi-day settlement times of traditional correspondent banking networks are giving way to cryptographically auditable distributed reconciliation layers.\n\nPilot projects run jointly by European and Asian central banks reported a 35% improvement in capital efficiency in liquidity management.",
			"image_url":     "https://images.unsplash.com/photo-1611974789855-9c2a0a7236a3?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Murat Çelik",
			"reading_time":  5,
			"published_at":  "Today 11:15",
			"is_lead":       false,
			"is_trending":   true,
		},
		{
			"title":         "Spatial Elegance: Mediterranean Minimalism and Contemporary Urban Architecture",
			"slug":          "spatial-elegance-mediterranean-minimalism",
			"category_slug": "culture",
			"excerpt":       "Natural limestone, play of shadows and brutalist simplicity are creating a new space of calm in modern residential design.",
			"body":          "Courtyards shielded from the city noise, raw timber and local stonework have moved to the center of contemporary design. Architects stress that material honesty, free of ostentation, is the key to long-term tranquility.",
			"image_url":     "https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Selin Vural",
			"reading_time":  3,
			"published_at":  "Yesterday 18:40",
			"is_lead":       false,
			"is_trending":   false,
		},
		{
			"title":         "James Webb Telescope Confirms the Oldest Galaxy Cluster in the Early Universe",
			"slug":          "james-webb-oldest-galaxy-cluster",
			"category_slug": "science",
			"excerpt":       "Structures dating to just 350 million years after the Big Bang are rewriting models of the universe's first star formation.",
			"body":          "Infrared spectrometer data showed that the first galaxies gained mass much faster than assumed and carry richer metal traces than expected.",
			"image_url":     "https://images.unsplash.com/photo-1451187580459-43490279c0fa?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Prof. Erdem Kaya",
			"reading_time":  6,
			"published_at":  "Yesterday 12:00",
			"is_lead":       false,
			"is_trending":   true,
		},
		{
			"title":         "The 1 Nanometer Threshold Is Crossed in the Semiconductor Industry: A New Transistor Geometry",
			"slug":          "semiconductor-1nm-threshold",
			"category_slug": "technology",
			"excerpt":       "A gate-all-around architecture that overcomes quantum tunneling barriers cuts energy use by 40% while doubling processing power.",
			"body":          "Chipmakers are preparing next-generation mobile and data center processors for volume production using extreme ultraviolet (EUV) lithography with atomic-level precision.",
			"image_url":     "https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Deniz Aksoy",
			"reading_time":  4,
			"published_at":  "2 days ago",
			"is_lead":       false,
			"is_trending":   false,
		},
		{
			"title":         "The Solid-State Battery Revolution in Green Energy: 1000 km Range for Electric Vehicles",
			"slug":          "solid-state-battery-revolution",
			"category_slug": "science",
			"excerpt":       "Ceramic solid-state batteries that eliminate the risk of liquid electrolytes are transforming the industry with 80% charging in 10 minutes.",
			"body":          "Prototypes developed in partnership between automotive giants and research institutes showed a 2000-cycle lifespan with no range loss even in freezing cold.",
			"image_url":     "https://images.unsplash.com/photo-1466611653911-95081537e5b7?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Murat Çelik",
			"reading_time":  4,
			"published_at":  "3 days ago",
			"is_lead":       false,
			"is_trending":   true,
		},
		{
			"title":         "Quantum Cryptography and the New Cybersecurity: QKD Networks Go Live",
			"slug":          "quantum-cryptography-new-cybersecurity",
			"category_slug": "technology",
			"excerpt":       "Quantum key distribution (QKD) protocols are creating an unbreakable encryption standard for financial transactions and critical infrastructure.",
			"body":          "Encryption methods based on the quantum states of photons are being deployed as the strongest shield against the security threats of future quantum computers.",
			"image_url":     "https://images.unsplash.com/photo-1563986768609-322da13575f3?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Deniz Aksoy",
			"reading_time":  5,
			"published_at":  "3 days ago",
			"is_lead":       false,
			"is_trending":   false,
		},
		{
			"title":         "A New Route for Global Supply Chains: Regional Manufacturing Corridors",
			"slug":          "new-route-global-supply-chains",
			"category_slug": "economy",
			"excerpt":       "Multinational manufacturers aiming to reduce dependence on a single hub have committed 50 billion dollars to regional logistics corridors.",
			"body":          "Swings in logistics costs and geopolitical fragility have accelerated nearshoring strategies. Eastern Europe and the Mediterranean basin are turning into strategic manufacturing hubs.",
			"image_url":     "https://images.unsplash.com/photo-1586528116311-ad8dd3c8310d?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Murat Çelik",
			"reading_time":  4,
			"published_at":  "4 days ago",
			"is_lead":       false,
			"is_trending":   true,
		},
		{
			"title":         "AI and Contemporary Art: The Human Signature Meets Algorithmic Aesthetics",
			"slug":          "ai-and-contemporary-art",
			"category_slug": "culture",
			"excerpt":       "Next-generation generative installations shown at the Venice Biennale are redefining the artist's intent and algorithmic creativity.",
			"body":          "Art critics note that we are entering a contemporary era in which AI is accepted not as a substitute but as a new layer of expression, like brush and canvas.",
			"image_url":     "https://images.unsplash.com/photo-1579783902614-a3fb3927b675?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Selin Vural",
			"reading_time":  3,
			"published_at":  "5 days ago",
			"is_lead":       false,
			"is_trending":   false,
		},
		{
			"title":         "Mars Sample Return Mission: Next-Generation Robotic Arm Tests Completed",
			"slug":          "mars-sample-return-mission",
			"category_slug": "science",
			"excerpt":       "Critical firing tests of the rocket system that will bring rock and atmosphere samples from the Red Planet to Earth have concluded successfully.",
			"body":          "The sample encapsulation mechanism, tested in high-vacuum chambers, has reached the precision needed to protect microscopic fossil traces.",
			"image_url":     "https://images.unsplash.com/photo-1614728894747-a83421e2b9c9?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Prof. Erdem Kaya",
			"reading_time":  5,
			"published_at":  "5 days ago",
			"is_lead":       false,
			"is_trending":   true,
		},
		{
			"title":         "Carbon-Neutral Transport in Smart Cities: Micromobility and Autonomous Lines",
			"slug":          "carbon-neutral-smart-city-transport",
			"category_slug": "headlines",
			"excerpt":       "Shared electric service fleets that reduce fossil-fuel private car traffic in metropolises have started operating in pilot districts.",
			"body":          "Air quality sensors in city centers documented a 28% drop in nitrogen dioxide levels thanks to pedestrianized smart corridors.",
			"image_url":     "https://images.unsplash.com/photo-1519641471654-76ce0107ad1b?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Deniz Aksoy",
			"reading_time":  4,
			"published_at":  "6 days ago",
			"is_lead":       false,
			"is_trending":   false,
		},
		{
			"title":         "New Funding Models and Liquidity in Deep Tech Startups",
			"slug":          "deep-tech-startup-funding-models",
			"category_slug": "economy",
			"excerpt":       "Corporate venture capital funds targeting early-stage AI and semiconductor companies raised record amounts.",
			"body":          "Capital flows into strategic hardware technologies that need long-term R&D are becoming a decisive force in the global innovation race.",
			"image_url":     "https://images.unsplash.com/photo-1559526324-4b87b5e36e44?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Murat Çelik",
			"reading_time":  5,
			"published_at":  "1 week ago",
			"is_lead":       false,
			"is_trending":   false,
		},
	}

	return settings, categories, articles
}

// SeedNewsStore provisions schemas and initial data for the news archetype.
func SeedNewsStore(ctx context.Context, colSvc collections.Service, scope tenant.Scope, name, phone, prompt string) error {
	tenantCtx := tenant.WithScope(ctx, scope)
	settings, categories, articles := GenerateNewsCatalog(name, phone, prompt)

	if _, err := colSvc.CreateRecord(tenantCtx, "site_settings", settings); err != nil {
		return fmt.Errorf("seed news site_settings: %w", err)
	}
	for _, cat := range categories {
		if _, err := colSvc.CreateRecord(tenantCtx, "news_categories", cat); err != nil {
			return fmt.Errorf("seed news category %q: %w", cat["name"], err)
		}
	}
	for _, art := range articles {
		if _, err := colSvc.CreateRecord(tenantCtx, "news_articles", art); err != nil {
			return fmt.Errorf("seed news article %q: %w", art["title"], err)
		}
	}

	// Seed Columnists
	colData := []map[string]any{
		{"name": "Prof. Melih Arslan", "title": "Senior Chief Economist", "latest_article_title": "The Global Rate-Cut Cycle and the Direction of Capital Flows to Emerging Markets", "latest_article_slug": "kuresel-faiz-indirim-dongusu", "sort_order": 1},
		{"name": "Selin Doğan", "title": "Technology & Innovation Desk", "latest_article_title": "The Future of AI and Software Teams: Code Writers or System Architects?", "latest_article_slug": "age-of-autonomous-software-agents", "sort_order": 2},
		{"name": "Barış Yılmaz", "title": "Global Security & Foreign Policy", "latest_article_title": "The Critical Raw Materials Race: Rare Earth Elements and the Supply Chain", "latest_article_slug": "kritik-hammadde-savasi", "sort_order": 3},
		{"name": "Zeynep Erdem", "title": "Culture, Architecture & Urban Desk", "latest_article_title": "Digital Nomadism and the Spatial Transformation Dynamics of Metropolises", "latest_article_slug": "dijital-gocebelik-ve-metropoller", "sort_order": 4},
	}
	for _, c := range colData {
		_, _ = colSvc.CreateRecord(tenantCtx, "columnists", c)
	}

	// Seed Poll
	_, _ = colSvc.CreateRecord(tenantCtx, "polls", map[string]any{
		"question":    "Should training AI models on copyrighted data be tied to a global licensing requirement?",
		"options":     "Yes, it must be mandatory, No, open research should stay free, Undecided, it depends on the details",
		"category":    "Technology & Law",
		"total_votes": 1420,
		"is_active":   true,
	})

	// Seed Galleries
	galleries := []map[string]any{
		{"title": "The Clean-Room Manufacturing Process of Quantum Processors", "slug": "age-of-autonomous-software-agents", "category": "Technology Dossier", "cover_image": "https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=800&q=80", "badge_text": "12 Photos"},
		{"title": "Early-Universe Galaxies Discovered with the James Webb Telescope", "slug": "age-of-autonomous-software-agents", "category": "Space & Science", "cover_image": "https://images.unsplash.com/photo-1451187580459-43490279c0fa?auto=format&fit=crop&w=800&q=80", "badge_text": "Special Report"},
		{"title": "Next-Generation Solid-State Batteries and Battery Factories", "slug": "solid-state-battery-revolution", "category": "Energy & Mobility", "cover_image": "https://images.unsplash.com/photo-1466611653911-95081537e5b7?auto=format&fit=crop&w=800&q=80", "badge_text": "Research"},
	}
	for _, g := range galleries {
		_, _ = colSvc.CreateRecord(tenantCtx, "galleries", g)
	}

	return nil
}
