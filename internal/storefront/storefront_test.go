package storefront

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderStorefront(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	data := StorefrontData{
		StoreName:       "Vera Seramik",
		Slogan:          "Zarif El Yapımı Seramikler",
		PrimaryColor:    "#1B5E43",
		BGTone:          "#FAF8F5",
		CleanPhone:      "05551234567",
		AnnouncementBar: "Tüm Siparişlerde Ücretsiz Kargo",
		IsDemo:          true,
		HeroBadge:       "2026 GÜZ KOLEKSİYONU",
		HeroTitle:       "Topraktan Sofraya",
		HeroSubtitle:    "Doğal killer ve kurşunsuz sırlarla atölyemizde tek tek el tornasında çekilen seramikler.",
		StoryBadge:      "ZANAAT VE SABIR",
		StoryTitle:      "Çamurun Dönüşüm Hikayesi",
		StoryText:       "Atölyemizde hiçbir nesne aceleye getirilmez.",
		StoryQuote:      "Kusursuz simetri yerine doğallık.",
		PillarsTitle:    "Atölye Sözümüz",
		Pillars: []FeaturePillar{
			{Title: "1240°C Stoneware", Desc: "Yüksek mukavemet."},
		},
		Testimonials: []Testimonial{
			{Customer: "Ceren A.", Location: "Kadıköy", Quote: "Harika fincanlar.", Item: "Espresso Seti"},
		},
		FAQs: []FAQItem{
			{Question: "Bulaşık makinesinde yıkanabilir mi?", Answer: "Evet, yıkanabilir."},
		},
		AtelierAddress: "Kadıköy Moda",
		AtelierHours:   "10:00 - 19:00",
		Instagram:      "@veraseramik",
		Products: []ProductView{
			{
				ID:       "prod-1",
				Title:    "Stoneware Kupa",
				Slug:     "stoneware-kupa",
				Price:    420,
				InStock:  true,
				ImageURL: "https://example.com/kupa.jpg",
			},
		},
	}

	err := Render(rec, req, data)
	if err != nil {
		t.Fatalf("unexpected Render error: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Vera Seramik") {
		t.Errorf("expected body to contain store name 'Vera Seramik'")
	}
	if !strings.Contains(body, "önizleme modundadır") {
		t.Errorf("expected demo banner in body")
	}
	if !strings.Contains(body, "Stoneware Kupa") {
		t.Errorf("expected product in body")
	}
	if !strings.Contains(body, "905551234567") {
		t.Errorf("expected clean phone in body")
	}
	if !strings.Contains(body, "Topraktan Sofraya") {
		t.Errorf("expected hero title in body")
	}
	if !strings.Contains(body, "1240°C Stoneware") {
		t.Errorf("expected pillar in body")
	}
	if !strings.Contains(body, "Çamurun Dönüşüm Hikayesi") {
		t.Errorf("expected story title in body")
	}
	if !strings.Contains(body, "Harika fincanlar.") {
		t.Errorf("expected testimonial in body")
	}
	if !strings.Contains(body, "Bulaşık makinesinde yıkanabilir mi?") {
		t.Errorf("expected FAQ question in body")
	}
}

func TestRenderSuspended(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	data := SuspendedData{
		StoreName:  "Vera Seramik",
		OwnerPhone: "05551234567",
	}

	err := RenderSuspended(rec, req, data)
	if err != nil {
		t.Fatalf("unexpected RenderSuspended error: %v", err)
	}

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "geçici olarak bakımdadır") {
		t.Errorf("expected maintenance message in body")
	}
}

func TestRenderProductDetail(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/urun/stoneware-kupa", nil)

	data := ProductDetailData{
		StoreName:    "Vera Seramik",
		PrimaryColor: "#1B5E43",
		CleanPhone:   "05551234567",
		Product: ProductView{
			ID:          "prod-1",
			Title:       "El Yapımı Stoneware Kupa",
			Slug:        "stoneware-kupa",
			Price:       450,
			Description: "1240 derecede fırınlanmış özel tasarım kupa.",
			InStock:     true,
			Badge:       "Çok Satan",
		},
		RelatedProducts: []ProductView{
			{ID: "prod-2", Title: "Espresso Fincanı", Slug: "espresso-fincani", Price: 320, InStock: true},
		},
	}

	err := RenderProductDetail(rec, req, data)
	if err != nil {
		t.Fatalf("unexpected RenderProductDetail error: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "El Yapımı Stoneware Kupa") {
		t.Errorf("expected product title in body")
	}
	if !strings.Contains(body, "Espresso Fincanı") {
		t.Errorf("expected related product title in body")
	}
	if !strings.Contains(body, "WhatsApp ile Doğrudan Sipariş Ver") {
		t.Errorf("expected WhatsApp order button in body")
	}
}

func TestRenderCategory(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/kategori/kupalar", nil)

	data := CategoryData{
		StoreName:       "Vera Seramik",
		CurrentCategory: "Kupalar",
		Categories:      []string{"Kupalar", "Tabaklar"},
		TotalCount:      1,
		Products: []ProductView{
			{ID: "prod-1", Title: "Stoneware Kupa", Slug: "stoneware-kupa", Price: 450, InStock: true},
		},
	}

	err := RenderCategory(rec, req, data)
	if err != nil {
		t.Fatalf("unexpected RenderCategory error: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Kupalar") {
		t.Errorf("expected category title in body")
	}
	if !strings.Contains(body, "Stoneware Kupa") {
		t.Errorf("expected product in body")
	}
}

func TestRenderAbout(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/hakkimizda", nil)

	data := AboutData{
		StoreName:  "Vera Seramik",
		StoryBadge: "ZANAAT VE SABIR",
		StoryTitle: "Topraktan Doğan Sanat",
		StoryText:  "Atölyemizde her parça elde şekillenir.",
		Pillars: []FeaturePillar{
			{Title: "Doğal Hammadde", Desc: "Kurşunsuz sırlar."},
		},
	}

	err := RenderAbout(rec, req, data)
	if err != nil {
		t.Fatalf("unexpected RenderAbout error: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Topraktan Doğan Sanat") {
		t.Errorf("expected story title in body")
	}
	if !strings.Contains(body, "Doğal Hammadde") {
		t.Errorf("expected pillar in body")
	}
}

func TestRenderContact(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/iletisim", nil)

	data := ContactData{
		StoreName:      "Vera Seramik",
		CleanPhone:     "05551234567",
		AtelierAddress: "Kadıköy Moda Cad. No: 12",
		AtelierHours:   "Hafta içi 10:00 - 19:00",
	}

	err := RenderContact(rec, req, data)
	if err != nil {
		t.Fatalf("unexpected RenderContact error: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Kadıköy Moda Cad. No: 12") {
		t.Errorf("expected address in body")
	}
	if !strings.Contains(body, "WhatsApp Destek") {
		t.Errorf("expected WhatsApp button in body")
	}
}
