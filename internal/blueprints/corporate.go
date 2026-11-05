package blueprints

import (
	"context"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// CorporateDefinitions returns the collections needed for a modern corporate / B2B agency site.
func CorporateDefinitions() []contenttypes.Definition {
	return []contenttypes.Definition{
		{
			ID:          "site_settings",
			Label:       "Şirket & Site Ayarları",
			Description: "Kurumsal kimlik, iletişim bilgileri, istatistikler ve tema renkleri.",
			Group:       "Kurumsal Yönetim",
			Icon:        "building",
			TitleField:  "brand_name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "brand_name", Label: "Şirket / Ajans Adı", Kind: "text", Required: true, List: true},
				{ID: "tagline", Label: "Slogan / Alt Başlık", Kind: "text", List: true},
				{ID: "primary_color", Label: "Ana Vurgu Rengi", Kind: "text"},
				{ID: "bg_tone", Label: "Arka Plan Tonu", Kind: "text"},
				{ID: "font_preset", Label: "Tipografi Paleti", Kind: "select", Options: []string{"v0_geist", "editorial_playfair", "artisan_garamond", "monumental_cinzel", "avantgarde_syne", "modern_grotesk"}},
				{ID: "phone", Label: "Telefon", Kind: "text", List: true},
				{ID: "email", Label: "E-Posta", Kind: "text", List: true},
				{ID: "address", Label: "Ofis Adresi", Kind: "text"},
				{ID: "hero_badge", Label: "Hero Rozet", Kind: "text"},
				{ID: "hero_title", Label: "Hero Başlık", Kind: "text"},
				{ID: "hero_subtitle", Label: "Hero Alt Metin", Kind: "textarea"},
				{ID: "about_title", Label: "Hakkımızda Başlık", Kind: "text"},
				{ID: "about_text", Label: "Hakkımızda Metin", Kind: "textarea"},
				{ID: "stat_1_val", Label: "İstatistik 1 Değer", Kind: "text"},
				{ID: "stat_1_lbl", Label: "İstatistik 1 Etiket", Kind: "text"},
				{ID: "stat_2_val", Label: "İstatistik 2 Değer", Kind: "text"},
				{ID: "stat_2_lbl", Label: "İstatistik 2 Etiket", Kind: "text"},
				{ID: "stat_3_val", Label: "İstatistik 3 Değer", Kind: "text"},
				{ID: "stat_3_lbl", Label: "İstatistik 3 Etiket", Kind: "text"},
			},
		},
		{
			ID:          "services",
			Label:       "Hizmetlerimiz",
			Description: "Şirketin sunduğu uzmanlık alanları ve danışmanlık hizmetleri.",
			Group:       "Kurumsal Yönetim",
			Icon:        "briefcase",
			TitleField:  "title",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Hizmet Başlığı", Kind: "text", Required: true, List: true},
				{ID: "slug", Label: "URL Kodu (Slug)", Kind: "text", Required: true, Unique: true, List: true},
				{ID: "icon", Label: "İkon / Rozet", Kind: "text", List: true},
				{ID: "short_description", Label: "Kısa Açıklama", Kind: "textarea", Required: true, List: true},
				{ID: "detail_description", Label: "Detaylı Açıklama", Kind: "richtext"},
				{ID: "featured", Label: "Öne Çıkarılan", Kind: "boolean", List: true},
			},
		},
		{
			ID:          "projects",
			Label:       "Projeler & Vaka İncelemeleri",
			Description: "Tamamlanan projeler, müşteri başarı hikayeleri ve portföy.",
			Group:       "Kurumsal Yönetim",
			Icon:        "folder",
			TitleField:  "title",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Proje Adı", Kind: "text", Required: true, List: true},
				{ID: "slug", Label: "URL Kodu (Slug)", Kind: "text", Required: true, Unique: true, List: true},
				{ID: "client_name", Label: "Müşteri / Marka", Kind: "text", List: true},
				{ID: "category", Label: "Kategori", Kind: "text", List: true},
				{ID: "image_url", Label: "Görsel URL", Kind: "text"},
				{ID: "summary", Label: "Özet Açıklama", Kind: "textarea"},
				{ID: "metrics", Label: "Başarı Metriği", Kind: "text", List: true},
				{ID: "featured", Label: "Öne Çıkan", Kind: "boolean", List: true},
			},
		},
		{
			ID:          "testimonials",
			Label:       "Müşteri Görüşleri",
			Description: "Birlikte çalıştığımız kurumların ve liderlerin değerlendirmeleri.",
			Group:       "Kurumsal Yönetim",
			Icon:        "message-circle",
			TitleField:  "client_name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "client_name", Label: "Kişi Adı", Kind: "text", Required: true, List: true},
				{ID: "company", Label: "Şirket / Kurum", Kind: "text", List: true},
				{ID: "role", Label: "Pozisyon / Unvan", Kind: "text", List: true},
				{ID: "quote", Label: "Görüş / Yorum", Kind: "textarea", Required: true},
				{ID: "avatar_url", Label: "Fotoğraf URL", Kind: "text"},
			},
		},
		{
			ID:          "team",
			Label:       "Yönetim & Ekip",
			Description: "Kurumsal ekibimiz ve lider kadromuz.",
			Group:       "Kurumsal Yönetim",
			Icon:        "users",
			TitleField:  "name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "name", Label: "Ad Soyad", Kind: "text", Required: true, List: true},
				{ID: "role", Label: "Pozisyon / Uzmanlık", Kind: "text", Required: true, List: true},
				{ID: "bio", Label: "Özgeçmiş Özeti", Kind: "textarea"},
				{ID: "avatar_url", Label: "Profil Fotoğrafı URL", Kind: "text"},
				{ID: "linkedin", Label: "LinkedIn Profili", Kind: "text"},
			},
		},
		{
			ID:          "inquiries",
			Label:       "Gelen Talepler & Teklifler",
			Description: "Web sitesi iletişim ve teklif alma formundan düşen lead'ler.",
			Group:       "Gelen Kutusu",
			Icon:        "mail",
			TitleField:  "full_name",
			Public:      false,
			Fields: []contenttypes.Field{
				{ID: "full_name", Label: "Ad Soyad", Kind: "text", Required: true, List: true},
				{ID: "email", Label: "E-Posta", Kind: "text", Required: true, List: true},
				{ID: "phone", Label: "Telefon", Kind: "text", List: true},
				{ID: "company", Label: "Şirket Adı", Kind: "text", List: true},
				{ID: "service_needed", Label: "İlgilenilen Hizmet", Kind: "text", List: true},
				{ID: "message", Label: "Mesaj / Talep Özeti", Kind: "textarea", Required: true},
				{ID: "status", Label: "Durum", Kind: "select", Options: []string{"yeni", "gorusuluyor", "teklif_verildi", "tamamlandi", "arsiv"}, List: true},
			},
		},
	}
}

