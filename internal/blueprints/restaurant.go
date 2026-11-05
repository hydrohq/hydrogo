package blueprints

import (
	"context"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// RestaurantDefinitions returns the collections needed for a modern restaurant, cafe & QR menu site.
func RestaurantDefinitions() []contenttypes.Definition {
	return []contenttypes.Definition{
		{
			ID:          "site_settings",
			Label:       "Restoran & Menü Ayarları",
			Description: "Mekan bilgileri, çalışma saatleri, konum ve tema renkleri.",
			Group:       "Restoran Yönetimi",
			Icon:        "utensils",
			TitleField:  "brand_name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "brand_name", Label: "Restoran / Kafe Adı", Kind: "text", Required: true, List: true},
				{ID: "tagline", Label: "Slogan / Konsept", Kind: "text", List: true},
				{ID: "primary_color", Label: "Ana Vurgu Rengi", Kind: "text"},
				{ID: "bg_tone", Label: "Arka Plan Tonu", Kind: "text"},
				{ID: "font_preset", Label: "Tipografi Paleti", Kind: "select", Options: []string{"v0_geist", "editorial_playfair", "artisan_garamond", "monumental_cinzel", "avantgarde_syne", "modern_grotesk"}},
				{ID: "phone", Label: "Telefon Numarası", Kind: "text", List: true},
				{ID: "whatsapp_number", Label: "WhatsApp Hattı", Kind: "text", List: true},
				{ID: "address", Label: "Adres / Konum", Kind: "text"},
				{ID: "opening_hours", Label: "Çalışma Saatleri", Kind: "text"},
				{ID: "hero_badge", Label: "Hero Rozet", Kind: "text"},
				{ID: "hero_title", Label: "Hero Başlık", Kind: "text"},
				{ID: "hero_subtitle", Label: "Hero Alt Metin", Kind: "textarea"},
				{ID: "hero_image", Label: "Hero Görseli", Kind: "text"},
			},
		},
		{
			ID:          "menu_categories",
			Label:       "Menü Kategorileri",
			Description: "Başlangıçlar, Ana Yemekler, İçecekler vb.",
			Group:       "Menü Yönetimi",
			Icon:        "list",
			TitleField:  "name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "name", Label: "Kategori Adı", Kind: "text", Required: true, List: true},
				{ID: "slug", Label: "URL Kodu (Slug)", Kind: "text", Required: true, Unique: true, List: true},
				{ID: "sort_order", Label: "Sıralama", Kind: "number", List: true},
				{ID: "description", Label: "Kısa Açıklama", Kind: "text"},
			},
		},
		{
			ID:          "menu_items",
			Label:       "Menü Kalemleri (Yiyecek & İçecek)",
			Description: "Yemekler, fiyatlar, içerikler ve alerjen bilgileri.",
			Group:       "Menü Yönetimi",
			Icon:        "coffee",
			TitleField:  "name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "name", Label: "Yemek / İçecek Adı", Kind: "text", Required: true, List: true},
				{ID: "category_slug", Label: "Kategori Kodu", Kind: "text", Required: true, List: true},
				{ID: "price", Label: "Fiyat (TL)", Kind: "number", Decimals: intPtr(2), Required: true, List: true},
				{ID: "description", Label: "İçerik & Sunum Açıklaması", Kind: "textarea", List: true},
				{ID: "image_url", Label: "Görsel URL", Kind: "text"},
				{ID: "is_chef_special", Label: "Şefin İmzası ⭐", Kind: "boolean", List: true},
				{ID: "is_vegetarian", Label: "Vejetaryen 🌱", Kind: "boolean", List: true},
				{ID: "is_gluten_free", Label: "Glutensiz 🌾", Kind: "boolean", List: true},
				{ID: "is_available", Label: "Serviste Mevcut", Kind: "boolean", List: true},
			},
		},
		{
			ID:          "reservations",
			Label:       "Masa Rezervasyonları",
			Description: "Web sitesinden gelen online rezervasyon talepleri.",
			Group:       "Rezervasyonlar",
			Icon:        "calendar",
			TitleField:  "guest_name",
			Public:      false,
			Fields: []contenttypes.Field{
				{ID: "guest_name", Label: "Misafir Ad Soyad", Kind: "text", Required: true, List: true},
				{ID: "phone", Label: "Telefon", Kind: "text", Required: true, List: true},
				{ID: "email", Label: "E-Posta", Kind: "text", List: true},
				{ID: "party_size", Label: "Kişi Sayısı", Kind: "number", Required: true, List: true},
				{ID: "reservation_date", Label: "Tarih", Kind: "date", Required: true, List: true},
				{ID: "reservation_time", Label: "Saat", Kind: "text", Required: true, List: true},
				{ID: "special_requests", Label: "Özel İstek / Not", Kind: "textarea"},
				{ID: "status", Label: "Durum", Kind: "select", Options: []string{"bekliyor", "onaylandi", "tamamlandi", "iptal"}, List: true},
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
		"tagline":         "Akdeniz Esintili Gurme Mutfak & Kokteyller",
		"primary_color":   "#B45309", // Warm Amber
		"bg_tone":         "#18181B", // Dark Zinc / Night Lounge
		"font_preset":     "artisan_garamond",
		"phone":           cleanPhone,
		"whatsapp_number": cleanPhone,
		"address":         "Bebek Mah. Cevdetpaşa Cad. No:44 Beşiktaş / İstanbul",
		"opening_hours":   "Her Gün 11:30 - 01:30 (Mutfak Kapanış 23:30)",
		"hero_badge":      "Mevsimsel Yerel Lezzetler & QR Dijital Menü",
		"hero_title":      "Tarihi Dokuda Çağdaş Akdeniz Gastronomisi",
		"hero_subtitle":   "Taş fırından taze çıkan artizan lezzetler, el yapımı makarnalar ve miksoloji uzmanlarından imza kokteyller.",
		"hero_image":      "https://images.unsplash.com/photo-1517248135467-4c7edcad34c4?auto=format&fit=crop&w=1200&q=80",
	}

	categories := []map[string]any{
		{"name": "Başlangıçlar & Antipasti", "slug": "starters", "sort_order": 1, "description": "Paylaşımlık sıcak ve soğuk tabaklar"},
		{"name": "El Yapımı Makarnalar & Risotto", "slug": "pasta", "sort_order": 2, "description": "Her sabah taze açılan artizan hamurlar"},
		{"name": "Odun Ateşinde Ana Yemekler", "slug": "mains", "sort_order": 3, "description": "Özel marine edilmiş et ve deniz ürünleri"},
		{"name": "İmza Kokteyller & İçecekler", "slug": "cocktails", "sort_order": 4, "description": "Özel aromalarla hazırlanan miksoloji kreasyonları"},
	}

	items := []map[string]any{
		{
			"name":            "Trüflü Yabani Mantarlı Arancini",
			"category_slug":   "starters",
			"price":           380.0,
			"description":     "Porçini mantarı, trüf ezmesi, fior di latte mozzarella ve safranlı aioli sos ile.",
			"image_url":       "https://images.unsplash.com/photo-1541529086526-db283c563270?auto=format&fit=crop&w=600&q=80",
			"is_chef_special": true,
			"is_vegetarian":   true,
			"is_gluten_free":  false,
			"is_available":    true,
		},
		{
			"name":            "Burrata & Fırınlanmış İncir",
			"category_slug":   "starters",
			"price":           460.0,
			"description":     "Puglia burratası, karamelize taze incir, ceviz kıtırı, nar ekşisi redüksiyonu ve fesleğen yağı.",
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
			"description":     "8 saat kısık ateşte ağır pişirilmiş dana kaburga ragu, 24 ay dinlendirilmiş parmesan ve adaçayı.",
			"image_url":       "https://images.unsplash.com/photo-1621996346565-e3d5d62810a9?auto=format&fit=crop&w=600&q=80",
			"is_chef_special": true,
			"is_vegetarian":   false,
			"is_gluten_free":  false,
			"is_available":    true,
		},
		{
			"name":            "Deniz Mahsullü Siyah Tagliolini",
			"category_slug":   "pasta",
			"price":           640.0,
			"description":     "Mürekkep balıklı taze makarna, jumbo karides, kalamar, sarımsaklı kiraz domates ve taze chili.",
			"image_url":       "https://images.unsplash.com/photo-1563379091339-03b21ab4a4f8?auto=format&fit=crop&w=600&q=80",
			"is_chef_special": false,
			"is_vegetarian":   false,
			"is_gluten_free":  false,
			"is_available":    true,
		},
		{
			"name":            "Izgara Kaya Levreği & Enginar Püresi",
			"category_slug":   "mains",
			"price":           720.0,
			"description":     "Çıtır derili taze kaya levreği, kadife enginar püresi, kapari ve limonlu tereyağ sos.",
			"image_url":       "https://images.unsplash.com/photo-1519708227418-c8fd9a32b7a2?auto=format&fit=crop&w=600&q=80",
			"is_chef_special": true,
			"is_vegetarian":   false,
			"is_gluten_free":  true,
			"is_available":    true,
		},
		{
			"name":            "Soleil Botanica (İmza Kokteyl)",
			"category_slug":   "cocktails",
			"price":           390.0,
			"description":     "Biberiye demlenmiş cin, mürver çiçeği likörü, taze misket limonu, salatalık ve tonik.",
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
