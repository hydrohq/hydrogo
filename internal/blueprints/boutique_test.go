package blueprints

import (
	"testing"
)

func TestBoutiqueDefinitions(t *testing.T) {
	defs := BoutiqueDefinitions()
	if len(defs) != 3 {
		t.Fatalf("expected 3 boutique definitions, got %d", len(defs))
	}

	foundSettings := false
	foundProducts := false
	foundOrders := false

	for _, d := range defs {
		switch d.ID {
		case "store_settings":
			foundSettings = true
		case "products":
			foundProducts = true
		case "orders":
			foundOrders = true
		}
	}

	if !foundSettings || !foundProducts || !foundOrders {
		t.Errorf("missing core definitions: settings=%v, products=%v, orders=%v", foundSettings, foundProducts, foundOrders)
	}
}

func TestGenerateLuxuryCatalog(t *testing.T) {
	settings, products := GenerateLuxuryCatalog("Vera Deri", "+905551234567", "El yapımı deri cüzdan ve çanta, taba tonları")

	if settings["brand_name"] != "Vera Deri" {
		t.Errorf("expected brand_name 'Vera Deri', got %v", settings["brand_name"])
	}
	if settings["whatsapp_number"] != "+905551234567" {
		t.Errorf("expected whatsapp_number '+905551234567', got %v", settings["whatsapp_number"])
	}
	if len(products) != 6 {
		t.Errorf("expected 6 luxury products, got %d", len(products))
	}
	for i, p := range products {
		if p["title"] == "" || p["price"] == nil || p["slug"] == "" {
			t.Errorf("product %d is missing required fields: %+v", i, p)
		}
	}
}
