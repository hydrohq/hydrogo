package adminstudio

import "testing"

func TestFilterActiveCollectionsFollowsModuleLifecycle(t *testing.T) {
	collections := []Collection{
		{ID: "commerce", Label: "Products", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText}}, Actions: []Action{{Label: "List", Href: "/admin/commerce"}}},
		{ID: "commerce_variants", Label: "Variants", Fields: []Field{{ID: "sku", Label: "SKU", Kind: FieldText}}, Actions: []Action{{Label: "List", Href: "/admin/commerce"}}},
		{ID: "news", Label: "News", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText}}, Actions: []Action{{Label: "List", Href: "/admin/news"}}},
	}
	got := FilterActiveCollections(collections, map[string]bool{"commerce": true})
	if len(got) != 2 || got[0].ID != "commerce" || got[1].ID != "commerce_variants" {
		t.Fatalf("unexpected active collections: %#v", got)
	}
	got[0].Fields[0].Label = "mutated"
	if collections[0].Fields[0].Label == "mutated" {
		t.Fatal("filter leaked collection backing storage")
	}
}
