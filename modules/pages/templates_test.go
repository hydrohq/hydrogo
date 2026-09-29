package pages

import "testing"

func TestBuiltinLayoutTemplatesResolveValidatedBlocks(t *testing.T) {
	for _, layout := range BuiltinLayoutTemplates() {
		blocks, err := ResolveLayoutTemplate(layout.ID, nil)
		if err != nil {
			t.Fatalf("resolve %s: %v", layout.ID, err)
		}
		if len(blocks) != len(layout.Blocks) {
			t.Fatalf("%s blocks=%d want %d", layout.ID, len(blocks), len(layout.Blocks))
		}
		if err := ValidateBlocks(blocks); err != nil {
			t.Fatalf("%s has invalid blocks: %v", layout.ID, err)
		}
	}
}

func TestLayoutTemplateExtensionsAreValidatedAndCannotReplaceBuiltins(t *testing.T) {
	custom := LayoutTemplate{ID: "edition_layout", Name: "Edition page", Description: "Edition provided page", Blocks: []Block{{Type: "hero", Payload: []byte(`{"title":"Edition title"}`)}}}
	blocks, err := ResolveLayoutTemplate(custom.ID, []LayoutTemplate{custom})
	if err != nil || len(blocks) != 1 {
		t.Fatalf("extension blocks=%#v error=%v", blocks, err)
	}
	for _, invalid := range []LayoutTemplate{
		{ID: "blank", Name: "Shadow", Blocks: nil},
		{ID: "bad/id", Name: "Invalid ID"},
		{ID: "unknown", Name: "Invalid block", Blocks: []Block{{Type: "hero", Payload: []byte(`{}`)}}},
	} {
		if _, err := ResolveLayoutTemplate("blank", []LayoutTemplate{invalid}); err == nil {
			t.Fatalf("invalid extension accepted: %#v", invalid)
		}
	}
	if _, err := ResolveLayoutTemplate("missing", nil); err == nil {
		t.Fatal("unknown template accepted")
	}
}
