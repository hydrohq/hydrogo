package storefront

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderShowcase(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	data := ShowcaseData{
		BaseDomain: "ardalabs.dev",
	}

	if err := RenderShowcase(w, r, data); err != nil {
		t.Fatalf("RenderShowcase failed: %v", err)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Tek Platform") {
		t.Errorf("Showcase missing headline")
	}
	if !strings.Contains(body, "ardalabs.dev") {
		t.Errorf("Showcase missing base domain")
	}
}

func TestRenderCorporate(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	data := CorporateData{
		BrandName:    "Acme Kurumsal",
		Tagline:      "Strateji ve Büyüme",
		PrimaryColor: "#1E3A8A",
		Services: []CorporateService{
			{Title: "Dijital Strateji", ShortDescription: "Uçtan uca dönüşüm"},
		},
		Projects: []CorporateProject{
			{Title: "Banka Entegrasyonu", Category: "Fintech", Summary: "Hızlı ödeme"},
		},
	}

	if err := RenderCorporate(w, r, data); err != nil {
		t.Fatalf("RenderCorporate failed: %v", err)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Acme Kurumsal") {
		t.Errorf("Corporate missing brand name")
	}
}

func TestRenderRestaurant(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	data := RestaurantData{
		BrandName:    "Bistro Soleil",
		Tagline:      "Gurme Lezzetler",
		PrimaryColor: "#B45309",
		Categories: []RestaurantCategory{
			{Name: "Başlangıçlar", Slug: "starters"},
		},
		Items: []RestaurantItem{
			{Name: "Trüflü Arancini", CategorySlug: "starters", Price: 380},
		},
	}

	if err := RenderRestaurant(w, r, data); err != nil {
		t.Fatalf("RenderRestaurant failed: %v", err)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Bistro Soleil") {
		t.Errorf("Restaurant missing brand name")
	}
}

func TestRenderSaaS(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	data := SaaSData{
		BrandName:    "NovaFlow AI",
		Tagline:      "İş Akış Otomasyonu",
		PrimaryColor: "#2563EB",
		CTALabel:     "Hemen Başla",
		Features: []SaaSFeature{
			{Title: "Bento Feature", Badge: "Hız", Description: "Milisaniyelik"},
		},
		PricingTiers: []SaaSPricingTier{
			{Name: "Pro Plan", PriceMonthly: 99, PriceAnnual: 79, Features: []string{"Sınırsız İşlem"}},
		},
	}

	if err := RenderSaaS(w, r, data); err != nil {
		t.Fatalf("RenderSaaS failed: %v", err)
	}
	body := w.Body.String()
	if !strings.Contains(body, "NovaFlow AI") {
		t.Errorf("SaaS missing brand name")
	}
}

func TestRenderPortfolio(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	data := PortfolioData{
		ArtistName:   "Apex Stüdyo",
		Tagline:      "Minimalist Mimari",
		PrimaryColor: "#0F172A",
		Works: []PortfolioWork{
			{Title: "Bodrum Evi", Category: "Konut", Year: "2025"},
		},
	}

	if err := RenderPortfolio(w, r, data); err != nil {
		t.Fatalf("RenderPortfolio failed: %v", err)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Apex Stüdyo") {
		t.Errorf("Portfolio missing artist name")
	}
}
