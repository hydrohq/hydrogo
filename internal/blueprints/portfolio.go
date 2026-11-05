package blueprints

import (
	"context"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// PortfolioDefinitions returns collections for a creative studio / architect / artist portfolio.
func PortfolioDefinitions() []contenttypes.Definition {
	return []contenttypes.Definition{
		{
			ID:          "site_settings",
			Label:       "Stüdyo & Sanatçı Ayarları",
			Description: "Portföy kimliği, biyografi, iletişim ve tema renkleri.",
			Group:       "Portfolyo Yönetimi",
			Icon:        "camera",
			TitleField:  "artist_name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "artist_name", Label: "Stüdyo / Sanatçı Adı", Kind: "text", Required: true, List: true},
				{ID: "tagline", Label: "Disiplin / Unvan", Kind: "text", List: true},
				{ID: "primary_color", Label: "Ana Vurgu Rengi", Kind: "text"},
				{ID: "bg_tone", Label: "Arka Plan Tonu", Kind: "text"},
				{ID: "font_preset", Label: "Tipografi Paleti", Kind: "select", Options: []string{"v0_geist", "editorial_playfair", "artisan_garamond", "monumental_cinzel", "avantgarde_syne", "modern_grotesk"}},
				{ID: "contact_email", Label: "İletişim E-Postası", Kind: "text", List: true},
				{ID: "phone", Label: "Telefon", Kind: "text", List: true},
				{ID: "location", Label: "Konum / Şehir", Kind: "text"},
				{ID: "instagram", Label: "Instagram", Kind: "text"},
				{ID: "bio", Label: "Biyografi / Sanatçı Manifestosu", Kind: "textarea"},
				{ID: "hero_title", Label: "Hero Başlık", Kind: "text"},
				{ID: "hero_subtitle", Label: "Hero Alt Metin", Kind: "textarea"},
			},
		},
		{
			ID:          "works",
			Label:       "Projeler & İşler",
			Description: "Fotoğraf serileri, mimari tasarımlar, görsel eserler.",
			Group:       "Portfolyo Yönetimi",
			Icon:        "grid",
			TitleField:  "title",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Proje Başlığı", Kind: "text", Required: true, List: true},
				{ID: "slug", Label: "URL Kodu (Slug)", Kind: "text", Required: true, Unique: true, List: true},
				{ID: "category", Label: "Kategori (Mimari, Fotoğraf, vb.)", Kind: "text", Required: true, List: true},
				{ID: "year", Label: "Yıl", Kind: "text", List: true},
				{ID: "client", Label: "Müşteri / Kurum", Kind: "text"},
				{ID: "cover_image", Label: "Kapak Görseli URL", Kind: "text", Required: true},
				{ID: "gallery_json", Label: "Galeri Görselleri (JSON)", Kind: "textarea"},
				{ID: "description", Label: "Proje Hikayesi", Kind: "textarea"},
				{ID: "featured", Label: "Ana Sayfada Öne Çıkar", Kind: "boolean", List: true},
			},
		},
		{
			ID:          "awards",
			Label:       "Ödüller & Yayınlar",
			Description: "Alınan ödüller, sergiler ve basında yer alan haberler.",
			Group:       "Portfolyo Yönetimi",
			Icon:        "award",
			TitleField:  "title",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Ödül / Sergi Adı", Kind: "text", Required: true, List: true},
				{ID: "organization", Label: "Kurum / Festival", Kind: "text", Required: true, List: true},
				{ID: "year", Label: "Yıl", Kind: "text", Required: true, List: true},
				{ID: "project_name", Label: "İlgili Eser", Kind: "text"},
			},
		},
		{
			ID:          "inquiries",
			Label:       "Proje & İş Birlikleri",
			Description: "Gelen sergi, sipariş ve proje teklifleri.",
			Group:       "Gelen Kutusu",
			Icon:        "inbox",
			TitleField:  "client_name",
			Public:      false,
			Fields: []contenttypes.Field{
				{ID: "client_name", Label: "İsim / Kurum", Kind: "text", Required: true, List: true},
				{ID: "email", Label: "E-Posta", Kind: "text", Required: true, List: true},
				{ID: "project_type", Label: "Proje Tipi", Kind: "text", List: true},
				{ID: "timeline", Label: "Zaman Planı", Kind: "text"},
				{ID: "message", Label: "Proje Detayı", Kind: "textarea", Required: true},
				{ID: "status", Label: "Durum", Kind: "select", Options: []string{"yeni", "cevaplandi", "anlasildi", "arsiv"}, List: true},
			},
		},
	}
}

