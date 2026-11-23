package blueprints

import (
	"context"
	"fmt"
	"strings"

	"github.com/hydrohq/hydrogo/internal/collections"
	"github.com/hydrohq/hydrogo/internal/contenttypes"
	"github.com/hydrohq/hydrogo/internal/tenant"
)

// RestaurantDefinitions returns the collections needed for a modern restaurant, cafe & QR menu site.
func RestaurantDefinitions() []contenttypes.Definition {
	return []contenttypes.Definition{
		{
			ID:          "site_settings",
			Label:       "Restaurant & Menu Settings",
			Description: "Venue details, opening hours, location and theme colors.",
			Group:       "Restaurant Management",
			Icon:        "utensils",
			TitleField:  "brand_name",
			Public:      true,
			Singleton:   true,
			Fields: []contenttypes.Field{
				{ID: "brand_name", Label: "Restaurant / Cafe Name", Kind: "text", Required: true, List: true},
				{ID: "tagline", Label: "Tagline / Concept", Kind: "text", List: true},
				{ID: "primary_color", Label: "Primary Accent Color", Kind: "text"},
				{ID: "bg_tone", Label: "Background Tone", Kind: "text"},
				{ID: "font_preset", Label: "Typography Palette", Kind: "select", Options: []string{"v0_geist", "editorial_playfair", "artisan_garamond", "monumental_cinzel", "avantgarde_syne", "modern_grotesk"}},
				{ID: "phone", Label: "Phone Number", Kind: "text", List: true},
				{ID: "whatsapp_number", Label: "WhatsApp Line", Kind: "text", List: true},
				{ID: "address", Label: "Address / Location", Kind: "text"},
				{ID: "opening_hours", Label: "Opening Hours", Kind: "text"},
				{ID: "hero_badge", Label: "Hero Badge", Kind: "text"},
				{ID: "hero_title", Label: "Hero Title", Kind: "text"},
				{ID: "hero_subtitle", Label: "Hero Subtext", Kind: "textarea"},
				{ID: "hero_image", Label: "Hero Image", Kind: "text", Widget: "media"},
			},
		},
		{
			ID:          "menu_categories",
			Label:       "Menu Categories",
			Description: "Starters, Main Courses, Drinks, etc.",
			Group:       "Menu Management",
			Icon:        "list",
			TitleField:  "name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "name", Label: "Category Name", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "slug", Label: "URL Slug", Kind: "text", Required: true, Unique: true, List: true, Filterable: true},
				{ID: "sort_order", Label: "Sort Order", Kind: "number", List: true},
				{ID: "description", Label: "Short Description", Kind: "text"},
			},
		},
		{
			ID:          "menu_items",
			Label:       "Menu Items (Food & Drink)",
			Description: "Dishes, prices, ingredients and allergen information.",
			Group:       "Menu Management",
			Icon:        "coffee",
			TitleField:  "name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "name", Label: "Dish / Drink Name", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "category_slug", Label: "Category", Kind: "text", Widget: "relation", To: "menu_categories", ValueField: "slug", Required: true, List: true, Filterable: true},
				{ID: "price", Label: "Price (TRY)", Kind: "number", Decimals: intPtr(2), Required: true, List: true},
				{ID: "description", Label: "Ingredients & Presentation Description", Kind: "textarea", List: true, Searchable: true},
				{ID: "image_url", Label: "Image", Kind: "text", Widget: "media", List: true},
				{ID: "is_chef_special", Label: "Chef's Signature", Kind: "boolean", List: true, Filterable: true},
				{ID: "is_vegetarian", Label: "Vegetarian", Kind: "boolean", List: true, Filterable: true},
				{ID: "is_gluten_free", Label: "Gluten-Free", Kind: "boolean", List: true, Filterable: true},
				{ID: "is_available", Label: "Available for Service", Kind: "boolean", List: true, Filterable: true},
			},
		},
		{
			ID:          "reservations",
			Label:       "Table Reservations",
			Description: "Online reservation requests from the website.",
			Group:       "Reservations",
			Icon:        "calendar",
			TitleField:  "guest_name",
			Public:      false,
			Fields: []contenttypes.Field{
				{ID: "guest_name", Label: "Guest Full Name", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "phone", Label: "Phone", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "email", Label: "Email", Kind: "text", List: true, Searchable: true},
				{ID: "party_size", Label: "Party Size", Kind: "number", Required: true, List: true},
				{ID: "reservation_date", Label: "Date", Kind: "date", Required: true, List: true, Filterable: true},
				{ID: "reservation_time", Label: "Time", Kind: "text", Required: true, List: true},
				{ID: "special_requests", Label: "Special Request / Note", Kind: "textarea"},
				{ID: "status", Label: "Status", Kind: "select", Options: []string{"pending", "confirmed", "completed", "cancelled"}, List: true, Filterable: true},
			},
		},
	}
}

