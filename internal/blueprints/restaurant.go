package blueprints

import (
	"github.com/hydrohq/hydrogo/internal/contenttypes"
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
