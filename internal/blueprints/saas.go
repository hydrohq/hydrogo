package blueprints

import (
	"context"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// SaaSDefinitions returns collections for a modern SaaS product landing page.
func SaaSDefinitions() []contenttypes.Definition {
	return []contenttypes.Definition{
		{
			ID:          "site_settings",
			Label:       "SaaS Ürün & Sayfa Ayarları",
			Description: "Ürün başlıkları, CTA butonları, ana renkler ve iletişim.",
			Group:       "SaaS Yönetimi",
			Icon:        "cpu",
			TitleField:  "brand_name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "brand_name", Label: "Ürün / SaaS Adı", Kind: "text", Required: true, List: true},
				{ID: "tagline", Label: "Ürün Sloganı", Kind: "text", List: true},
				{ID: "primary_color", Label: "Ana Vurgu Rengi", Kind: "text"},
				{ID: "bg_tone", Label: "Arka Plan Tonu", Kind: "text"},
				{ID: "font_preset", Label: "Tipografi Paleti", Kind: "select", Options: []string{"v0_geist", "editorial_playfair", "artisan_garamond", "monumental_cinzel", "avantgarde_syne", "modern_grotesk"}},
				{ID: "contact_email", Label: "İletişim E-Postası", Kind: "text", List: true},
				{ID: "hero_badge", Label: "Hero Rozet", Kind: "text"},
				{ID: "hero_title", Label: "Hero Başlık", Kind: "text"},
				{ID: "hero_subtitle", Label: "Hero Alt Metin", Kind: "textarea"},
				{ID: "cta_label", Label: "Ana CTA Buton Metni", Kind: "text"},
				{ID: "cta_link", Label: "Ana CTA Bağlantısı", Kind: "text"},
				{ID: "hero_image", Label: "Hero Dashboard Görseli", Kind: "text"},
			},
		},
		{
			ID:          "features",
			Label:       "Ürün Özellikleri (Bento Grid)",
			Description: "Platformun sunduğu temel özellikler ve yetenekler.",
			Group:       "SaaS Yönetimi",
			Icon:        "zap",
			TitleField:  "title",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Özellik Başlığı", Kind: "text", Required: true, List: true},
				{ID: "badge", Label: "Rozet / Kategori", Kind: "text", List: true},
				{ID: "description", Label: "Açıklama", Kind: "textarea", Required: true, List: true},
				{ID: "icon", Label: "İkon", Kind: "text"},
				{ID: "image_url", Label: "Görsel URL", Kind: "text"},
				{ID: "sort_order", Label: "Sıralama", Kind: "number", List: true},
			},
		},
		{
			ID:          "pricing_tiers",
			Label:       "Fiyatlandırma Paketleri",
			Description: "Aylık ve yıllık abonelik planları, özellik listeleri.",
			Group:       "SaaS Yönetimi",
			Icon:        "credit-card",
			TitleField:  "name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "name", Label: "Plan Adı", Kind: "text", Required: true, List: true},
				{ID: "price_monthly", Label: "Aylık Ücret ($ / ₺)", Kind: "number", Required: true, List: true},
				{ID: "price_annual", Label: "Yıllık Ücret ($ / ₺)", Kind: "number", Required: true, List: true},
				{ID: "description", Label: "Plan Açıklaması", Kind: "text"},
				{ID: "features_list", Label: "Dahil Olan Özellikler (Satır satır)", Kind: "textarea", Required: true},
				{ID: "is_highlighted", Label: "Öne Çıkarılan Plan ⭐", Kind: "boolean", List: true},
				{ID: "button_label", Label: "Buton Metni", Kind: "text"},
				{ID: "button_link", Label: "Buton Linki", Kind: "text"},
			},
		},
		{
			ID:          "faqs",
			Label:       "Sıkça Sorulan Sorular",
			Description: "Fiyatlandırma, teknik destek ve entegrasyon S.S.S.",
			Group:       "SaaS Yönetimi",
			Icon:        "help-circle",
			TitleField:  "question",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "question", Label: "Soru", Kind: "text", Required: true, List: true},
				{ID: "answer", Label: "Cevap", Kind: "textarea", Required: true},
				{ID: "category", Label: "Kategori", Kind: "text", List: true},
				{ID: "sort_order", Label: "Sıralama", Kind: "number", List: true},
			},
		},
		{
			ID:          "waitlist",
			Label:       "Erken Erişim & Bekleme Listesi",
			Description: "Lansman öncesi veya ücretsiz deneme için kayıt bırakanlar.",
			Group:       "Leads & Büyüme",
			Icon:        "user-plus",
			TitleField:  "email",
			Public:      false,
			Fields: []contenttypes.Field{
				{ID: "email", Label: "E-Posta Adresi", Kind: "text", Required: true, List: true},
				{ID: "full_name", Label: "Ad Soyad", Kind: "text", List: true},
				{ID: "company_name", Label: "Şirket Adı", Kind: "text", List: true},
				{ID: "use_case", Label: "Kullanım Amacı", Kind: "textarea"},
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
		"tagline":       "İş Akışlarınızı Yapay Zeka ile Otomatikleştiren Modern Platform",
		"primary_color": "#2563EB", // Electric Royal Blue
		"bg_tone":       "#09090B", // Pure Zinc Dark
		"font_preset":   "v0_geist",
		"contact_email": "hello@" + strings.ToLower(strings.ReplaceAll(brandName, " ", "")) + ".io",
		"hero_badge":    "NovaFlow v3.0 Şimdi Yayında ✨",
		"hero_title":    "Takımınızın Hızını 10 Katına Çıkaran Akıllı Otomasyon",
		"hero_subtitle": "Dağınık araçları tek bir yüksek performanslı kontrol merkezinde birleştirin. Sıfır kodlama ile dakikalar içinde canlıya alın.",
		"cta_label":     "14 Gün Ücretsiz Deneyin",
		"cta_link":      "#pricing",
		"hero_image":    "https://images.unsplash.com/photo-1551288049-bebda4e38f71?auto=format&fit=crop&w=1200&q=80",
	}

	features := []map[string]any{
		{
			"title":       "Gerçek Zamanlı Veri Senkronizasyonu",
			"badge":       "Performans",
			"description": "50+ popüler SaaS servisi ve veritabanıyla çift yönlü, milisaniyelik veri akışı sağlayın.",
			"icon":        "RefreshCw",
			"sort_order":  1,
		},
		{
			"title":       "Yapay Zeka Destekli Otomatik Analiz",
			"badge":       "Makine Öğrenmesi",
			"description": "Gelen verileri otomatik sınıflandırın, anormallikleri saptayın ve ekibinize proaktif uyarılar gönderin.",
			"icon":        "Cpu",
			"sort_order":  2,
		},
		{
			"title":       "Uçtan Uca Kurumsal Güvenlik",
			"badge":       "SOC2 & GDPR",
			"description": "Tüm verileriniz dinlenme ve iletim halinde AES-256 ile şifrelenir. Rol tabanlı erişim kontrolü (RBAC) dahildir.",
			"icon":        "Shield",
			"sort_order":  3,
		},
		{
			"title":       "Geliştirici Dostu Açık API & Webhook'lar",
			"badge":       "Geliştirici Deneyimi",
			"description": "TypeScript ve Go SDK'ları, OpenAPI belgeleri ve anlık webhook bildirimleriyle tam esneklik.",
			"icon":        "Code",
			"sort_order":  4,
		},
	}

	pricing := []map[string]any{
		{
			"name":           "Starter",
			"price_monthly":  29.0,
			"price_annual":   24.0,
			"description":    "Büyümekte olan küçük ekipler ve solo kurucular için ideal başlangıç.",
			"features_list":  "5 Kullanıcıya kadar\n100.000 İşlem / Ay\nStandart Entegrasyonlar\nTopluluk Desteği",
			"is_highlighted": false,
			"button_label":   "Hemen Başlayın",
			"button_link":    "#waitlist",
		},
		{
			"name":           "Pro Growth",
			"price_monthly":  79.0,
			"price_annual":   64.0,
			"description":    "Hızlı ölçeklenen startuplar ve orta ölçekli şirketler için tam güç.",
			"features_list":  "Sınırsız Kullanıcı\n2.500.000 İşlem / Ay\nAI Otomasyon Motoru\nÖncelikli 7/24 Destek\nÖzel Webhook & Entegrasyonlar",
			"is_highlighted": true,
			"button_label":   "14 Gün Ücretsiz Dene",
			"button_link":    "#waitlist",
		},
		{
			"name":           "Enterprise",
			"price_monthly":  249.0,
			"price_annual":   199.0,
			"description":    "Özel güvenlik, SLA garantisi ve on-premise kurulum arayan büyük kurumlar.",
			"features_list":  "Sınırsız Hacim & İşlem\nÖzel Dedicated Sunucu\n%99.99 SLA Garantisi\nÖzel Müşteri Başarı Yöneticisi\nSSO / SAML Entegrasyonu",
			"is_highlighted": false,
			"button_label":   "Satış Ekibiyle Görüşün",
			"button_link":    "#waitlist",
		},
	}

	faqs := []map[string]any{
		{
			"question":   "Ücretsiz deneme için kredi kartı gerekiyor mu?",
			"answer":     "Hayır! 14 günlük deneme sürümünü hiçbir kart bilgisi girmeden anında başlatabilirsiniz.",
			"category":   "Genel",
			"sort_order": 1,
		},
		{
			"question":   "Mevcut sistemlerimle entegrasyon ne kadar sürer?",
			"answer":     "Hazır bağlayıcılarımız sayesinde tipik bir entegrasyon 15 dakikadan az sürer. Gelişmiş API entegrasyonları için hazır SDK'larımızı sunuyoruz.",
			"category":   "Teknik",
			"sort_order": 2,
		},
		{
			"question":   "İstediğim zaman aboneliğimi iptal edebilir miyim?",
			"answer":     "Kesinlikle. Hiçbir taahhüt veya gizli ücret yoktur. Panelinizden tek tıkla iptal edebilirsiniz.",
			"category":   "Fiyatlandırma",
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
