package sitegen

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	aicore "github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/ai/prompts"
)

// Generate creates a complete SitePlan from brand information and user brief.
// If the AI assistant is configured and enabled, it calls the LLM via prompt registry.
// Otherwise, it returns a deterministic, high-quality fallback plan.
func Generate(ctx context.Context, brief, brandName, phone string, assistant aicore.Assistant) (SitePlan, error) {
	brandName = strings.TrimSpace(brandName)
	if brandName == "" {
		brandName = "Vera Atelier"
	}
	phone = strings.TrimSpace(phone)
	if phone == "" {
		phone = "+905550000000"
	}
	brief = strings.TrimSpace(brief)
	if brief == "" {
		brief = "Özel tasarım butik mağaza ve zanaat ürünleri."
	}

	// 1. If AI assistant is enabled and has an active provider, generate via LLM
	if assistant.Enabled && assistant.Provider != nil {
		renderedPrompt, pMeta, err := prompts.Render("sitegen_system", map[string]string{
			"BRAND_NAME":    brandName,
			"PHONE":         phone,
			"BRIEF":         brief,
			"EXTRA_CONTEXT": "",
		})
		if err == nil && renderedPrompt != "" {
			sug, err := assistant.Suggest(ctx, aicore.Request{
				Operation: aicore.Refine,
				Locale:    "tr",
				Tone:      "luxury boutique",
				Body:      renderedPrompt,
				Consent:   true,
			})
			if err == nil && strings.TrimSpace(sug.Body) != "" {
				if plan, parseErr := parseModelOutput(sug.Body); parseErr == nil {
					if pMeta != nil {
						plan.PromptID = pMeta.ID
						plan.PromptVersion = pMeta.Version
						plan.PromptHash = pMeta.Hash
					}
					if err := Validate(&plan); err == nil {
						return plan, nil
					}
				}
			}
		}
	}

	// 2. Deterministic fallback based on instruction defaults
	return GenerateDeterministicFallback(brief, brandName, phone), nil
}

func parseModelOutput(raw string) (SitePlan, error) {
	clean := strings.TrimSpace(raw)
	if idx := strings.Index(clean, "{"); idx != -1 {
		clean = clean[idx:]
	}
	if idx := strings.LastIndex(clean, "}"); idx != -1 {
		clean = clean[:idx+1]
	}

	var plan SitePlan
	if err := json.Unmarshal([]byte(clean), &plan); err != nil {
		return SitePlan{}, fmt.Errorf("decode site plan json: %w", err)
	}
	return plan, nil
}

