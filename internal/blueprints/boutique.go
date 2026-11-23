package blueprints

import (
	"context"
	"fmt"

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
