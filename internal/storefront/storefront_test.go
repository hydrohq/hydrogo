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
		LayoutMode:      "single_page",
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
	if !strings.Contains(body, `const storeWhatsapp = "905551234567";`) {
		t.Errorf("expected quoted storeWhatsapp in script")
	}
	if !strings.Contains(body, "secondary-img") {
		t.Errorf("expected secondary-img hover support")
	}
	if !strings.Contains(body, "/api/v1/store/orders") {
		t.Errorf("expected /api/v1/store/orders endpoint in storefront order script")
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
			Options: []OptionDef{
				{Name: "Renk", Values: []string{"Taba", "Siyah"}},
				{Name: "Beden", Values: []string{"Standart", "Büyük"}},
			},
			Variants: []VariantDef{
				{SKU: "TB-STD", Title: "Taba / Standart", PriceDiff: 0, Stock: 5},
				{SKU: "SY-BYK", Title: "Siyah / Büyük", PriceDiff: 50, Stock: 0},
			},
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
	if !strings.Contains(body, "Taba") || !strings.Contains(body, "Standart") {
		t.Errorf("expected option values in product detail body")
	}
	if !strings.Contains(body, "/api/v1/store/orders") {
		t.Errorf("expected /api/v1/store/orders in detail page script")
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

func TestResolveFontFamilies(t *testing.T) {
	hg, bg := ResolveFontFamilies("v0_geist")
	if !strings.Contains(string(hg), "Geist") || !strings.Contains(string(bg), "Geist") {
		t.Errorf("expected Geist font for v0_geist, got %s / %s", hg, bg)
	}

	h1, b1 := ResolveFontFamilies("editorial_playfair")
	if !strings.Contains(string(h1), "Playfair Display") || !strings.Contains(string(b1), "Plus Jakarta Sans") {
		t.Errorf("expected Playfair Display & Plus Jakarta Sans, got %s / %s", h1, b1)
	}

	h2, b2 := ResolveFontFamilies("monumental_cinzel")
	if !strings.Contains(string(h2), "Cinzel") || !strings.Contains(string(b2), "Inter") {
		t.Errorf("expected Cinzel & Inter, got %s / %s", h2, b2)
	}

	h3, b3 := ResolveFontFamilies("avantgarde_syne")
	if !strings.Contains(string(h3), "Syne") || !strings.Contains(string(b3), "Inter") {
		t.Errorf("expected Syne & Inter, got %s / %s", h3, b3)
	}
}

func TestServeFont(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/storefront/fonts/playfair-display.woff2", nil)
	req.SetPathValue("name", "playfair-display.woff2")

	ServeFont(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "font/woff2" {
		t.Errorf("expected font/woff2, got %s", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("expected immutable cache-control")
	}
	if rec.Body.Len() == 0 {
		t.Errorf("expected font body bytes")
	}

	recG := httptest.NewRecorder()
	reqG := httptest.NewRequest(http.MethodGet, "/storefront/fonts/geist-regular.woff2", nil)
	reqG.SetPathValue("name", "geist-regular.woff2")
	ServeFont(recG, reqG)
	if recG.Code != http.StatusOK {
		t.Fatalf("expected 200 for geist font, got %d", recG.Code)
	}
	if recG.Header().Get("Content-Type") != "font/woff2" {
		t.Errorf("expected font/woff2, got %s", recG.Header().Get("Content-Type"))
	}
}

func TestRenderStorefront_CustomLayout(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	layoutJSON := `[
		{
			"type": "hero",
			"props": {
				"title": "Özel Tasarım Vitrin",
				"subtitle": "Benzersiz el yapımı seramikler",
				"badge": "YENİ KOLEKSİYON"
			}
		},
		{
			"type": "newsletter",
			"props": {
				"title": "Bültenimize Abone Olun",
				"subtitle": "İlk siz haberdar olun"
			}
		}
	]`

	data := StorefrontData{
		StoreName:  "Vera Atelier",
		LayoutJSON: layoutJSON,
	}

	if err := Render(rec, req, data); err != nil {
		t.Fatalf("Render with custom layout failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Özel Tasarım Vitrin") {
		t.Errorf("expected custom hero title in body")
	}
	if !strings.Contains(body, "YENİ KOLEKSİYON") {
		t.Errorf("expected custom hero badge in body")
	}
	if !strings.Contains(body, "Bültenimize Abone Olun") {
		t.Errorf("expected custom newsletter section in body")
	}

	// Verify ordering: hero must appear before newsletter
	heroIdx := strings.Index(body, "Özel Tasarım Vitrin")
	newsIdx := strings.Index(body, "Bültenimize Abone Olun")
	if heroIdx >= newsIdx {
		t.Errorf("expected hero section to render before newsletter section (fidelity)")
	}
}

func TestRenderStorefront_UnknownSectionFallback(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	layoutJSON := `[
		{"type": "unknown_future_section", "props": {"foo": "bar"}},
		{"type": "hero", "props": {"title": "Sağlam Vitrin"}}
	]`

	data := StorefrontData{
		StoreName:  "Vera Atelier",
		LayoutJSON: layoutJSON,
	}

	if err := Render(rec, req, data); err != nil {
		t.Fatalf("Render should not fail on unknown section type: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Sağlam Vitrin") {
		t.Errorf("expected known hero section to render properly")
	}
	if strings.Contains(body, "unknown_future_section") {
		t.Errorf("unknown section should not be rendered into markup")
	}
}

func TestRenderStorefront_MalformedLayoutJSONFallback(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	data := StorefrontData{
		StoreName:  "Vera Atelier",
		LayoutJSON: "{invalid-json-string-malformed",
		HeroTitle:  "Varsayılan Kahraman",
	}

	// Should not crash or fail; falls back to default layout gracefully
	if err := Render(rec, req, data); err != nil {
		t.Fatalf("Render should not fail on malformed layout_json: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Varsayılan Kahraman") {
		t.Errorf("expected fallback default hero title")
	}
}

func TestRenderStorefront_ConditionalGET(t *testing.T) {
	data := StorefrontData{
		StoreName: "Vera Atelier",
		HeroTitle: "Zarif Seramik",
	}

	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	if err := Render(rec1, req1, data); err != nil {
		t.Fatalf("first Render: %v", err)
	}
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}
	etag := rec1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header in response")
	}
	if !strings.Contains(rec1.Header().Get("Cache-Control"), "stale-while-revalidate") {
		t.Errorf("expected Cache-Control header with stale-while-revalidate, got %s", rec1.Header().Get("Cache-Control"))
	}

	// Conditional GET request with If-None-Match
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("If-None-Match", etag)
	if err := Render(rec2, req2, data); err != nil {
		t.Fatalf("second Render: %v", err)
	}
	if rec2.Code != http.StatusNotModified {
		t.Fatalf("expected 304 Not Modified, got %d", rec2.Code)
	}
	if rec2.Body.Len() != 0 {
		t.Errorf("expected empty body for 304 Not Modified, got %d bytes", rec2.Body.Len())
	}
}
