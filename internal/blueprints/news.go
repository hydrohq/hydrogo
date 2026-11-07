package blueprints

import (
	"context"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// NewsDefinitions returns the collection schemas for a modern digital news portal.
func NewsDefinitions() []contenttypes.Definition {
	return []contenttypes.Definition{
		{
			ID:          "site_settings",
			Label:       "Portal & Yayın Ayarları",
			Description: "Gazete/portal adı, slogan, vurgu renkleri, künye ve duyurular.",
			Group:       "Yayın Yönetimi",
			Icon:        "newspaper",
			TitleField:  "brand_name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "brand_name", Label: "Portal Adı", Kind: "text", Required: true, List: true},
				{ID: "tagline", Label: "Yayın Mottosu / Slogan", Kind: "text", List: true},
				{ID: "primary_color", Label: "Ana Vurgu Rengi", Kind: "text"},
				{ID: "bg_tone", Label: "Arka Plan Tonu", Kind: "text"},
				{ID: "font_preset", Label: "Tipografi Paleti", Kind: "select", Options: []string{"v0_geist", "editorial_playfair", "artisan_garamond", "monumental_cinzel", "avantgarde_syne", "modern_grotesk"}},
				{ID: "breaking_news", Label: "Son Dakika Manşeti", Kind: "text", List: true},
				{ID: "contact_email", Label: "Yazı İşleri E-Postası", Kind: "text", List: true},
			},
		},
		{
			ID:          "news_categories",
			Label:       "Haber Kategorileri",
			Description: "Gündem, Teknoloji, Ekonomi, Kültür, Bilim vb.",
			Group:       "Yayın Yönetimi",
			Icon:        "folder",
			TitleField:  "name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "name", Label: "Kategori Adı", Kind: "text", Required: true, List: true},
				{ID: "slug", Label: "URL Kodu (Slug)", Kind: "text", Required: true, Unique: true, List: true},
				{ID: "sort_order", Label: "Sıralama", Kind: "number", List: true},
				{ID: "description", Label: "Kategori Açıklaması", Kind: "text"},
			},
		},
		{
			ID:          "news_articles",
			Label:       "Haberler ve Makaleler",
			Description: "Yayımlanan manşetler, analizler ve köşe yazıları.",
			Group:       "Yayın Yönetimi",
			Icon:        "file-text",
			TitleField:  "title",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Haber Başlığı", Kind: "text", Required: true, List: true},
				{ID: "slug", Label: "URL Kodu (Slug)", Kind: "text", Required: true, Unique: true, List: true},
				{ID: "category_slug", Label: "Kategori", Kind: "text", Required: true, List: true},
				{ID: "excerpt", Label: "Özet / Spot Metin", Kind: "textarea", Required: true, List: true},
				{ID: "body", Label: "Haber Detayı", Kind: "richtext", Required: true},
				{ID: "image_url", Label: "Kapak Görseli", Kind: "text"},
				{ID: "author_name", Label: "Yazar / Muhabir", Kind: "text", List: true},
				{ID: "reading_time", Label: "Okuma Süresi (Dk)", Kind: "number", List: true},
				{ID: "published_at", Label: "Yayın Tarihi", Kind: "text", List: true},
				{ID: "is_lead", Label: "Günün Ana Manşeti", Kind: "boolean", List: true},
				{ID: "is_trending", Label: "Çok Okunanlar", Kind: "boolean", List: true},
			},
		},
	}
}

