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
