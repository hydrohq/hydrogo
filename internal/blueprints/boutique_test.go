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