// GenerateNewsCatalog builds authentic, rich editorial news content.
func GenerateNewsCatalog(brandName, phone, prompt string) (map[string]any, []map[string]any, []map[string]any) {
	if brandName == "" {
		brandName = "Nexus Haber"
	}

	settings := map[string]any{
		"brand_name":    brandName,
		"tagline":       "Bağımsız Dijital Gazete, Derinlemesine Analiz & Teknoloji",
		"primary_color": "#DC2626", // Editorial Crimson
		"bg_tone":       "#09090B",
		"font_preset":   "editorial_playfair",
		"contact_email": "editor@" + strings.ToLower(strings.ReplaceAll(brandName, " ", "")) + ".com",
		"breaking_news": "Yapay Zeka Regülasyonlarında Yeni Dönem: Küresel Açık Kaynak Standartları Kabul Edildi",
	}

	categories := []map[string]any{
		{"name": "Gündem", "slug": "gundem", "sort_order": 1, "description": "Türkiye ve dünyadan sıcak gelişmeler"},
		{"name": "Teknoloji", "slug": "teknoloji", "sort_order": 2, "description": "Yapay zeka, yazılım ve dijital dönüşüm"},
		{"name": "Ekonomi", "slug": "ekonomi", "sort_order": 3, "description": "Piyasalar, girişimcilik ve makroekonomi"},
		{"name": "Kültür & Sanat", "slug": "kultur", "sort_order": 4, "description": "Edebiyat, mimari, sinema ve düşünce"},
		{"name": "Bilim", "slug": "bilim", "sort_order": 5, "description": "Uzay araştırmaları, biyoteknoloji ve iklim"},
	}

	articles := []map[string]any{
		{
			"title":         "Otonom Yazılım Ajanları Çağı: Kod Yazımından Süreç Yönetimine Geçiş",
			"slug":          "otonom-yazilim-ajanlari-cagi",
			"category_slug": "teknoloji",
			"excerpt":       "Geliştirici ekipleri artık satır satır kod yazmaktan ziyade, hedefleri tanımlayan ve otonom döngüleri denetleyen sistem mimarlarına dönüşüyor.",
			"body":          "Yapay zeka modellerinin basit metin tamamlama seviyesinden çok adımlı planlama ve kendi kendine hata ayıklama yeteneğine evrilmesi, yazılım mühendisliğini kökten dönüştürüyor.\n\nYeni nesil ajanik çerçeveler, veritabanı şemalarından uçtan uca mikroservis entegrasyonlarına kadar tüm yaşam döngüsünü otonom yönetebiliyor. Sektör liderleri, 2027 yılına kadar kurumsal yazılımların %70'inin otonom ajan koordinasyonuyla inşa edileceğini öngörüyor.",
			"image_url":     "https://images.unsplash.com/photo-1526374965328-7f61d4dc18c5?auto=format&fit=crop&w=1200&q=80",
			"author_name":   "Deniz Aksoy",
			"reading_time":  4,
			"published_at":  "Bugün 14:30",
			"is_lead":       true,
			"is_trending":   true,
		},
		{
			"title":         "Küresel Merkez Bankaları ve Dijital Para Mimarileri: Yeni Finansal Düzen",
			"slug":          "kuresel-merkez-bankalari-dijital-para",
			"category_slug": "ekonomi",
			"excerpt":       "Sınır ötesi ödemelerde takas sürelerini saniyelere indiren yeni nesil CBDC ve dağıtık mutabakat protokolleri yürürlüğe giriyor.",
			"body":          "Geleneksel muhabir bankacılık ağlarının yüksek işlem maliyetleri ve çok günlük takas süreleri yerini kriptografik olarak denetlenebilir dağıtık mutabakat katmanlarına bırakıyor.\n\nAvrupa ve Asya merkez bankalarının ortak yürüttüğü pilot projeler, likidite yönetiminde sermaye verimliliğini %35 artırdığını raporladı.",
			"image_url":     "https://images.unsplash.com/photo-1611974789855-9c2a0a7236a3?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Murat Çelik",
			"reading_time":  5,
			"published_at":  "Bugün 11:15",
			"is_lead":       false,
			"is_trending":   true,
		},
		{
			"title":         "Mekansal Zarafet: Akdeniz Minimalizmi ile Çağdaş Kent Mimarisi",
			"slug":          "mekansal-zarafet-akdeniz-minimalizmi",
			"category_slug": "kultur",
			"excerpt":       "Doğal kireçtaşı, gölge oyunları ve brütalist sadelik; modern konut tasarımlarında yeni bir sükunet alanı yaratıyor.",
			"body":          "Şehrin gürültüsünden arınmış iç bahçeler, ham ahşap ve yerel taş işçiliği çağdaş tasarımın odağına yerleşti. Mimarlar, gösterişten uzak malzeme dürüstlüğünün uzun vadeli huzurun anahtarı olduğunu vurguluyor.",
			"image_url":     "https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Selin Vural",
			"reading_time":  3,
			"published_at":  "Dün 18:40",
			"is_lead":       false,
			"is_trending":   false,
		},
		{
			"title":         "James Webb Teleskobu Erken Evrenin En Yaşlı Galaksi Kümesini Doğruladı",
			"slug":          "james-webb-en-yasli-galaksi-kumesi",
			"category_slug": "bilim",
			"excerpt":       "Büyük Patlama'dan yalnızca 350 milyon yıl sonrasına tarihlenen yapılar, evrenin ilk yıldız oluşum modellerini yeniden yazdırıyor.",
			"body":          "Kızılötesi spektrometre verileri, ilk galaksilerin bugüne kadar varsayılandan çok daha hızlı kütle kazandığını ve beklenenden zengin metal izleri barındırdığını kanıtladı.",
			"image_url":     "https://images.unsplash.com/photo-1451187580459-43490279c0fa?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Prof. Dr. Erdem Kaya",
			"reading_time":  6,
			"published_at":  "Dün 12:00",
			"is_lead":       false,
			"is_trending":   true,
		},
		{
			"title":         "Yarıiletken Endüstrisinde 1 Nanometre Eşiği Aşılıyor: Yeni Transistör Geometrisi",
			"slug":          "yariiletken-endustrisinde-1nm-esigi",
			"category_slug": "teknoloji",
			"excerpt":       "Kuantum tünelleme engellerini aşan gate-all-around mimarisi ile enerji tüketimi %40 azalırken işlem gücü ikiye katlanıyor.",
			"body":          "Çip üreticileri, atomik düzeyde hassasiyet sağlayan aşırı ultraviyole (EUV) litografi teknikleriyle yeni nesil mobil ve veri merkezi işlemcilerini seri üretime hazırlıyor.",
			"image_url":     "https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Deniz Aksoy",
			"reading_time":  4,
			"published_at":  "2 gün önce",
			"is_lead":       false,
			"is_trending":   false,
		},
		{
			"title":         "Yeşil Enerjide Katı Hal Pil Devrimi: Elektrikli Araçlarda 1000 Km Menzil",
			"slug":          "yesil-enerjide-kati-hal-pil-devrimi",
			"category_slug": "bilim",
			"excerpt":       "Sıvı elektrolit riskini ortadan kaldıran seramik katı hal bataryalar, 10 dakikada %80 şarj kabiliyetiyle endüstriyi dönüştürüyor.",
			"body":          "Otomotiv devleri ve araştırma enstitüleri ortaklığında geliştirilen prototipler, dondurucu soğuklarda dahi menzil kaybı yaşamaksızın 2000 döngü ömür sunduğunu gösterdi.",
			"image_url":     "https://images.unsplash.com/photo-1466611653911-95081537e5b7?auto=format&fit=crop&w=800&q=80",
			"author_name":   "Murat Çelik",
			"reading_time":  4,
			"published_at":  "3 gün önce",
			"is_lead":       false,
			"is_trending":   true,
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
	return nil
}
