package blueprints

import (
	"context"
	"fmt"
	"strings"

	"github.com/hydrohq/hydrogo/internal/collections"
	"github.com/hydrohq/hydrogo/internal/contenttypes"
	"github.com/hydrohq/hydrogo/internal/tenant"
)

// BoutiqueDefinitions returns the core collections needed for a luxury boutique store.
func BoutiqueDefinitions() []contenttypes.Definition {
	return []contenttypes.Definition{
		{
			ID:          "store_settings",
			Label:       "Store and Appearance Settings",
			Description: "Boutique store identity, color palette, typography and quick-order settings.",
			Group:       "Store Management",
			Icon:        "palette",
			TitleField:  "brand_name",
			Public:      true,
			Revisions:   true,
			Singleton:   true,
			Fields: []contenttypes.Field{
				{ID: "brand_name", Label: "Business Name", Kind: "text", Required: true, List: true},
				{ID: "tagline", Label: "Slogan / Subtitle", Kind: "text", List: true},
				{ID: "primary_color", Label: "Primary Accent Color (Hex)", Kind: "text"},
				{ID: "bg_tone", Label: "Background Tone (Hex)", Kind: "text"},
				{ID: "whatsapp_number", Label: "WhatsApp Order Line", Kind: "text", Required: true, List: true},
				{ID: "shopier_url", Label: "Payment Link", Kind: "text"},
				{ID: "payment_link", Label: "Payment URL", Kind: "text"},
				{ID: "announcement_bar", Label: "Announcement Bar", Kind: "text"},
				{ID: "layout_mode", Label: "Storefront Page Layout", Kind: "select", Options: []string{"hybrid", "single_page", "multipage"}},
				{ID: "font_preset", Label: "Typography & Font Palette", Kind: "select", Options: []string{"v0_geist", "editorial_playfair", "artisan_garamond", "monumental_cinzel", "avantgarde_syne", "modern_grotesk"}},
				{ID: "site_type", Label: "Site Type", Kind: "text"},
				{ID: "archetype", Label: "Archetype", Kind: "text"},
				// Cross-compatibility fields with existing boutique theme engine
				{ID: "store_name", Label: "Store Name", Kind: "text"},
				{ID: "slogan", Label: "Slogan", Kind: "text"},
				{ID: "accent_color", Label: "Accent Color", Kind: "text"},
				{ID: "bg_primary_color", Label: "Background", Kind: "text"},
				{ID: "theme_preset", Label: "Theme Template", Kind: "select", Options: []string{"minimal_cream", "noir_gold", "pastel_blush", "emerald_silk"}},
				{ID: "font_heading", Label: "Heading Font", Kind: "text"},
				{ID: "border_radius_style", Label: "Corner Style", Kind: "text"},
				{ID: "hero_badge", Label: "Hero Badge", Kind: "text"},
				{ID: "hero_title", Label: "Hero Title", Kind: "text"},
				{ID: "hero_subtitle", Label: "Hero Subtitle", Kind: "textarea"},
				{ID: "hero_image", Label: "Hero Image", Kind: "text", Widget: "media"},
				{ID: "story_badge", Label: "Story Badge", Kind: "text"},
				{ID: "story_title", Label: "Story Title", Kind: "text"},
				{ID: "story_text", Label: "Story Text", Kind: "textarea"},
				{ID: "story_quote", Label: "Story Quote", Kind: "text"},
				{ID: "story_image", Label: "Story Image", Kind: "text", Widget: "media"},
				{ID: "pillars_title", Label: "Principles Title", Kind: "text"},
				{ID: "pillars_json", Label: "Principles (JSON)", Kind: "textarea"},
				{ID: "testimonials_json", Label: "Customer Reviews (JSON)", Kind: "textarea"},
				{ID: "faqs_json", Label: "FAQs (JSON)", Kind: "textarea"},
				{ID: "atelier_address", Label: "Workshop Address", Kind: "text"},
				{ID: "atelier_hours", Label: "Opening Hours", Kind: "text"},
				{ID: "instagram", Label: "Instagram Account", Kind: "text"},
				{ID: "layout_json", Label: "Custom Section Layout (JSON)", Kind: "textarea"},
			},
		},
		{
			ID:          "products",
			Label:       "Products",
			Description: "Boutique product catalog, prices, variants and stock status.",
			Group:       "Store Management",
			Icon:        "shopping-bag",
			TitleField:  "title",
			Public:      true,
			Drafts:      true,
			Revisions:   true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Product Title", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "slug", Label: "URL Slug", Kind: "text", Required: true, Unique: true, List: true, Filterable: true},
				{ID: "price", Label: "Price (TRY)", Kind: "number", Decimals: intPtr(2), Min: floatPtr(0), Required: true, List: true, Filterable: true},
				{ID: "compare_at_price", Label: "Compare-at Price (TRY)", Kind: "number", Decimals: intPtr(2), Min: floatPtr(0), List: true},
				{ID: "description", Label: "Description", Kind: "richtext", Searchable: true},
				{ID: "image_url", Label: "Image", Kind: "text", Widget: "media", List: true},
				{ID: "secondary_image", Label: "Second Image (Hover)", Kind: "text", Widget: "media"},
				{ID: "cover_image", Label: "Cover Image", Kind: "media"},
				{ID: "options_json", Label: "Options (JSON)", Kind: "textarea"},
				{ID: "variants_json", Label: "Variants (JSON)", Kind: "textarea"},
				{ID: "sizes", Label: "Sizes (S, M, L)", Kind: "text", List: true},
				{ID: "colors", Label: "Colors", Kind: "text", List: true},
				{ID: "in_stock", Label: "In Stock", Kind: "boolean", List: true, Filterable: true},
				{ID: "stock_quantity", Label: "Stock Quantity", Kind: "number", Min: floatPtr(0), List: true},
				{ID: "badge", Label: "Badge", Kind: "text", List: true, Filterable: true},
				{ID: "payment_link", Label: "Quick Buy Link", Kind: "text"},
			},
		},
		{
			ID:          "orders",
			Label:       "Orders",
			Description: "Customer orders placed through WhatsApp and credit card.",
			Group:       "Store Management",
			Icon:        "package",
			TitleField:  "order_code",
			Public:      false,
			Revisions:   true,
			Fields: []contenttypes.Field{
				{ID: "order_code", Label: "Order Code", Kind: "text", Sequence: &collections.Sequence{Prefix: "ORD-", Pad: 5}, List: true},
				{ID: "customer_name", Label: "Customer Full Name", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "customer_phone", Label: "Phone", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "customer_address", Label: "Delivery Address", Kind: "textarea", Required: true},
				{ID: "items_json", Label: "Order Items", Kind: "textarea", Required: true},
				{ID: "total_price", Label: "Total Amount (TRY)", Kind: "number", Decimals: intPtr(2), Min: floatPtr(0), Required: true, List: true},
				{ID: "status", Label: "Status", Kind: "select", Options: []string{"PENDING_WHATSAPP", "new", "confirmed", "shipped", "completed", "cancelled"}, List: true, Filterable: true},
				// Cross-compatibility fields with boutique_orders
				{ID: "order_no", Label: "Order No.", Kind: "text"},
				{ID: "shipping_address", Label: "Address", Kind: "textarea"},
				{ID: "items_summary", Label: "Items", Kind: "textarea"},
				{ID: "total_amount", Label: "Amount", Kind: "number"},
				{ID: "order_channel", Label: "Channel", Kind: "text"},
			},
		},
	}
}

