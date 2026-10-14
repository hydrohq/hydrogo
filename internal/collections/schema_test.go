package collections

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestSchemaValidationEnforcesFieldTypesAndConstraints(t *testing.T) {
	minimum, maximum := 1.0, 10.0
	schema := Schema{ID: "events", Label: "Events", Fields: []Field{{ID: "title", Label: "Title", Kind: Text, Required: true, Unique: true}, {ID: "seats", Label: "Seats", Kind: Number, Min: &minimum, Max: &maximum}, {ID: "active", Label: "Active", Kind: Boolean}, {ID: "kind", Label: "Kind", Kind: Select, Options: []string{"public", "private"}}}}
	if err := schema.Validate(); err != nil {
		t.Fatal(err)
	}
	valid, err := schema.ValidateRecord(map[string]any{"title": "Concert", "seats": float64(5), "active": false, "kind": "public"})
	if err != nil || valid["active"] != false {
		t.Fatalf("record=%v err=%v", valid, err)
	}
	for _, data := range []map[string]any{{"title": ""}, {"title": "Ok", "seats": 0.0}, {"title": "Ok", "kind": "unknown"}, {"title": "Ok", "extra": "field"}} {
		if _, err := schema.ValidateRecord(data); !errors.Is(err, ErrInvalidRecord) {
			t.Errorf("data %v error=%v", data, err)
		}
	}
}

func TestSchemaRejectsUnsafeIdentifiersUnknownKindsAndDuplicateFields(t *testing.T) {
	for _, schema := range []Schema{
		{ID: "../events", Label: "Events", Fields: []Field{{ID: "title", Label: "Title", Kind: Text}}},
		{ID: "events", Label: "Events", Fields: []Field{{ID: "title", Label: "Title", Kind: "sql"}}},
		{ID: "events", Label: "Events", Fields: []Field{{ID: "title", Label: "Title", Kind: Text}, {ID: "title", Label: "Again", Kind: Text}}},
	} {
		if err := schema.Validate(); !errors.Is(err, ErrInvalidSchema) {
			t.Fatalf("schema=%+v err=%v", schema, err)
		}
	}
}

func TestMultiRelationAndDatetimeTypes(t *testing.T) {
	schema := Schema{ID: "articles", Label: "Articles", Fields: []Field{{ID: "published_at", Label: "Published", Kind: DateTime}, {ID: "labels", Label: "Labels", Kind: MultiSelect, Options: []string{"news", "culture"}}, {ID: "topics", Label: "Topics", Kind: RelationMany, Relation: "topics"}}}
	if err := schema.Validate(); err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"published_at": "2026-09-28T10:00:00Z", "labels": []any{"news"}, "topics": []any{"topic_1"}}
	encoded, _ := json.Marshal(input)
	var decoded map[string]any
	_ = json.Unmarshal(encoded, &decoded)
	if _, err := schema.ValidateRecord(decoded); err != nil {
		t.Fatalf("valid typed fields rejected: %v", err)
	}
}

func TestLocalizedFieldsAreLimitedToTranslatableKinds(t *testing.T) {
	for _, field := range []Field{
		{ID: "flag", Label: "Flag", Kind: Boolean, Localized: true},
		{ID: "owner", Label: "Owner", Kind: Relation, Relation: "team", Localized: true},
		{ID: "slug", Label: "Slug", Kind: Text, Unique: true, Localized: true},
	} {
		if err := (Schema{ID: "items", Label: "Items", Fields: []Field{field}}).Validate(); err == nil {
			t.Fatalf("localized %s field was accepted", field.Kind)
		}
	}
	schema := Schema{ID: "items", Label: "Items", Fields: []Field{{ID: "title", Label: "Title", Kind: Text, Required: true, Localized: true}, {ID: "rank", Label: "Rank", Kind: Number}}}
	if err := schema.Validate(); err != nil || !schema.Localized() {
		t.Fatalf("localized text field rejected: %v", err)
	}
	clean, err := schema.ValidateTranslation(map[string]any{"title": ""})
	if err != nil || len(clean) != 0 {
		t.Fatalf("empty translation should be valid and dropped: %v %v", clean, err)
	}
	if _, err := schema.ValidateTranslation(map[string]any{"rank": 1}); err == nil {
		t.Fatal("translation of an unlocalized field was accepted")
	}
	for locale, want := range map[string]bool{"en": true, "pt-br": true, "EN": false, "": false, "e": false, "en_US": false} {
		if ValidLocale(locale) != want {
			t.Fatalf("ValidLocale(%q) != %v", locale, want)
		}
	}
}