// GenerateDeterministicFallback constructs a structured, complete SitePlan without calling external LLMs.
func GenerateDeterministicFallback(brief, brandName, phone string) SitePlan {
	lower := strings.ToLower(brief)

	theme := ThemeConfig{
		Preset:       "minimal_cream",
		FontPreset:   "v0_geist",
		BGColor:      "#FAF8F5",
		TextColor:    "#1C1917",
		AccentColor:  "#8C6D46",
		BorderRadius: "soft",
	}

	if strings.Contains(lower, "siyah") || strings.Contains(lower, "noir") || strings.Contains(lower, "koyu") {
		theme.Preset = "noir_gold"
		theme.BGColor = "#121212"
		theme.TextColor = "#F5F5F5"
		theme.AccentColor = "#D4AF37"
		theme.FontPreset = "monumental_cinzel"
	} else if strings.Contains(lower, "pastel") || strings.Contains(lower, "pembe") || strings.Contains(lower, "bebek") {
		theme.Preset = "pastel_blush"
		theme.BGColor = "#FFF8F6"
		theme.TextColor = "#2A2528"
		theme.AccentColor = "#C48B8B"
		theme.FontPreset = "editorial_playfair"
	} else if strings.Contains(lower, "yesil") || strings.Contains(lower, "silk") || strings.Contains(lower, "botanik") {
		theme.Preset = "emerald_silk"
		theme.BGColor = "#F4F7F5"
		theme.TextColor = "#15241C"
		theme.AccentColor = "#1E4D36"
		theme.FontPreset = "artisan_garamond"
	}

	layout := []LayoutSection{
		{
			Type: "hero",
			Props: map[string]any{
				"badge":     "ÖZEL KOLEKSİYON",
				"title":     brandName + " Seçkisi",
				"subtitle":  brief,
				"cta_text":  "Koleksiyonu İncele",
				"cta_link":  "/kategori/tumu",
				"image_url": "https://images.unsplash.com/photo-1565193566173-7a0ee3dbe261?auto=format&fit=crop&w=1200&q=80",
			},
		},
		{
			Type: "editorial",
			Props: map[string]any{
				"badge":     "ZANAAT VE ÖZEN",
				"title":     "Her Detayında Yaşayan Bir Ruh",
				"text":      brandName + " atölyesinde üretilen her parça, ustalık ve zamansız estetik anlayışıyla bir araya gelir.",
				"quote":     "Kusursuzluk sadelikte gizlidir.",
				"image_url": "https://images.unsplash.com/photo-1578749556568-bc2c40e68b61?auto=format&fit=crop&w=1200&q=80",
			},
		},
		{
			Type: "featured_grid",
			Props: map[string]any{
				"title":    "Öne Çıkan Seçkiler",
				"subtitle": "SINIRLI SAYIDA ÜRETİM",
			},
		},
		{
			Type: "pillars",
			Props: map[string]any{
				"title": "Atölye Sözümüz",
				"items": []any{
					map[string]any{"title": "Özgün Tasarım", "desc": "Her ürün tek tek el işçiliğiyle üretilir."},
					map[string]any{"title": "Kusursuz Paketleme", "desc": "Özel koruyucu ambalaj ve kırılma garantisi."},
					map[string]any{"title": "Hızlı WhatsApp Siparişi", "desc": "Tek tıkla atölye ile doğrudan iletişim."},
				},
			},
		},
		{
			Type: "contact_bar",
			Props: map[string]any{
				"address": "Kadıköy Moda Atölyesi",
				"hours":   "10:00 - 19:00",
				"phone":   phone,
			},
		},
	}

	products := []ProductItem{
		{
			Title:          "İmza Tasarım Koleksiyon Parçası",
			Slug:           "imza-tasarim-parcasi",
			Price:          1450,
			CompareAtPrice: 1750,
			Description:    "Zanaatkarlarımız tarafından özenle şekillendirilmiş, zamansız ve özel seri parça.",
			Badge:          "YENİ",
			ImageURL:       "https://images.unsplash.com/photo-1578749556568-bc2c40e68b61?auto=format&fit=crop&w=800&q=80",
		},
		{
			Title:          "El Yapımı Butik Seçki",
			Slug:           "el-yapimi-butik-secki",
			Price:          980,
			CompareAtPrice: 1200,
			Description:    "Doğal materyaller ve geleneksel yöntemlerle elde hazırlanan özel üretim ürün.",
			Badge:          "ÖZEL SERİ",
			ImageURL:       "https://images.unsplash.com/photo-1565193566173-7a0ee3dbe261?auto=format&fit=crop&w=800&q=80",
		},
		{
			Title:       "Minimalist Zanaat Objesi",
			Slug:        "minimalist-zanaat-objesi",
			Price:       720,
			Description: "Yaşam alanlarına zarafet ve dinginlik katan şık atölye ürünü.",
			Badge:       "ÇOK SATAN",
			ImageURL:    "https://images.unsplash.com/photo-1616486338812-3dadae4b4ace?auto=format&fit=crop&w=800&q=80",
		},
		{
			Title:          "Atölye Koleksiyon Aksesuarı",
			Slug:           "atolye-koleksiyon-aksesuari",
			Price:          1850,
			CompareAtPrice: 2200,
			Description:    "Modern yaşamla el emeğini buluşturan, yüksek kaliteli materyallerle üretilmiş parça.",
			Badge:          "SINIRLI SAYIDA",
			ImageURL:       "https://images.unsplash.com/photo-1513519245088-0e12902e5a38?auto=format&fit=crop&w=800&q=80",
		},
	}

	pMeta, _ := prompts.Load("sitegen_system")
	var pID, pVer, pHash string
	if pMeta != nil {
		pID = pMeta.ID
		pVer = pMeta.Version
		pHash = pMeta.Hash
	}

	return SitePlan{
		Brief:  brief,
		Theme:  theme,
		Layout: layout,
		StoreSettings: map[string]any{
			"store_name":       brandName,
			"slogan":           "Zanaat ve zarafetin buluştuğu nokta",
			"whatsapp_number":  phone,
			"announcement_bar": "Tüm siparişlerde ücretsiz kargo & özel kutu",
		},
		Products:      products,
		PromptID:      pID,
		PromptVersion: pVer,
		PromptHash:    pHash,
	}
}