// GenerateRestaurantCatalog generates default restaurant and QR menu data.
func GenerateRestaurantCatalog(name, phone, prompt string) (map[string]any, []map[string]any, []map[string]any) {
	brandName := strings.TrimSpace(name)
	if brandName == "" {
		brandName = "Bistro Soleil & Lounge"
	}
	cleanPhone := strings.TrimSpace(phone)
	if cleanPhone == "" {
		cleanPhone = "+90 212 284 1020"
	}

	settings := map[string]any{
		"brand_name":      brandName,
		"tagline":         "Mediterranean-Inspired Gourmet Cuisine & Cocktails",
		"primary_color":   "#B45309", // Warm Amber
		"bg_tone":         "#18181B", // Dark Zinc / Night Lounge
		"font_preset":     "artisan_garamond",
		"phone":           cleanPhone,
		"whatsapp_number": cleanPhone,
		"address":         "12 Harbor Street, Old Town",
		"opening_hours":   "Daily 11:30 - 01:30 (Kitchen Closes 23:30)",
		"hero_badge":      "Seasonal Local Flavors & QR Digital Menu",
		"hero_title":      "Contemporary Mediterranean Gastronomy in a Historic Setting",
		"hero_subtitle":   "Artisan dishes fresh from the stone oven, handmade pastas and signature cocktails from mixology experts.",
		"hero_image":      "https://images.unsplash.com/photo-1517248135467-4c7edcad34c4?auto=format&fit=crop&w=1200&q=80",
	}

	categories := []map[string]any{
		{"name": "Starters & Antipasti", "slug": "starters", "sort_order": 1, "description": "Hot and cold plates for sharing"},
		{"name": "Handmade Pasta & Risotto", "slug": "pasta", "sort_order": 2, "description": "Artisan doughs opened fresh every morning"},
		{"name": "Wood-Fired Mains", "slug": "mains", "sort_order": 3, "description": "Specially marinated meats and seafood"},
		{"name": "Signature Cocktails & Drinks", "slug": "cocktails", "sort_order": 4, "description": "Mixology creations prepared with special aromas"},
	}

	items := []map[string]any{
		{
			"name":            "Truffle Wild Mushroom Arancini",
			"category_slug":   "starters",
			"price":           380.0,
			"description":     "With porcini mushrooms, truffle paste, fior di latte mozzarella and saffron aioli.",
			"image_url":       "https://images.unsplash.com/photo-1541529086526-db283c563270?auto=format&fit=crop&w=600&q=80",
			"is_chef_special": true,
			"is_vegetarian":   true,
			"is_gluten_free":  false,
			"is_available":    true,
		},
		{
			"name":            "Burrata & Roasted Fig",
			"category_slug":   "starters",
			"price":           460.0,
			"description":     "Puglia burrata, caramelized fresh figs, walnut crunch, pomegranate molasses reduction and basil oil.",
			"image_url":       "https://images.unsplash.com/photo-1592417817098-8f3d69109853?auto=format&fit=crop&w=600&q=80",
			"is_chef_special": false,
			"is_vegetarian":   true,
			"is_gluten_free":  true,
			"is_available":    true,
		},
		{
			"name":            "Pappardelle al Ragù di Manzo",
			"category_slug":   "pasta",
			"price":           560.0,
			"description":     "Beef short rib ragu slow-cooked for 8 hours, 24-month aged parmesan and sage.",
			"image_url":       "https://images.unsplash.com/photo-1621996346565-e3d5d62810a9?auto=format&fit=crop&w=600&q=80",
			"is_chef_special": true,
			"is_vegetarian":   false,
			"is_gluten_free":  false,
			"is_available":    true,
		},
		{
			"name":            "Black Tagliolini with Seafood",
			"category_slug":   "pasta",
			"price":           640.0,
			"description":     "Fresh squid-ink pasta with jumbo prawns, calamari, garlic cherry tomatoes and fresh chili.",
			"image_url":       "https://images.unsplash.com/photo-1563379091339-03b21ab4a4f8?auto=format&fit=crop&w=600&q=80",
			"is_chef_special": false,
			"is_vegetarian":   false,
			"is_gluten_free":  false,
			"is_available":    true,
		},
		{
			"name":            "Grilled Sea Bass & Artichoke Purée",
			"category_slug":   "mains",
			"price":           720.0,
			"description":     "Crispy-skinned fresh sea bass, velvet artichoke purée, capers and lemon butter sauce.",
			"image_url":       "https://images.unsplash.com/photo-1519708227418-c8fd9a32b7a2?auto=format&fit=crop&w=600&q=80",
			"is_chef_special": true,
			"is_vegetarian":   false,
			"is_gluten_free":  true,
			"is_available":    true,
		},
		{
			"name":            "Soleil Botanica (Signature Cocktail)",
			"category_slug":   "cocktails",
			"price":           390.0,
			"description":     "Rosemary-infused gin, elderflower liqueur, fresh lime, cucumber and tonic.",
			"image_url":       "https://images.unsplash.com/photo-1551024709-8f23befc6f87?auto=format&fit=crop&w=600&q=80",
			"is_chef_special": true,
			"is_vegetarian":   true,
			"is_gluten_free":  true,
			"is_available":    true,
		},
	}

	return settings, categories, items
}

// SeedRestaurantStore provisions schemas and initial items for restaurant archetype.
func SeedRestaurantStore(ctx context.Context, colSvc collections.Service, scope tenant.Scope, name, phone, prompt string) error {
	tenantCtx := tenant.WithScope(ctx, scope)
	settings, categories, items := GenerateRestaurantCatalog(name, phone, prompt)

	if _, err := colSvc.CreateRecord(tenantCtx, "site_settings", settings); err != nil {
		return fmt.Errorf("seed restaurant site_settings: %w", err)
	}
	for _, cat := range categories {
		if _, err := colSvc.CreateRecord(tenantCtx, "menu_categories", cat); err != nil {
			return fmt.Errorf("seed menu_category %q: %w", cat["name"], err)
		}
	}
	for _, item := range items {
		if _, err := colSvc.CreateRecord(tenantCtx, "menu_items", item); err != nil {
			return fmt.Errorf("seed menu_item %q: %w", item["name"], err)
		}
	}
	return nil
}
