package contract_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/commerce"
	"github.com/ardaozyildirim/hydrogo/modules/news"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func TestPublicModelsUseDocumentedJSONFieldNames(t *testing.T) {
	for name, test := range map[string]struct {
		value    any
		required []string
	}{
		"page":     {pages.Page{ID: "page-1", PublicationGeneration: 1}, []string{`"id":"page-1"`, `"publication_generation":1`}},
		"news":     {news.Item{ID: "news-1", ScheduleGeneration: 1}, []string{`"id":"news-1"`, `"schedule_generation":1`}},
		"product":  {commerce.Product{ID: "product-1", SKU: "SKU", PriceCents: 49900}, []string{`"id":"product-1"`, `"sku":"SKU"`, `"price_cents":49900`}},
		"checkout": {commerce.CheckoutItem{ProductID: "product-1", Quantity: 1}, []string{`"product_id":"product-1"`, `"quantity":1`}},
	} {
		encoded, err := json.Marshal(test.value)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, want := range test.required {
			if !strings.Contains(string(encoded), want) {
				t.Errorf("%s JSON missing %s: %s", name, want, encoded)
			}
		}
		if strings.Contains(string(encoded), `"ID"`) || strings.Contains(string(encoded), `"PriceCents"`) {
			t.Errorf("%s leaked Go field casing: %s", name, encoded)
		}
	}
}
