package openapi

import (
	"encoding/json"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
)

func TestOpenAPIGenerator(t *testing.T) {
	definitions := []contenttypes.Definition{
		{
			ID:     "product",
			Label:  "Products",
			Public: true,
			Drafts: true,
			Fields: []contenttypes.Field{
				{ID: "title", Kind: "text", Required: true},
				{ID: "price", Kind: "number", Required: true},
				{ID: "in_stock", Kind: "boolean"},
				{ID: "status", Kind: "select", Options: []string{"available", "backorder", "discontinued"}},
				{ID: "categories", Kind: "relation", To: "category", Many: true},
				{ID: "tags", Kind: "text", List: true},
			},
		},
	}

	docBytes, err := BuildDocument(nil, definitions)
	if err != nil {
		t.Fatalf("BuildDocument failed: %v", err)
	}

	var doc struct {
		OpenAPI string `json:"openapi"`
		Paths   map[string]struct {
			Get    map[string]any `json:"get"`
			Post   map[string]any `json:"post"`
			Put    map[string]any `json:"put"`
			Delete map[string]any `json:"delete"`
		} `json:"paths"`
		Components struct {
			Schemas map[string]struct {
				Type       string                    `json:"type"`
				Required   []string                  `json:"required"`
				Properties map[string]map[string]any `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}

	if err := json.Unmarshal(docBytes, &doc); err != nil {
		t.Fatalf("Unmarshal generated OpenAPI doc failed: %v", err)
	}

	if doc.OpenAPI != "3.1.0" {
		t.Fatalf("expected openapi 3.1.0, got %s", doc.OpenAPI)
	}

	// Verify Schema
	schema, ok := doc.Components.Schemas["ProductRecord"]
	if !ok {
		t.Fatalf("expected ProductRecord schema in components, got schemas: %+v", doc.Components.Schemas)
	}

	if schema.Properties["title"]["type"] != "string" {
		t.Fatalf("expected title type string, got: %+v", schema.Properties["title"])
	}
	if schema.Properties["price"]["type"] != "number" {
		t.Fatalf("expected price type number, got: %+v", schema.Properties["price"])
	}
	if schema.Properties["in_stock"]["type"] != "boolean" {
		t.Fatalf("expected in_stock type boolean, got: %+v", schema.Properties["in_stock"])
	}
	if schema.Properties["status"]["enum"] == nil {
		t.Fatalf("expected status enum, got: %+v", schema.Properties["status"])
	}
	if schema.Properties["categories"]["type"] != "array" {
		t.Fatalf("expected categories type array, got: %+v", schema.Properties["categories"])
	}
	if schema.Properties["tags"]["type"] != "array" {
		t.Fatalf("expected tags type array, got: %+v", schema.Properties["tags"])
	}

	// Verify Paths
	if _, ok := doc.Paths["/api/v1/content/product"]; !ok {
		t.Fatalf("expected path /api/v1/content/product, got: %+v", doc.Paths)
	}
	if _, ok := doc.Paths["/api/v1/content/product/{id}"]; !ok {
		t.Fatalf("expected path /api/v1/content/product/{id}, got: %+v", doc.Paths)
	}
}
