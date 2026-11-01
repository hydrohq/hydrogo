package blueprints

import (
	"context"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// BoutiqueDefinitions returns the core collections needed for a luxury boutique store.
func BoutiqueDefinitions() []contenttypes.Definition {
	return []contenttypes.Definition{
		{
			ID:          "store_settings",
			Label:       "Mağaza ve Görünüm Ayarları",
			Description: "Butik mağaza kimliği, renk paleti, tipografi ve hızlı sipariş ayarları.",
			Group:       "Mağaza Yönetimi",
			Icon:        "palette",
			TitleField:  "brand_name",
			Public:      true,
			Revisions:   true,
			Fields: []contenttypes.Field{
				{ID: "brand_name", Label: "İşletme Adı", Kind: "text", Required: true, List: true},
				{ID: "tagline", Label: "Slogan / Alt Başlık", Kind: "text", List: true},
				{ID: "primary_color", Label: "Ana Vurgu Rengi (Hex)", Kind: "text"},
				{ID: "bg_tone", Label: "Arka Plan Tonu (Hex)", Kind: "text"},
				{ID: "whatsapp_number", Label: "WhatsApp Sipariş Hattı", Kind: "text", Required: true, List: true},
				{ID: "shopier_url", Label: "Shopier / Ödeme Linki", Kind: "text"},
				{ID: "payment_link", Label: "Ödeme Bağlantısı", Kind: "text"},
				{ID: "announcement_bar", Label: "Duyuru Bandı", Kind: "text"},
				{ID: "layout_mode", Label: "Vitrin Sayfa Mimarisi", Kind: "select", Options: []string{"hybrid", "single_page", "multipage"}},
				{ID: "font_preset", Label: "Tipografi & Yazı Tipi Paleti", Kind: "select", Options: []string{"v0_geist", "editorial_playfair", "artisan_garamond", "monumental_cinzel", "avantgarde_syne", "modern_grotesk"}},
				// Cross-compatibility fields with existing boutique theme engine
				{ID: "store_name", Label: "Mağaza Adı", Kind: "text"},
				{ID: "slogan", Label: "Slogan", Kind: "text"},
				{ID: "accent_color", Label: "Vurgu Rengi", Kind: "text"},
				{ID: "bg_primary_color", Label: "Arka Plan", Kind: "text"},
				{ID: "theme_preset", Label: "Tema Şablonu", Kind: "select", Options: []string{"minimal_cream", "noir_gold", "pastel_blush", "emerald_silk"}},
				{ID: "font_heading", Label: "Başlık Tipi", Kind: "text"},
				{ID: "border_radius_style", Label: "Köşe Stili", Kind: "text"},
				{ID: "hero_badge", Label: "Hero Rozet", Kind: "text"},
				{ID: "hero_title", Label: "Hero Başlık", Kind: "text"},
				{ID: "hero_subtitle", Label: "Hero Alt Başlık", Kind: "textarea"},
				{ID: "hero_image", Label: "Hero Görseli", Kind: "text"},
				{ID: "story_badge", Label: "Hikaye Rozet", Kind: "text"},
				{ID: "story_title", Label: "Hikaye Başlık", Kind: "text"},
				{ID: "story_text", Label: "Hikaye Metni", Kind: "textarea"},
				{ID: "story_quote", Label: "Hikaye Alıntısı", Kind: "text"},
				{ID: "story_image", Label: "Hikaye Görseli", Kind: "text"},
				{ID: "pillars_title", Label: "İlkeler Başlığı", Kind: "text"},
				{ID: "pillars_json", Label: "İlkeler (JSON)", Kind: "textarea"},
				{ID: "testimonials_json", Label: "Müşteri Yorumları (JSON)", Kind: "textarea"},
				{ID: "faqs_json", Label: "Sıkça Sorulan Sorular (JSON)", Kind: "textarea"},
				{ID: "atelier_address", Label: "Atölye Adresi", Kind: "text"},
				{ID: "atelier_hours", Label: "Çalışma Saatleri", Kind: "text"},
				{ID: "instagram", Label: "Instagram Hesabı", Kind: "text"},
			},
		},
		{
			ID:          "products",
			Label:       "Ürünler",
			Description: "Butik ürün kataloğu, fiyatlar, varyasyonlar ve stok durumu.",
			Group:       "Mağaza Yönetimi",
			Icon:        "shopping-bag",
			TitleField:  "title",
			Public:      true,
			Drafts:      true,
			Revisions:   true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Ürün Başlığı", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "slug", Label: "URL Kodu (Slug)", Kind: "text", Required: true, Unique: true, List: true, Filterable: true},
				{ID: "price", Label: "Fiyat (TL)", Kind: "number", Decimals: intPtr(2), Min: floatPtr(0), Required: true, List: true, Filterable: true},
				{ID: "compare_at_price", Label: "İndirimsiz Fiyat (TL)", Kind: "number", Decimals: intPtr(2), Min: floatPtr(0), List: true},
				{ID: "description", Label: "Açıklama", Kind: "richtext", Searchable: true},
				{ID: "image_url", Label: "Görsel URL", Kind: "text"},
				{ID: "cover_image", Label: "Kapak Görseli", Kind: "media"},
				{ID: "variants_json", Label: "Varyasyonlar (JSON)", Kind: "text"},
				{ID: "sizes", Label: "Bedenler (S, M, L)", Kind: "text", List: true},
				{ID: "colors", Label: "Renkler", Kind: "text", List: true},
				{ID: "in_stock", Label: "Stokta Var", Kind: "boolean", List: true, Filterable: true},
				{ID: "stock_quantity", Label: "Stok Adedi", Kind: "number", Min: floatPtr(0), List: true},
				{ID: "badge", Label: "Rozet", Kind: "text", List: true},
				{ID: "payment_link", Label: "Hızlı Satın Al (Shopier) Linki", Kind: "text"},
			},
		},
		{
			ID:          "orders",
			Label:       "Siparişler",
			Description: "WhatsApp ve kredi kartı ile oluşturulan müşteri siparişleri.",
			Group:       "Mağaza Yönetimi",
			Icon:        "package",
			TitleField:  "order_code",
			Public:      false,
			Revisions:   true,
			Fields: []contenttypes.Field{
				{ID: "order_code", Label: "Sipariş Kodu", Kind: "text", Sequence: &collections.Sequence{Prefix: "ORD-", Pad: 5}, List: true},
				{ID: "customer_name", Label: "Müşteri Ad Soyad", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "customer_phone", Label: "Telefon", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "customer_address", Label: "Teslimat Adresi", Kind: "textarea", Required: true},
				{ID: "items_json", Label: "Sipariş Kalemleri", Kind: "textarea", Required: true},
				{ID: "total_price", Label: "Toplam Tutar (TL)", Kind: "number", Decimals: intPtr(2), Min: floatPtr(0), Required: true, List: true},
				{ID: "status", Label: "Durum", Kind: "select", Options: []string{"yeni", "onaylandi", "kargolandi", "tamamlandi", "iptal"}, List: true, Filterable: true},
				// Cross-compatibility fields with boutique_orders
				{ID: "order_no", Label: "Sipariş No", Kind: "text"},
				{ID: "shipping_address", Label: "Adres", Kind: "textarea"},
				{ID: "items_summary", Label: "Kalemler", Kind: "textarea"},
				{ID: "total_amount", Label: "Tutar", Kind: "number"},
				{ID: "order_channel", Label: "Kanal", Kind: "text"},
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

	tagline := "Zamansız Zarafet & Seçkin Tasarımlar"
	announcement := "1500 TL Üzeri Siparişlerde Ücretsiz Kargo · WhatsApp ile Hızlı Sipariş"

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
		"shopier_url":      "https://shopier.com/" + strings.ToLower(strings.ReplaceAll(brandName, " ", "")),
		"payment_link":     "https://shopier.com/" + strings.ToLower(strings.ReplaceAll(brandName, " ", "")),
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
				"title":            "El Yapımı Hakiki Deri Kartlık & Cüzdan",
				"slug":             "el-yapimi-deri-kartlik",
				"price":            850.0,
				"compare_at_price": 1100.0,
				"description":      "Vaketa deri kullanılarak elde mumlu iplikle dikilmiş, minimalist ve uzun ömürlü kartlık.",
				"image_url":        "https://images.unsplash.com/photo-1627123424574-724758594e93?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Taba", "Kestane", "Siyah"]}`,
				"sizes":            "Standart",
				"colors":           "Taba, Kestane, Siyah",
				"in_stock":         true,
				"stock_quantity":   15,
				"badge":            "Yeni Sezon",
			},
			{
				"title":            "Vintage Postacı Çantası (Hakiki Deri)",
				"slug":             "vintage-postaci-cantasi",
				"price":            3450.0,
				"compare_at_price": 4200.0,
				"description":      "1. sınıf toskana derisi, pirinç tokalar ve ayarlanabilir omuz askısı ile zamansız bir klasik.",
				"image_url":        "https://images.unsplash.com/photo-1548036328-c9fa89d128fa?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Taba", "Antik Kahve"]}`,
				"sizes":            "Standart (38x28 cm)",
				"colors":           "Taba, Antik Kahve",
				"in_stock":         true,
				"stock_quantity":   8,
				"badge":            "Özel Tasarım",
			},
			{
				"title":            "Minimalist Deri Anahtarlık & AirTag Kılıfı",
				"slug":             "deri-anahtarlik-airtag",
				"price":            450.0,
				"compare_at_price": 600.0,
				"description":      "Paslanmaz çelik halka ve zarif deri gövde ile anahtarlarınızı şık bir aksesuara dönüştürün.",
				"image_url":        "https://images.unsplash.com/photo-1606760227091-3dd870d97f1d?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Taba", "Zümrüt Yeşili", "Lacivert"]}`,
				"sizes":            "Standart",
				"colors":           "Taba, Zümrüt Yeşili, Lacivert",
				"in_stock":         true,
				"stock_quantity":   25,
				"badge":            "Popüler",
			},
			{
				"title":            "İtalyan Deri Laptop Çantası (14-16 inç)",
				"slug":             "deri-laptop-cantasi",
				"price":            3950.0,
				"compare_at_price": 4800.0,
				"description":      "Darbe emici süet iç astar, fermuarlı evrak cebi ve el yapımı dikiş detayları.",
				"image_url":        "https://images.unsplash.com/photo-1553062407-98eeb64c6a62?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"ebat": ["14 inç", "16 inç"], "renkler": ["Taba", "Siyah"]}`,
				"sizes":            "14 inç, 16 inç",
				"colors":           "Taba, Siyah",
				"in_stock":         true,
				"stock_quantity":   6,
				"badge":            "Son 3 Ürün",
			},
			{
				"title":            "Klasik Deri Kemer (Pirinç Tokalı)",
				"slug":             "klasik-deri-kemer",
				"price":            950.0,
				"compare_at_price": 1250.0,
				"description":      "Masif pirinç toka ve 3.8 cm genişliğinde tam damarlı İtalyan sığır derisi.",
				"image_url":        "https://images.unsplash.com/photo-1624222247344-550fb60583dc?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"beden": ["90 cm", "100 cm", "110 cm"], "renk": ["Kahverengi", "Siyah"]}`,
				"sizes":            "90 cm, 100 cm, 110 cm",
				"colors":           "Kahverengi, Siyah",
				"in_stock":         true,
				"stock_quantity":   18,
				"badge":            "Klasik",
			},
			{
				"title":            "Deri Seyahat Not Defteri & Pasaport Kılıfı",
				"slug":             "deri-pasaport-kilifi",
				"price":            750.0,
				"compare_at_price": 950.0,
				"description":      "Değiştirilebilir defter içi, pasaport yuvası ve kalem tutucu ile seyahatlerinize eşlik eder.",
				"image_url":        "https://images.unsplash.com/photo-1544816155-12df9643f363?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Taba", "Bordo", "Koyu Yeşil"]}`,
				"sizes":            "Standart",
				"colors":           "Taba, Bordo, Koyu Yeşil",
				"in_stock":         true,
				"stock_quantity":   20,
				"badge":            "Hediye Önerisi",
			},
		}
	} else if isCeramic {
		products = []map[string]any{
			{
				"title":            "El Yapımı Stoneware Kupa (Krem & Terracotta)",
				"slug":             "stoneware-kupa",
				"price":            420.0,
				"compare_at_price": 550.0,
				"description":      "1250 derecede fırınlanmış dayanıklı stoneware çamur, gıdaya uygun mat sır.",
				"image_url":        "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Krem", "Terracotta"]}`,
				"sizes":            "250 ml",
				"colors":           "Krem, Terracotta",
				"in_stock":         true,
				"stock_quantity":   20,
				"badge":            "En Çok Satan",
			},
			{
				"title":            "Organik Formlu Seramik Vazo (30 cm)",
				"slug":             "organik-seramik-vazo",
				"price":            1450.0,
				"compare_at_price": 1850.0,
				"description":      "Heykelsi hatlara sahip, kurutulmuş veya taze çiçekler için su geçirmez dekoratif vazo.",
				"image_url":        "https://images.unsplash.com/photo-1612196808214-b8e1d6145a8c?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"ebat": ["30 cm"]}`,
				"sizes":            "30 cm",
				"colors":           "Ham Kum Beyazı",
				"in_stock":         true,
				"stock_quantity":   7,
				"badge":            "Özel Tasarım",
			},
			{
				"title":            "Mat Siyah Espresso Fincan Seti (2'li)",
				"slug":             "espresso-fincan-seti",
				"price":            580.0,
				"compare_at_price": 750.0,
				"description":      "Minimalist ahşap altlıkları ile birlikte gelen 2 adet el tornası espresso fincanı.",
				"image_url":        "https://images.unsplash.com/photo-1577937927133-66ef06acdf18?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Mat Siyah", "Benekli Krem"]}`,
				"sizes":            "90 ml (2 adet)",
				"colors":           "Mat Siyah, Benekli Krem",
				"in_stock":         true,
				"stock_quantity":   12,
				"badge":            "Yeni Sezon",
			},
			{
				"title":            "Doğal Kenarlı Seramik Sunum Tabağı",
				"slug":             "seramik-sunum-tabagi",
				"price":            750.0,
				"compare_at_price": 950.0,
				"description":      "Özel davet sofraları ve peynir sunumları için el yapımı rustik tabak.",
				"image_url":        "https://images.unsplash.com/photo-1578749556568-bc2c40e68b61?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"cap": ["26 cm", "32 cm"]}`,
				"sizes":            "26 cm, 32 cm",
				"colors":           "Toprak Dokulu",
				"in_stock":         true,
				"stock_quantity":   9,
				"badge":            "Özel Tasarım",
			},
			{
				"title":            "Soya Mumu Seramik Çanak (Lavanta & Sedir)",
				"slug":             "soya-mumu-seramik",
				"price":            490.0,
				"compare_at_price": 620.0,
				"description":      "yüzde 100 doğal soya mumu, ahşap fitil ve tütsülenmiş sedir kokusu.",
				"image_url":        "https://images.unsplash.com/photo-1603006905003-be475563bc59?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"koku": ["Lavanta & Sedir", "Vanilya & Amber"]}`,
				"sizes":            "300 gr",
				"colors":           "Toprak Çanak",
				"in_stock":         true,
				"stock_quantity":   25,
				"badge":            "Hediye Önerisi",
			},
			{
				"title":            "Ayaklı Meyvelik & Salata Kasesi",
				"slug":             "ayakli-meyvelik-seramik",
				"price":            1850.0,
				"compare_at_price": 2300.0,
				"description":      "Mutfak adanıza sofistike bir hava katan geniş hacimli el yapımı ayaklı kase.",
				"image_url":        "https://images.unsplash.com/photo-1590402494587-44b71d7772f6?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Benekli Bej", "Adaçayı Yeşili"]}`,
				"sizes":            "Genişlik 28 cm",
				"colors":           "Benekli Bej, Adaçayı Yeşili",
				"in_stock":         true,
				"stock_quantity":   5,
				"badge":            "Son 2 Ürün",
			},
		}
	} else {
		// Default luxury apparel & accessories
		products = []map[string]any{
			{
				"title":            "Saf İpek Kruvaze Gömlek",
				"slug":             "saf-ipek-kruvaze-gomlek",
				"price":            1850.0,
				"compare_at_price": 2300.0,
				"description":      "Doğal dökümlü saf ipek kumaş, sedef düğmeler ve zarif manşet detayları.",
				"image_url":        "https://images.unsplash.com/photo-1598554747436-c9293d6a588f?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"beden": ["S", "M", "L"], "renkler": ["Ekru", "Şampanya", "Siyah"]}`,
				"sizes":            "S, M, L",
				"colors":           "Ekru, Şampanya, Siyah",
				"in_stock":         true,
				"stock_quantity":   14,
				"badge":            "Yeni Sezon",
			},
			{
				"title":            "Yüksek Bel Doğal Keten Palazzo Pantolon",
				"slug":             "keten-palazzo-pantolon",
				"price":            1490.0,
				"compare_at_price": 1850.0,
				"description":      "Nefes alan doğal keten kumaşı ve rahat kalıbıyla yaz boyu konfor.",
				"image_url":        "https://images.unsplash.com/photo-1509631179647-0177331693ae?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"beden": ["S", "M", "L", "XL"], "renkler": ["Kum Beji", "Taş", "Lacivert"]}`,
				"sizes":            "S, M, L, XL",
				"colors":           "Kum Beji, Taş, Lacivert",
				"in_stock":         true,
				"stock_quantity":   20,
				"badge":            "Özel Tasarım",
			},
			{
				"title":            "Saten Dokulu Sırt Detaylı Midi Elbise",
				"slug":             "saten-midi-elbise",
				"price":            2450.0,
				"compare_at_price": 2950.0,
				"description":      "Özel davetler ve akşam etkinlikleri için tasarlanmış dökümlü saten elbise.",
				"image_url":        "https://images.unsplash.com/photo-1572804013309-59a88b7e92f1?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"beden": ["S", "M", "L"], "renkler": ["Zümrüt", "Siyah", "Bordo"]}`,
				"sizes":            "S, M, L",
				"colors":           "Zümrüt, Siyah, Bordo",
				"in_stock":         true,
				"stock_quantity":   6,
				"badge":            "Son 3 Ürün",
			},
			{
				"title":            "Oversize Kaşmir Karışımlı Triko Hırka",
				"slug":             "kasmir-triko-hirka",
				"price":            1950.0,
				"compare_at_price": 2400.0,
				"description":      "Yumuşak dokulu kaşmir karışımı, boynuz düğmeler ve rahat kesim.",
				"image_url":        "https://images.unsplash.com/photo-1434389677669-e08b4cac3105?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"beden": ["Standart (S-L)"], "renkler": ["Vizon", "Krem", "Antrasit"]}`,
				"sizes":            "Standart (S-L)",
				"colors":           "Vizon, Krem, Antrasit",
				"in_stock":         true,
				"stock_quantity":   11,
				"badge":            "En Çok Beğenilen",
			},
			{
				"title":            "Keten Kuşaklı Midi Kimono Ceket",
				"slug":             "keten-kusakli-kimono",
				"price":            1650.0,
				"compare_at_price": 1950.0,
				"description":      "Geniş kollar ve çıkarılabilir kuşak detayı ile hem plajda hem şehirde zarif bir parça.",
				"image_url":        "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"beden": ["Standart"], "renkler": ["Ekru", "Zeytin Yeşili", "Taba"]}`,
				"sizes":            "Standart",
				"colors":           "Ekru, Zeytin Yeşili, Taba",
				"in_stock":         true,
				"stock_quantity":   15,
				"badge":            "Yeni Sezon",
			},
			{
				"title":            "El Yapımı Deri & Kanvas Omuz Çantası",
				"slug":             "deri-kanvas-omuz-cantasi",
				"price":            2150.0,
				"compare_at_price": 2600.0,
				"description":      "Ağır gramajlı pamuk kanvas gövde ve hakiki deri askılarla zamansız bir şehir çantası.",
				"image_url":        "https://images.unsplash.com/photo-1544816155-12df9643f363?auto=format&fit=crop&w=800&q=80",
				"variants_json":    `{"renkler": ["Naturel/Taba", "Siyah/Siyah"]}`,
				"sizes":            "Geniş Hacimli",
				"colors":           "Naturel/Taba, Siyah/Siyah",
				"in_stock":         true,
				"stock_quantity":   8,
				"badge":            "Özel Tasarım",
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
