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

func TestRegistryAcceptsCollectionsWithoutActionsAndChecksSource(t *testing.T) {
	base := Collection{ID: "faq", Label: "FAQ", Source: SourceContentType, APIPath: "/api/v1/admin/content/faq", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText}}}
	if _, err := NewRegistry([]Collection{base}); err != nil {
		t.Fatalf("collection without actions rejected: %v", err)
	}
	badSource := base
	badSource.Source = "plugin"
	if _, err := NewRegistry([]Collection{badSource}); err == nil {
		t.Fatal("unknown source accepted")
	}
	badPath := base
	badPath.APIPath = "https://example.test"
	if _, err := NewRegistry([]Collection{badPath}); err == nil {
		t.Fatal("external api path accepted")
	}
}

func TestWorkflowNeedsVersionedRecordsButNotRevisions(t *testing.T) {
	workflow := &Workflow{StateField: "status", States: []PublicationState{StateDraft, StatePublished}, Transitions: map[PublicationState][]PublicationState{StateDraft: {StatePublished}}}
	collection := Collection{ID: "faq", Label: "FAQ", Versioned: true, Workflow: workflow,
		Fields:  []Field{{ID: "status", Label: "Status", Kind: FieldSelect, Options: []string{"draft", "published"}}},
		Actions: []Action{{Label: "New", Href: "/admin/resources/faq/new"}}}
	if _, err := NewRegistry([]Collection{collection}); err != nil {
		t.Fatalf("workflow without revisions: %v", err)
	}
	collection.Versioned = false
	if _, err := NewRegistry([]Collection{collection}); err == nil {
		t.Fatal("workflow without versioned records was accepted")
	}
}