// GeneratePortfolioCatalog generates default portfolio data.
func GeneratePortfolioCatalog(name, phone, prompt string) (map[string]any, []map[string]any, []map[string]any) {
	brandName := strings.TrimSpace(name)
	if brandName == "" {
		brandName = "Apex Mimarlık & Tasarım Stüdyosu"
	}

	settings := map[string]any{
		"artist_name":   brandName,
		"tagline":       "Minimalist Mimari, Mekan Deneyimi ve Sürdürülebilir Tasarım",
		"primary_color": "#0F172A", // Deep Charcoal / Slate
		"bg_tone":       "#F8FAFC", // Museum White
		"font_preset":   "monumental_cinzel",
		"contact_email": "studio@" + strings.ToLower(strings.ReplaceAll(brandName, " ", "")) + ".com",
		"phone":         phone,
		"location":      "İstanbul / Berlin",
		"instagram":     "@apex_studio",
		"bio":           "Işık, malzeme dürüstlüğü ve mekansal akışkanlığı merkeze alarak insan odaklı, zamansız mimari yapılar ve iç mekanlar tasarlıyoruz.",
		"hero_title":    "Mekanın Ruhunu Şekillendiren Çağdaş Mimari",
		"hero_subtitle": "Brütalizm ile doğallığı harmanlayan konut, ticari ve kültürel mekan tasarımları.",
	}

	works := []map[string]any{
		{
			"title":       "Bodrum Evi: Taş ve Işığın Dansı",
			"slug":        "bodrum-evi-tas-ve-isik",
			"category":    "Konut Mimarisi",
			"year":        "2025",
			"client":      "Özel Müşteri",
			"cover_image": "https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&w=800&q=80",
			"description": "Yerel kayrak taşları ve sürdürülebilir zeytin ağacı gölgelikleri ile Ege peyzajına gömülen sıfır enerjili villa projesi.",
			"featured":    true,
		},
		{
			"title":       "Galata Loft & Sanatçı Rezidansı",
			"slug":        "galata-loft-rezidans",
			"category":    "İç Mimari & Restorasyon",
			"year":        "2024",
			"client":      "Koleksiyoncu Evi",
			"cover_image": "https://images.unsplash.com/photo-1600607687939-ce8a6c25118c?auto=format&fit=crop&w=800&q=80",
			"description": "120 yıllık tarihi tuğla tonozların modern ham beton ve heykelsi çelik merdivenlerle yeniden yorumlanması.",
			"featured":    true,
		},
		{
			"title":       "Pavilion X: Çağdaş Heykel Pavyonu",
			"slug":        "pavilion-x-sanat",
			"category":    "Kültürel Alan",
			"year":        "2024",
			"client":      "İstanbul Kültür Sanat Vakfı",
			"cover_image": "https://images.unsplash.com/photo-1513694203232-719a280e022f?auto=format&fit=crop&w=800&q=80",
			"description": "Geçici enstalasyonlar için parametrik ahşap kirişlerle inşa edilmiş yarı açık sergi mekanı.",
			"featured":    true,
		},
		{
			"title":       "Nordic Monolith Villa",
			"slug":        "nordic-monolith-villa",
			"category":    "Konut Mimarisi",
			"year":        "2023",
			"client":      "Özel Tasarım",
			"cover_image": "https://images.unsplash.com/photo-1600566753376-12c8ab7fb75b?auto=format&fit=crop&w=800&q=80",
			"description": "Göl kıyısında monolitik siyah ahşap kabuk ve yerden tavana panoramik cam cephe tasarımı.",
			"featured":    false,
		},
	}

	awards := []map[string]any{
		{
			"title":        "Yılın En İyi Konut Tasarımı",
			"organization": "Ulusal Mimarlık Ödülleri",
			"year":         "2025",
			"project_name": "Bodrum Evi",
		},
		{
			"title":        "Tarihi Yapıda Yeniden İşlevlendirme Özel Ödülü",
			"organization": "European Heritage Design Awards",
			"year":         "2024",
			"project_name": "Galata Loft",
		},
		{
			"title":        "Sürdürülebilir Mimarlık Madalyası",
			"organization": "Green Building Council",
			"year":         "2023",
			"project_name": "Pavilion X",
		},
	}

	return settings, works, awards
}

// SeedPortfolioStore provisions schemas and initial data for portfolio archetype.
func SeedPortfolioStore(ctx context.Context, colSvc collections.Service, scope tenant.Scope, name, phone, prompt string) error {
	tenantCtx := tenant.WithScope(ctx, scope)
	settings, works, awards := GeneratePortfolioCatalog(name, phone, prompt)

	if _, err := colSvc.CreateRecord(tenantCtx, "site_settings", settings); err != nil {
		return fmt.Errorf("seed portfolio site_settings: %w", err)
	}
	for _, w := range works {
		if _, err := colSvc.CreateRecord(tenantCtx, "works", w); err != nil {
			return fmt.Errorf("seed work %q: %w", w["title"], err)
		}
	}
	for _, a := range awards {
		if _, err := colSvc.CreateRecord(tenantCtx, "awards", a); err != nil {
			return fmt.Errorf("seed award: %w", err)
		}
	}
	return nil
}
