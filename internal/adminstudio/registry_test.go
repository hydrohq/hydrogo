package adminstudio

import "testing"

func TestFieldRegistrySupportsProviderNeutralKinds(t *testing.T) {
	registry := NewFieldRegistry()
	for _, kind := range []FieldKind{FieldText, FieldNumber, FieldBoolean, FieldDate, FieldMedia, FieldRichText, FieldSelect, FieldRelation, FieldArray, FieldBlocks} {
		if !registry.Supports(kind) {
			t.Fatalf("field kind %q is not registered", kind)
		}
	}
	if registry.Supports(FieldKind("unknown")) {
		t.Fatal("unknown field kind accepted")
	}
}

func TestRegistrySortsAndDefensivelyCopiesCollections(t *testing.T) {
	registry, err := NewRegistry([]Collection{
		{ID: "pages", Label: "Pages", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText}}, Actions: []Action{{Label: "Open", Href: "/admin/pages/demo"}}},
		{ID: "commerce", Label: "Commerce", Fields: []Field{{ID: "sku", Label: "SKU", Kind: FieldText}}, Actions: []Action{{Label: "Open", Href: "/admin/commerce/products"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	collections := registry.Collections()
	if len(collections) != 2 || collections[0].ID != "commerce" {
		t.Fatalf("collections = %#v", collections)
	}
	collections[0].Label = "changed"
	if registry.Collections()[0].Label == "changed" {
		t.Fatal("registry leaked collection slice")
	}
}

func TestRegistryRejectsAmbiguousOrUnsafeMetadata(t *testing.T) {
	base := Collection{ID: "pages", Label: "Pages", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText}}, Actions: []Action{{Label: "Open", Href: "/admin/pages/demo"}}}
	duplicate := base
	if _, err := NewRegistry([]Collection{base, duplicate}); err == nil {
		t.Fatal("duplicate collection accepted")
	}
	unsafe := base
	unsafe.Actions = []Action{{Label: "External", Href: "https://example.test"}}
	if _, err := NewRegistry([]Collection{unsafe}); err == nil {
		t.Fatal("external action accepted")
	}
}

func TestE2EManifestIsSafeAndDeterministic(t *testing.T) {
	registry, err := NewRegistry([]Collection{{ID: "pages", Label: "Pages", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, Required: true}}, Actions: []Action{{Label: "Open", Href: "/admin/pages"}}}})
	if err != nil {
		t.Fatal(err)
	}
	manifest := registry.E2EManifest()
	if manifest.Version != 1 || manifest.Admin.LoginPath != "/login" || len(manifest.Admin.Collections) != 1 {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	collection := manifest.Admin.Collections[0]
	if collection.Route != "/admin/pages/new" || collection.Fields[0].TestID != "admin-field-pages-title" {
		t.Fatalf("unexpected collection manifest: %#v", collection)
	}
}