// EnsureBoutiqueSchemas creates or updates the 3 core collections for the given tenant scope.
func EnsureBoutiqueSchemas(ctx context.Context, colSvc collections.Service, scope tenant.Scope) error {
	defs := BoutiqueDefinitions()
	schemas, err := contenttypes.Schemas(defs)
	if err != nil {
		return fmt.Errorf("compile boutique schemas: %w", err)
	}

	tenantCtx := tenant.WithScope(ctx, scope)
	if err := colSvc.SyncSchemas(tenantCtx, schemas); err != nil {
		return fmt.Errorf("sync boutique collections: %w", err)
	}
	return nil
}

func intPtr(i int) *int {
	return &i
}

func floatPtr(f float64) *float64 {
	return &f
}

// GenerateLuxuryCatalog generates store settings and 6 realistic luxury boutique products from a short prompt.
func GenerateLuxuryCatalog(displayName, phone, prompt string) (map[string]any, []map[string]any) {
	lower := strings.ToLower(prompt)

	brandName := strings.TrimSpace(displayName)
	if brandName == "" {
		brandName = "Vera Atelier"
	}
	cleanPhone := strings.TrimSpace(phone)
	if cleanPhone == "" {
		cleanPhone = "+905550000000"
	}

	primaryColor := "#8C6D46" // Warm bronze
	bgTone := "#FAF8F5"       // Luxury cream
	preset := "minimal_cream"

	if strings.Contains(lower, "siyah") || strings.Contains(lower, "noir") || strings.Contains(lower, "koyu") {
		primaryColor = "#D4AF37" // Gold
		bgTone = "#121212"
		preset = "noir_gold"
	} else if strings.Contains(lower, "yeşil") || strings.Contains(lower, "zümrüt") || strings.Contains(lower, "emerald") {
		primaryColor = "#1B5E43"
		bgTone = "#F4F8F6"
		preset = "emerald_silk"
	} else if strings.Contains(lower, "pembe") || strings.Contains(lower, "blush") || strings.Contains(lower, "deri") {
		primaryColor = "#9B5D3D" // Leather tan / chestnut
		bgTone = "#FAF6F0"
	}

	tagline := "Timeless Elegance & Distinguished Designs"
	announcement := "Free Shipping on Orders Over 1500 TRY · Quick Order via WhatsApp"

	settings := map[string]any{
		"brand_name":       brandName,
		"store_name":       brandName,
		"tagline":          tagline,
		"slogan":           tagline,
		"primary_color":    primaryColor,
		"accent_color":     primaryColor,
		"bg_tone":          bgTone,
		"bg_primary_color": bgTone,
		"whatsapp_number":  cleanPhone,
		"shopier_url":      "https://example.com/pay/" + strings.ToLower(strings.ReplaceAll(brandName, " ", "")),
		"payment_link":     "https://example.com/pay/" + strings.ToLower(strings.ReplaceAll(brandName, " ", "")),
		"announcement_bar": announcement,
		"theme_preset":     preset,
		"font_heading":     "Cormorant Garamond",
	}

	// Enrich with deep artisanal storytelling, pillars, testimonials and FAQs
	editorial := GetNicheEditorialContent(brandName, prompt)
	editorial.MergeIntoSettings(settings)

	// 6 Realistic Luxury Products
	isLeather := strings.Contains(lower, "deri") || strings.Contains(lower, "çanta") || strings.Contains(lower, "cüzdan")
	isCeramic := strings.Contains(lower, "seramik") || strings.Contains(lower, "tabak") || strings.Contains(lower, "fincan")

	var products []map[string]any

	if isLeather {
		products = []map[string]any{
			{
				"title":            "Handmade Genuine Leather Cardholder & Wallet",
				"slug":             "handmade-leather-cardholder",
				"price":            850.0,
				"compare_at_price": 1100.0,
				"description":      "A minimalist, long-lasting cardholder hand-stitched with waxed thread from vegetable-tanned leather.",
				"image_url":        "https://images.unsplash.com/photo-1627123424574-724758594e93?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Tan", "Chestnut", "Black"]}`,
				"sizes":            "Standard",
				"colors":           "Tan, Chestnut, Black",
				"in_stock":         true,
				"stock_quantity":   15,
				"badge":            "New Season",
			},
			{
				"title":            "Vintage Messenger Bag (Genuine Leather)",
				"slug":             "vintage-messenger-bag",
				"price":            3450.0,
				"compare_at_price": 4200.0,
				"description":      "A timeless classic in first-grade Tuscan leather with brass buckles and an adjustable shoulder strap.",
				"image_url":        "https://images.unsplash.com/photo-1548036328-c9fa89d128fa?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Tan", "Antique Brown"]}`,
				"sizes":            "Standard (38x28 cm)",
				"colors":           "Tan, Antique Brown",
				"in_stock":         true,
				"stock_quantity":   8,
				"badge":            "Signature Design",
			},
			{
				"title":            "Minimalist Leather Keychain & AirTag Case",
				"slug":             "leather-keychain-airtag",
				"price":            450.0,
				"compare_at_price": 600.0,
				"description":      "Turn your keys into a stylish accessory with a stainless steel ring and an elegant leather body.",
				"image_url":        "https://images.unsplash.com/photo-1606760227091-3dd870d97f1d?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Tan", "Emerald Green", "Navy"]}`,
				"sizes":            "Standard",
				"colors":           "Tan, Emerald Green, Navy",
				"in_stock":         true,
				"stock_quantity":   25,
				"badge":            "Popular",
			},
			{
				"title":            "Italian Leather Laptop Bag (14-16 in)",
				"slug":             "leather-laptop-bag",
				"price":            3950.0,
				"compare_at_price": 4800.0,
				"description":      "Shock-absorbing suede lining, zippered document pocket and handmade stitching details.",
				"image_url":        "https://images.unsplash.com/photo-1553062407-98eeb64c6a62?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"ebat": ["14 in", "16 in"], "renkler": ["Tan", "Black"]}`,
				"sizes":            "14 in, 16 in",
				"colors":           "Tan, Black",
				"in_stock":         true,
				"stock_quantity":   6,
				"badge":            "Last 3 Items",
			},
			{
				"title":            "Classic Leather Belt (Brass Buckle)",
				"slug":             "classic-leather-belt",
				"price":            950.0,
				"compare_at_price": 1250.0,
				"description":      "Solid brass buckle and 3.8 cm wide full-grain Italian cowhide.",
				"image_url":        "https://images.unsplash.com/photo-1624222247344-550fb60583dc?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"beden": ["90 cm", "100 cm", "110 cm"], "renk": ["Brown", "Black"]}`,
				"sizes":            "90 cm, 100 cm, 110 cm",
				"colors":           "Brown, Black",
				"in_stock":         true,
				"stock_quantity":   18,
				"badge":            "Classic",
			},
			{
				"title":            "Leather Travel Notebook & Passport Cover",
				"slug":             "leather-passport-cover",
				"price":            750.0,
				"compare_at_price": 950.0,
				"description":      "Accompanies your travels with a refillable notebook insert, passport slot and pen holder.",
				"image_url":        "https://images.unsplash.com/photo-1544816155-12df9643f363?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Tan", "Burgundy", "Dark Green"]}`,
				"sizes":            "Standard",
				"colors":           "Tan, Burgundy, Dark Green",
				"in_stock":         true,
				"stock_quantity":   20,
				"badge":            "Gift Idea",
			},
		}
	} else if isCeramic {
		products = []map[string]any{
			{
				"title":            "Handmade Stoneware Mug (Cream & Terracotta)",
				"slug":             "stoneware-mug",
				"price":            420.0,
				"compare_at_price": 550.0,
				"description":      "Durable stoneware clay fired at 1250 degrees with a food-safe matte glaze.",
				"image_url":        "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Cream", "Terracotta"]}`,
				"sizes":            "250 ml",
				"colors":           "Cream, Terracotta",
				"in_stock":         true,
				"stock_quantity":   20,
				"badge":            "Best Seller",
			},
			{
				"title":            "Organic-Form Ceramic Vase (30 cm)",
				"slug":             "organic-ceramic-vase",
				"price":            1450.0,
				"compare_at_price": 1850.0,
				"description":      "A sculptural, waterproof decorative vase for dried or fresh flowers.",
				"image_url":        "https://images.unsplash.com/photo-1612196808214-b8e1d6145a8c?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"ebat": ["30 cm"]}`,
				"sizes":            "30 cm",
				"colors":           "Raw Sand White",
				"in_stock":         true,
				"stock_quantity":   7,
				"badge":            "Signature Design",
			},
			{
				"title":            "Matte Black Espresso Cup Set (2 pcs)",
				"slug":             "espresso-cup-set",
				"price":            580.0,
				"compare_at_price": 750.0,
				"description":      "Two hand-thrown espresso cups that come with minimalist wooden coasters.",
				"image_url":        "https://images.unsplash.com/photo-1577937927133-66ef06acdf18?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Matte Black", "Speckled Cream"]}`,
				"sizes":            "90 ml (2 pcs)",
				"colors":           "Matte Black, Speckled Cream",
				"in_stock":         true,
				"stock_quantity":   12,
				"badge":            "New Season",
			},
			{
				"title":            "Natural-Edge Ceramic Serving Plate",
				"slug":             "ceramic-serving-plate",
				"price":            750.0,
				"compare_at_price": 950.0,
				"description":      "A handmade rustic plate for special dinner tables and cheese boards.",
				"image_url":        "https://images.unsplash.com/photo-1578749556568-bc2c40e68b61?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"cap": ["26 cm", "32 cm"]}`,
				"sizes":            "26 cm, 32 cm",
				"colors":           "Earthy Texture",
				"in_stock":         true,
				"stock_quantity":   9,
				"badge":            "Signature Design",
			},
			{
				"title":            "Soy Candle in Ceramic Bowl (Lavender & Cedar)",
				"slug":             "soy-candle-ceramic",
				"price":            490.0,
				"compare_at_price": 620.0,
				"description":      "100 percent natural soy wax, a wooden wick and a smoky cedar scent.",
				"image_url":        "https://images.unsplash.com/photo-1603006905003-be475563bc59?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"koku": ["Lavender & Cedar", "Vanilla & Amber"]}`,
				"sizes":            "300 gr",
				"colors":           "Earthen Bowl",
				"in_stock":         true,
				"stock_quantity":   25,
				"badge":            "Gift Idea",
			},
			{
				"title":            "Footed Fruit & Salad Bowl",
				"slug":             "footed-ceramic-fruit-bowl",
				"price":            1850.0,
				"compare_at_price": 2300.0,
				"description":      "A large handmade footed bowl that adds a sophisticated touch to your kitchen island.",
				"image_url":        "https://images.unsplash.com/photo-1590402494587-44b71d7772f6?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Speckled Beige", "Sage Green"]}`,
				"sizes":            "Width 28 cm",
				"colors":           "Speckled Beige, Sage Green",
				"in_stock":         true,
				"stock_quantity":   5,
				"badge":            "Last 2 Items",
			},
		}
	} else {
		// Default luxury apparel & accessories
		products = []map[string]any{
			{
				"title":            "Pure Silk Double-Breasted Shirt",
				"slug":             "pure-silk-double-breasted-shirt",
				"price":            1850.0,
				"compare_at_price": 2300.0,
				"description":      "Naturally draped pure silk fabric with mother-of-pearl buttons and elegant cuff details.",
				"image_url":        "https://images.unsplash.com/photo-1598554747436-c9293d6a588f?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"beden": ["S", "M", "L"], "renkler": ["Ecru", "Champagne", "Black"]}`,
				"sizes":            "S, M, L",
				"colors":           "Ecru, Champagne, Black",
				"in_stock":         true,
				"stock_quantity":   14,
				"badge":            "New Season",
			},
			{
				"title":            "High-Waist Natural Linen Palazzo Trousers",
				"slug":             "linen-palazzo-trousers",
				"price":            1490.0,
				"compare_at_price": 1850.0,
				"description":      "All-summer comfort from breathable natural linen and a relaxed fit.",
				"image_url":        "https://images.unsplash.com/photo-1509631179647-0177331693ae?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"beden": ["S", "M", "L", "XL"], "renkler": ["Sand Beige", "Stone", "Navy"]}`,
				"sizes":            "S, M, L, XL",
				"colors":           "Sand Beige, Stone, Navy",
				"in_stock":         true,
				"stock_quantity":   20,
				"badge":            "Signature Design",
			},
			{
				"title":            "Satin Midi Dress with Back Detail",
				"slug":             "satin-midi-dress",
				"price":            2450.0,
				"compare_at_price": 2950.0,
				"description":      "A draped satin dress designed for special invitations and evening events.",
				"image_url":        "https://images.unsplash.com/photo-1572804013309-59a88b7e92f1?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"beden": ["S", "M", "L"], "renkler": ["Emerald", "Black", "Burgundy"]}`,
				"sizes":            "S, M, L",
				"colors":           "Emerald, Black, Burgundy",
				"in_stock":         true,
				"stock_quantity":   6,
				"badge":            "Last 3 Items",
			},
			{
				"title":            "Oversized Cashmere-Blend Knit Cardigan",
				"slug":             "cashmere-knit-cardigan",
				"price":            1950.0,
				"compare_at_price": 2400.0,
				"description":      "A soft cashmere blend with horn buttons and a relaxed cut.",
				"image_url":        "https://images.unsplash.com/photo-1434389677669-e08b4cac3105?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"beden": ["Standard (S-L)"], "renkler": ["Mink", "Cream", "Anthracite"]}`,
				"sizes":            "Standard (S-L)",
				"colors":           "Mink, Cream, Anthracite",
				"in_stock":         true,
				"stock_quantity":   11,
				"badge":            "Most Loved",
			},
			{
				"title":            "Linen Belted Midi Kimono Jacket",
				"slug":             "linen-belted-kimono",
				"price":            1650.0,
				"compare_at_price": 1950.0,
				"description":      "Wide sleeves and a removable belt make this an elegant piece for both the beach and the city.",
				"image_url":        "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"beden": ["Standard"], "renkler": ["Ecru", "Olive Green", "Tan"]}`,
				"sizes":            "Standard",
				"colors":           "Ecru, Olive Green, Tan",
				"in_stock":         true,
				"stock_quantity":   15,
				"badge":            "New Season",
			},
			{
				"title":            "Handmade Leather & Canvas Shoulder Bag",
				"slug":             "leather-canvas-shoulder-bag",
				"price":            2150.0,
				"compare_at_price": 2600.0,
				"description":      "A timeless city bag with a heavyweight cotton canvas body and genuine leather straps.",
				"image_url":        "https://images.unsplash.com/photo-1544816155-12df9643f363?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Natural/Tan", "Black/Black"]}`,
				"sizes":            "Roomy",
				"colors":           "Natural/Tan, Black/Black",
				"in_stock":         true,
				"stock_quantity":   8,
				"badge":            "Signature Design",
			},
		}
	}

	return settings, products
}

// SeedBoutiqueStore seeds store_settings and products into collections for a tenant.
func SeedBoutiqueStore(ctx context.Context, colSvc collections.Service, scope tenant.Scope, settings map[string]any, products []map[string]any) error {
	tenantCtx := tenant.WithScope(ctx, scope)

	// 1. Seed store_settings
	if _, err := colSvc.CreateRecord(tenantCtx, "store_settings", settings); err != nil {
		return fmt.Errorf("seed store_settings (%v): %w", err, err)
	}

	// 2. Seed products
	for _, prod := range products {
		prodCopy := make(map[string]any, len(prod)+1)
		for k, v := range prod {
			prodCopy[k] = v
		}
		prodCopy["status"] = contenttypes.StatusPublished
		if _, err := colSvc.CreateRecord(tenantCtx, "products", prodCopy); err != nil {
			return fmt.Errorf("seed product %q: %w", prod["title"], err)
		}
	}
	return nil
}
