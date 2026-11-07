package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/blueprints"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/storefront"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func (a *serveApp) getBaseDomain() string {
	baseDomain := strings.TrimSpace(a.cfg.Factory.BaseDomain)
	if baseDomain == "" {
		return "ardalabs.dev"
	}
	return baseDomain
}

func (a *serveApp) showcasePageHandler(w http.ResponseWriter, r *http.Request) {
	_ = storefront.RenderShowcase(w, r, storefront.ShowcaseData{
		BaseDomain: a.getBaseDomain(),
	})
}

func (a *serveApp) renderArchetypeStorefront(w http.ResponseWriter, r *http.Request, gov tenant.Governance, settings map[string]any, scope tenant.Scope, archetype string) bool {
	tenantCtx := tenant.WithScope(r.Context(), scope)

	switch archetype {
	case blueprints.ArchetypeCorporate:
		data := a.loadCorporateData(tenantCtx, gov, settings)
		if err := storefront.RenderCorporate(w, r, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return true

	case blueprints.ArchetypeRestaurant:
		data := a.loadRestaurantData(tenantCtx, gov, settings)
		if err := storefront.RenderRestaurant(w, r, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return true

	case blueprints.ArchetypeSaaS:
		data := a.loadSaaSData(tenantCtx, gov, settings)
		p := strings.Trim(r.URL.Path, "/")
		switch p {
		case "features", "ozellikler":
			data.ActivePage = "features"
		case "pricing", "fiyat":
			data.ActivePage = "pricing"
		case "about", "hakkimizda":
			data.ActivePage = "about"
		case "contact", "iletisim":
			data.ActivePage = "contact"
		default:
			data.ActivePage = "home"
		}
		if err := storefront.RenderSaaS(w, r, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return true

	case blueprints.ArchetypePortfolio:
		data := a.loadPortfolioData(tenantCtx, gov, settings)
		if err := storefront.RenderPortfolio(w, r, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return true

	case blueprints.ArchetypeNews:
		data := a.loadNewsData(tenantCtx, gov, settings, r)
		if err := storefront.RenderNews(w, r, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return true
	}

	return false
}

func (a *serveApp) loadCorporateData(ctx context.Context, gov tenant.Governance, settings map[string]any) storefront.CorporateData {
	data := storefront.CorporateData{
		BrandName:    gov.DisplayName,
		Tagline:      "Stratejik ve Dijital Dönüşüm",
		PrimaryColor: "#1E3A8A",
		BGTone:       "#F8FAFC",
		FontPreset:   "v0_geist",
		Phone:        gov.OwnerPhone,
		Email:        fmt.Sprintf("info@%s.%s", gov.TenantID, a.getBaseDomain()),
		Address:      "Levent 199, Büyükdere Cad. Şişli / İstanbul",
		HeroBadge:    "Kurumsal Dönüşüm",
		HeroTitle:    "Şirketinizi Yeni Nesil Stratejilerle Büyütün",
		HeroSubtitle: "Veri odaklı danışmanlık, dijital ürün mimarisi ve kurumsal stratejiyle ölçeklenebilir başarı sağlıyoruz.",
		Stat1Val:     "140+",
		Stat1Lbl:     "Tamamlanan Proje",
		Stat2Val:     "%99.4",
		Stat2Lbl:     "Müşteri Sadakati",
		Stat3Val:     "12 Yıl",
		Stat3Lbl:     "Sektörel Deneyim",
	}

	if s, ok := settings["brand_name"].(string); ok && s != "" {
		data.BrandName = s
	}
	if s, ok := settings["tagline"].(string); ok && s != "" {
		data.Tagline = s
	}
	if s, ok := settings["primary_color"].(string); ok && s != "" {
		data.PrimaryColor = s
	}
	if s, ok := settings["bg_tone"].(string); ok && s != "" {
		data.BGTone = s
	}
	if s, ok := settings["font_preset"].(string); ok && s != "" {
		data.FontPreset = s
	}
	if s, ok := settings["phone"].(string); ok && s != "" {
		data.Phone = s
	}
	if s, ok := settings["email"].(string); ok && s != "" {
		data.Email = s
	}
	if s, ok := settings["address"].(string); ok && s != "" {
		data.Address = s
	}
	if s, ok := settings["hero_badge"].(string); ok && s != "" {
		data.HeroBadge = s
	}
	if s, ok := settings["hero_title"].(string); ok && s != "" {
		data.HeroTitle = s
	}
	if s, ok := settings["hero_subtitle"].(string); ok && s != "" {
		data.HeroSubtitle = s
	}
	if s, ok := settings["stat_1_val"].(string); ok && s != "" {
		data.Stat1Val = s
	}
	if s, ok := settings["stat_1_lbl"].(string); ok && s != "" {
		data.Stat1Lbl = s
	}
	if s, ok := settings["stat_2_val"].(string); ok && s != "" {
		data.Stat2Val = s
	}
	if s, ok := settings["stat_2_lbl"].(string); ok && s != "" {
		data.Stat2Lbl = s
	}
	if s, ok := settings["stat_3_val"].(string); ok && s != "" {
		data.Stat3Val = s
	}
	if s, ok := settings["stat_3_lbl"].(string); ok && s != "" {
		data.Stat3Lbl = s
	}

	if a.collectionService.Repository != nil {
		if recs, err := a.collectionService.ListRecords(ctx, "services", 20, 0); err == nil {
			for _, r := range recs {
				data.Services = append(data.Services, storefront.CorporateService{
					Title:            strVal(r.Data["title"]),
					Slug:             strVal(r.Data["slug"]),
					Icon:             strVal(r.Data["icon"]),
					ShortDescription: strVal(r.Data["short_description"]),
					Featured:         boolVal(r.Data["featured"]),
				})
			}
		}
		if recs, err := a.collectionService.ListRecords(ctx, "projects", 20, 0); err == nil {
			for _, r := range recs {
				data.Projects = append(data.Projects, storefront.CorporateProject{
					Title:      strVal(r.Data["title"]),
					Slug:       strVal(r.Data["slug"]),
					ClientName: strVal(r.Data["client_name"]),
					Category:   strVal(r.Data["category"]),
					ImageURL:   strVal(r.Data["image_url"]),
					Summary:    strVal(r.Data["summary"]),
					Metrics:    strVal(r.Data["metrics"]),
					Featured:   boolVal(r.Data["featured"]),
				})
			}
		}
		if recs, err := a.collectionService.ListRecords(ctx, "testimonials", 10, 0); err == nil {
			for _, r := range recs {
				data.Testimonials = append(data.Testimonials, storefront.CorporateTestimonial{
					ClientName: strVal(r.Data["client_name"]),
					Company:    strVal(r.Data["company"]),
					Role:       strVal(r.Data["role"]),
					Quote:      strVal(r.Data["quote"]),
					AvatarURL:  strVal(r.Data["avatar_url"]),
				})
			}
		}
		if recs, err := a.collectionService.ListRecords(ctx, "team", 10, 0); err == nil {
			for _, r := range recs {
				data.Team = append(data.Team, storefront.CorporateTeamMember{
					Name:      strVal(r.Data["name"]),
					Role:      strVal(r.Data["role"]),
					Bio:       strVal(r.Data["bio"]),
					AvatarURL: strVal(r.Data["avatar_url"]),
					LinkedIn:  strVal(r.Data["linkedin"]),
				})
			}
		}
	}

	return data
}

func (a *serveApp) loadRestaurantData(ctx context.Context, gov tenant.Governance, settings map[string]any) storefront.RestaurantData {
	data := storefront.RestaurantData{
		BrandName:      gov.DisplayName,
		Tagline:        "Gurme Mutfak & Bar",
		PrimaryColor:   "#B45309",
		BGTone:         "#18181B",
		FontPreset:     "artisan_garamond",
		Phone:          gov.OwnerPhone,
		WhatsAppNumber: gov.OwnerPhone,
		Address:        "Bebek Mah. Cevdetpaşa Cad. Beşiktaş / İstanbul",
		OpeningHours:   "Her Gün 11:30 - 01:30",
		HeroBadge:      "QR Dijital Menü",
		HeroTitle:      "Çağdaş Gastronomi & İmza Kokteyller",
		HeroSubtitle:   "Taş fırından lezzetler ve taze açılan artizan makarnalar.",
		HeroImage:      "https://images.unsplash.com/photo-1517248135467-4c7edcad34c4?auto=format&fit=crop&w=1200&q=80",
	}

	if s, ok := settings["brand_name"].(string); ok && s != "" {
		data.BrandName = s
	}
	if s, ok := settings["tagline"].(string); ok && s != "" {
		data.Tagline = s
	}
	if s, ok := settings["primary_color"].(string); ok && s != "" {
		data.PrimaryColor = s
	}
	if s, ok := settings["bg_tone"].(string); ok && s != "" {
		data.BGTone = s
	}
	if s, ok := settings["font_preset"].(string); ok && s != "" {
		data.FontPreset = s
	}
	if s, ok := settings["phone"].(string); ok && s != "" {
		data.Phone = s
	}
	if s, ok := settings["whatsapp_number"].(string); ok && s != "" {
		data.WhatsAppNumber = s
	}
	if s, ok := settings["address"].(string); ok && s != "" {
		data.Address = s
	}
	if s, ok := settings["opening_hours"].(string); ok && s != "" {
		data.OpeningHours = s
	}
	if s, ok := settings["hero_badge"].(string); ok && s != "" {
		data.HeroBadge = s
	}
	if s, ok := settings["hero_title"].(string); ok && s != "" {
		data.HeroTitle = s
	}
	if s, ok := settings["hero_subtitle"].(string); ok && s != "" {
		data.HeroSubtitle = s
	}
	if s, ok := settings["hero_image"].(string); ok && s != "" {
		data.HeroImage = s
	}

	if a.collectionService.Repository != nil {
		if recs, err := a.collectionService.ListRecords(ctx, "menu_categories", 20, 0); err == nil {
			for _, r := range recs {
				data.Categories = append(data.Categories, storefront.RestaurantCategory{
					Name:        strVal(r.Data["name"]),
					Slug:        strVal(r.Data["slug"]),
					SortOrder:   intVal(r.Data["sort_order"]),
					Description: strVal(r.Data["description"]),
				})
			}
		}
		if recs, err := a.collectionService.ListRecords(ctx, "menu_items", 50, 0); err == nil {
			for _, r := range recs {
				data.Items = append(data.Items, storefront.RestaurantItem{
					Name:          strVal(r.Data["name"]),
					CategorySlug:  strVal(r.Data["category_slug"]),
					Price:         floatVal(r.Data["price"]),
					Description:   strVal(r.Data["description"]),
					ImageURL:      strVal(r.Data["image_url"]),
					IsChefSpecial: boolVal(r.Data["is_chef_special"]),
					IsVegetarian:  boolVal(r.Data["is_vegetarian"]),
					IsGlutenFree:  boolVal(r.Data["is_gluten_free"]),
					IsAvailable:   boolVal(r.Data["is_available"]),
				})
			}
		}
	}

	return data
}

func (a *serveApp) loadSaaSData(ctx context.Context, gov tenant.Governance, settings map[string]any) storefront.SaaSData {
	data := storefront.SaaSData{
		BrandName:    gov.DisplayName,
		Tagline:      "Akıllı İş Akış Otomasyonu",
		PrimaryColor: "#2563EB",
		BGTone:       "#09090B",
		FontPreset:   "v0_geist",
		ContactEmail: fmt.Sprintf("hello@%s.%s", gov.TenantID, a.getBaseDomain()),
		HeroBadge:    "v3.0 Dağıtık İş Akış Mimarisi",
		HeroTitle:    "İş Akışlarınızı Otonom Ajanlarla Yönetin",
		HeroSubtitle: "Tek bir yüksek performanslı motor üzerinde dağıtık süreçlerinizi yönetin. Sub-3ms P99 gecikme, SHA-256 denetim izi ve sıfır derleme dağıtım hızı.",
		CTALabel:     "14 Gün Ücretsiz Deneyin",
		CTALink:      "#waitlist",
		HeroImage:    "https://images.unsplash.com/photo-1551288049-bebda4e38f71?auto=format&fit=crop&w=1200&q=80",
	}

	if s, ok := settings["brand_name"].(string); ok && s != "" {
		data.BrandName = s
	}
	if s, ok := settings["tagline"].(string); ok && s != "" {
		data.Tagline = s
	}
	if s, ok := settings["primary_color"].(string); ok && s != "" {
		data.PrimaryColor = s
	}
	if s, ok := settings["bg_tone"].(string); ok && s != "" {
		data.BGTone = s
	}
	if s, ok := settings["font_preset"].(string); ok && s != "" {
		data.FontPreset = s
	}
	if s, ok := settings["contact_email"].(string); ok && s != "" {
		data.ContactEmail = s
	}
	if s, ok := settings["hero_badge"].(string); ok && s != "" {
		data.HeroBadge = s
	}
	if s, ok := settings["hero_title"].(string); ok && s != "" {
		data.HeroTitle = s
	}
	if s, ok := settings["hero_subtitle"].(string); ok && s != "" {
		data.HeroSubtitle = s
	}
	if s, ok := settings["cta_label"].(string); ok && s != "" {
		data.CTALabel = s
	}
	if s, ok := settings["hero_image"].(string); ok && s != "" {
		data.HeroImage = s
	}

	if a.collectionService.Repository != nil {
		if recs, err := a.collectionService.ListRecords(ctx, "features", 20, 0); err == nil {
			for _, r := range recs {
				data.Features = append(data.Features, storefront.SaaSFeature{
					Title:       strVal(r.Data["title"]),
					Badge:       strVal(r.Data["badge"]),
					Description: strVal(r.Data["description"]),
					Icon:        strVal(r.Data["icon"]),
					SortOrder:   intVal(r.Data["sort_order"]),
				})
			}
		}
		if recs, err := a.collectionService.ListRecords(ctx, "pricing_tiers", 10, 0); err == nil {
			for _, r := range recs {
				fl := strVal(r.Data["features_list"])
				lines := strings.Split(fl, "\n")
				var cleanLines []string
				for _, line := range lines {
					if t := strings.TrimSpace(line); t != "" {
						cleanLines = append(cleanLines, t)
					}
				}
				data.PricingTiers = append(data.PricingTiers, storefront.SaaSPricingTier{
					Name:          strVal(r.Data["name"]),
					PriceMonthly:  floatVal(r.Data["price_monthly"]),
					PriceAnnual:   floatVal(r.Data["price_annual"]),
					Description:   strVal(r.Data["description"]),
					FeaturesList:  fl,
					Features:      cleanLines,
					IsHighlighted: boolVal(r.Data["is_highlighted"]),
					ButtonLabel:   strVal(r.Data["button_label"]),
					ButtonLink:    strVal(r.Data["button_link"]),
				})
			}
		}
		if recs, err := a.collectionService.ListRecords(ctx, "faqs", 20, 0); err == nil {
			for _, r := range recs {
				data.FAQs = append(data.FAQs, storefront.SaaSFAQ{
					Question:  strVal(r.Data["question"]),
					Answer:    strVal(r.Data["answer"]),
					Category:  strVal(r.Data["category"]),
					SortOrder: intVal(r.Data["sort_order"]),
				})
			}
		}
	}

	return data
}

func (a *serveApp) loadPortfolioData(ctx context.Context, gov tenant.Governance, settings map[string]any) storefront.PortfolioData {
	data := storefront.PortfolioData{
		ArtistName:   gov.DisplayName,
		Tagline:      "Minimalist Mimari & Tasarım",
		PrimaryColor: "#0F172A",
		BGTone:       "#F8FAFC",
		FontPreset:   "monumental_cinzel",
		ContactEmail: fmt.Sprintf("studio@%s.%s", gov.TenantID, a.getBaseDomain()),
		Phone:        gov.OwnerPhone,
		Location:     "İstanbul / Berlin",
		Instagram:    "@studio",
		Bio:          "Işık, malzeme dürüstlüğü ve mekansal akışkanlığı merkeze alarak zamansız yapılar inşa ediyoruz.",
		HeroTitle:    "Mekanın Ruhunu Şekillendiren Çağdaş Tasarım",
		HeroSubtitle: "Brütalizm ile doğallığı harmanlayan konut, ticari ve kültürel alan tasarımları.",
	}

	if s, ok := settings["artist_name"].(string); ok && s != "" {
		data.ArtistName = s
	} else if s, ok := settings["brand_name"].(string); ok && s != "" {
		data.ArtistName = s
	}
	if s, ok := settings["tagline"].(string); ok && s != "" {
		data.Tagline = s
	}
	if s, ok := settings["primary_color"].(string); ok && s != "" {
		data.PrimaryColor = s
	}
	if s, ok := settings["bg_tone"].(string); ok && s != "" {
		data.BGTone = s
	}
	if s, ok := settings["font_preset"].(string); ok && s != "" {
		data.FontPreset = s
	}
	if s, ok := settings["contact_email"].(string); ok && s != "" {
		data.ContactEmail = s
	}
	if s, ok := settings["phone"].(string); ok && s != "" {
		data.Phone = s
	}
	if s, ok := settings["location"].(string); ok && s != "" {
		data.Location = s
	}
	if s, ok := settings["instagram"].(string); ok && s != "" {
		data.Instagram = s
	}
	if s, ok := settings["bio"].(string); ok && s != "" {
		data.Bio = s
	}
	if s, ok := settings["hero_title"].(string); ok && s != "" {
		data.HeroTitle = s
	}
	if s, ok := settings["hero_subtitle"].(string); ok && s != "" {
		data.HeroSubtitle = s
	}

	if a.collectionService.Repository != nil {
		if recs, err := a.collectionService.ListRecords(ctx, "works", 30, 0); err == nil {
			for _, r := range recs {
				data.Works = append(data.Works, storefront.PortfolioWork{
					Title:       strVal(r.Data["title"]),
					Slug:        strVal(r.Data["slug"]),
					Category:    strVal(r.Data["category"]),
					Year:        strVal(r.Data["year"]),
					Client:      strVal(r.Data["client"]),
					CoverImage:  strVal(r.Data["cover_image"]),
					Description: strVal(r.Data["description"]),
					Featured:    boolVal(r.Data["featured"]),
				})
			}
		}
		if recs, err := a.collectionService.ListRecords(ctx, "awards", 20, 0); err == nil {
			for _, r := range recs {
				data.Awards = append(data.Awards, storefront.PortfolioAward{
					Title:        strVal(r.Data["title"]),
					Organization: strVal(r.Data["organization"]),
					Year:         strVal(r.Data["year"]),
					ProjectName:  strVal(r.Data["project_name"]),
				})
			}
		}
	}

	return data
}

func (a *serveApp) storefrontInquireHandler(w http.ResponseWriter, r *http.Request) {
	scope, _, _, _, ok := a.resolveStorefrontContext(w, r)
	if !ok {
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "geçersiz istek", nil)
		return
	}
	req["status"] = contenttypes.StatusPublished
	tenantCtx := tenant.WithScope(r.Context(), scope)
	if a.collectionService.Repository != nil {
		_, _ = a.collectionService.CreateRecord(tenantCtx, "inquiries", req)
	}
	api.WriteData(w, http.StatusOK, map[string]any{"success": true, "message": "Talebiniz alındı."}, nil)
}

func (a *serveApp) storefrontReserveHandler(w http.ResponseWriter, r *http.Request) {
	scope, _, _, _, ok := a.resolveStorefrontContext(w, r)
	if !ok {
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "geçersiz istek", nil)
		return
	}
	req["status"] = "bekliyor"
	tenantCtx := tenant.WithScope(r.Context(), scope)
	if a.collectionService.Repository != nil {
		_, _ = a.collectionService.CreateRecord(tenantCtx, "reservations", req)
	}
	api.WriteData(w, http.StatusOK, map[string]any{"success": true, "message": "Rezervasyonunuz alındı."}, nil)
}

func (a *serveApp) storefrontWaitlistHandler(w http.ResponseWriter, r *http.Request) {
	scope, _, _, _, ok := a.resolveStorefrontContext(w, r)
	if !ok {
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "geçersiz istek", nil)
		return
	}
	tenantCtx := tenant.WithScope(r.Context(), scope)
	if a.collectionService.Repository != nil {
		_, _ = a.collectionService.CreateRecord(tenantCtx, "waitlist", req)
	}
	api.WriteData(w, http.StatusOK, map[string]any{"success": true, "message": "Erken erişim kaydınız alındı."}, nil)
}

func strVal(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func boolVal(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func intVal(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	if i, ok := v.(int); ok {
		return i
	}
	return 0
}

func floatVal(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	if i, ok := v.(int); ok {
		return float64(i)
	}
	return 0
}