// GenerateCorporateCatalog generates default corporate site data.
func GenerateCorporateCatalog(name, phone, prompt string) (map[string]any, []map[string]any, []map[string]any, []map[string]any, []map[string]any) {
	brandName := strings.TrimSpace(name)
	if brandName == "" {
		brandName = "Acme Strateji & Danışmanlık"
	}
	cleanPhone := strings.TrimSpace(phone)
	if cleanPhone == "" {
		cleanPhone = "+90 212 555 0199"
	}

	settings := map[string]any{
		"brand_name":    brandName,
		"tagline":       "Geleceğe Hazır Dijital ve Stratejik Çözümler",
		"primary_color": "#1E3A8A", // Deep Navy
		"bg_tone":       "#F8FAFC", // Clean Slate
		"font_preset":   "v0_geist",
		"phone":         cleanPhone,
		"email":         "iletisim@" + strings.ToLower(strings.ReplaceAll(brandName, " ", "")) + ".com",
		"address":       "Levent 199, Büyükdere Cad. No:199 Şişli / İstanbul",
		"hero_badge":    "Kurumsal Dönüşüm & Mühendislik",
		"hero_title":    "Şirketinizi Yeni Nesil Stratejilerle Büyütün",
		"hero_subtitle": "Veri odaklı danışmanlık, dijital ürün mimarisi ve kurumsal stratejiyle ölçeklenebilir başarı sağlıyoruz.",
		"about_title":   "10 Yılı Aşkın Sektörel Liderlik ve Güven",
		"about_text":    "Global standartlarda mühendislik ve stratejik danışmanlık vizyonuyla, 100'ü aşkın yerli ve yabancı kuruma değer katıyoruz.",
		"stat_1_val":    "140+",
		"stat_1_lbl":    "Tamamlanan Global Proje",
		"stat_2_val":    "%99.4",
		"stat_2_lbl":    "Müşteri Sadakati",
		"stat_3_val":    "12 Yıl",
		"stat_3_lbl":    "Sektörel Deneyim",
	}

	services := []map[string]any{
		{
			"title":             "Stratejik Yönetim & Danışmanlık",
			"slug":              "stratejik-yonetim",
			"icon":              "Compass",
			"short_description": "İş modellerinizi modernize ederek pazar payınızı artıran uçtan uca strateji planlaması.",
			"detail_description": "Detaylı büyüme ve reorganizasyon stratejileri.",
			"featured":          true,
		},
		{
			"title":             "Kurumsal Dijital Dönüşüm",
			"slug":              "dijital-donusum",
			"icon":              "Layers",
			"short_description": "Eski sistemlerden modern mikroservis ve bulut mimarilerine kesintisiz geçiş.",
			"detail_description": "Yüksek performanslı ve ölçeklenebilir bulut çözümleri.",
			"featured":          true,
		},
		{
			"title":             "Veri Analitiği & İş Zekası",
			"slug":              "veri-analitigi",
			"icon":              "BarChart3",
			"short_description": "Büyük veriyi karar vericiler için aksiyon alınabilir canlı panellere dönüştürüyoruz.",
			"detail_description": "Tahmine dayalı analitik ve yapay zeka entegrasyonları.",
			"featured":          true,
		},
		{
			"title":             "Siber Güvenlik & Uyum",
			"slug":              "siber-guvenlik",
			"icon":              "ShieldCheck",
			"short_description": "KVKK, GDPR ve ISO 27001 gereksinimlerine tam uyumlu kurumsal güvenlik denetimi.",
			"detail_description": "Sıfır güven (zero-trust) prensibiyle altyapı koruması.",
			"featured":          false,
		},
	}

	projects := []map[string]any{
		{
			"title":       "Fintech Ödeme Altyapısı Modernizasyonu",
			"slug":        "fintech-odeme-modernizasyonu",
			"client_name": "Global Pay Solutions",
			"category":    "Finansal Teknolojiler",
			"image_url":   "https://images.unsplash.com/photo-1551836022-d5d88e9218df?auto=format&fit=crop&w=800&q=80",
			"summary":     "Günlük 2M+ işlemi sıfır kesintiyle karşılayan yeni nesil dağıtık ödeme motoru.",
			"metrics":     "%340 Hız Artışı",
			"featured":    true,
		},
		{
			"title":       "B2B Tedarik Zinciri Otomasyonu",
			"slug":        "tedarik-zinciri-otomasyonu",
			"client_name": "Nordic Lojistik A.Ş.",
			"category":    "Lojistik & Operasyon",
			"image_url":   "https://images.unsplash.com/photo-1586528116311-ad8dd3c8310d?auto=format&fit=crop&w=800&q=80",
			"summary":     "Depo yönetimini gerçek zamanlı IoT sensörleri ile otomatikleştirerek maliyetleri düşürdük.",
			"metrics":     "%42 Maliyet Tasarrufu",
			"featured":    true,
		},
		{
			"title":       "Sağlık Grubu Hasta Portalı & Veri Entegrasyonu",
			"slug":        "saglik-portali-entegrasyonu",
			"client_name": "Medicana Yaşam Grubu",
			"category":    "Sağlık Teknolojileri",
			"image_url":   "https://images.unsplash.com/photo-1576091160399-112ba8d25d1d?auto=format&fit=crop&w=800&q=80",
			"summary":     "300.000 aktif hastanın randevu ve tahlil süreçlerini tek mobil deneyimde birleştirdik.",
			"metrics":     "4.9/5 Mobil Skor",
			"featured":    true,
		},
	}

	testimonials := []map[string]any{
		{
			"client_name": "Kemal Aksoy",
			"company":     "Atlas Finans",
			"role":        "Teknoloji Direktörü (CTO)",
			"quote":       "Acme ekibi karmaşık bankacılık entegrasyonlarımızı planlanandan 2 ay önce ve sıfır hata ile teslim etti. Güven veren bir iş ortaklığı.",
			"avatar_url":  "https://images.unsplash.com/photo-1534528741775-53994a69daeb?auto=format&fit=crop&w=200&q=80",
		},
		{
			"client_name": "Selin Varol",
			"company":     "Vortex Holding",
			"role":        "Operasyon Başkanı (COO)",
			"quote":       "Dijitalleşme yolculuğumuzda kurumsal hafızamızı ve operasyonel hızımızı iki katına çıkaran stratejileri sundular.",
			"avatar_url":  "https://images.unsplash.com/photo-1580489944761-15a19d654956?auto=format&fit=crop&w=200&q=80",
		},
	}

	team := []map[string]any{
		{
			"name":       "Dr. Emre Can",
			"role":       "Kurucu Ortak & Baş Mimar",
			"bio":        "Stanford mezunu, 15+ yıllık kurumsal yazılım ve organizasyonel tasarım tecrübesi.",
			"avatar_url": "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?auto=format&fit=crop&w=300&q=80",
			"linkedin":   "https://linkedin.com",
		},
		{
			"name":       "Derya Yılmaz",
			"role":       "Strateji ve Büyüme Direktörü",
			"bio":        "McKinsey kökenli, çok uluslu şirketlerde 30+ M&A ve dönüşüm projesini yönetti.",
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
