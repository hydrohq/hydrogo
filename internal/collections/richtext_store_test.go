package collections

import (
	"strings"
	"testing"
)

// Rich text reaches public storefronts that render it with innerHTML, so a
// script must never survive a save -- on the record or on a translation.
func TestRichTextIsSanitizedOnRecordsAndTranslations(t *testing.T) {
	schema := Schema{ID: "products", Label: "Products", Fields: []Field{
		{ID: "name", Label: "Name", Kind: Text},
		{ID: "description", Label: "Description", Kind: RichText, Localized: true},
	}}
	if err := schema.Validate(); err != nil {
		t.Fatal(err)
	}
	payload := `<p>ok</p><img src=x onerror="alert(document.cookie)"><script>alert(1)</script>`
	record, err := schema.ValidateRecord(map[string]any{"name": "Kemer", "description": payload})
	if err != nil {
		t.Fatal(err)
	}
	stored := record["description"].(string)
	if strings.Contains(stored, "onerror") || strings.Contains(stored, "<script") || !strings.Contains(stored, "<p>ok</p>") {
		t.Fatalf("stored record rich text = %q", stored)
	}
	translation, err := schema.ValidateTranslation(map[string]any{"description": payload})
	if err != nil {
		t.Fatal(err)
	}
	if text := translation["description"].(string); strings.Contains(text, "onerror") || strings.Contains(text, "<script") {
		t.Fatalf("stored translation rich text = %q", text)
	}
	plain, err := schema.ValidateRecord(map[string]any{"name": "<b>not rich</b>"})
	if err != nil {
		t.Fatal(err)
	}
	if plain["name"] != "<b>not rich</b>" {
		t.Fatalf("plain text fields are stored verbatim (escaped by the renderer), got %q", plain["name"])
	}
}
