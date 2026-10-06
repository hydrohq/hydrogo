package adminstudio

import (
	"encoding/json"
	"testing"
)

func TestParseManifestValidatesAndSortsResources(t *testing.T) {
	data, err := json.Marshal(Manifest{Version: "v1", Resources: []Collection{
		{ID: "zebra", Label: "Zebra", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText}}, Actions: []Action{{Label: "Open", Href: "/admin/zebra"}}},
		{ID: "article", Label: "Article", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText}}, Actions: []Action{{Label: "Open", Href: "/admin/article"}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := ParseManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	items := registry.Collections()
	if len(items) != 2 || items[0].ID != "article" || items[1].ID != "zebra" {
		t.Fatalf("resources not validated and sorted: %#v", items)
	}
}

func TestParseManifestRejectsInvalidDocuments(t *testing.T) {
	for _, data := range [][]byte{
		nil,
		[]byte(`{"version":"v2","resources":[]}`),
		[]byte(`{"version":"v1","resources":[],"other":true}`),
		[]byte(`{"version":"v1","resources":[]} {}`),
		[]byte(`{"version":"v1","resources":[]}`),
		make([]byte, (1<<20)+1),
	} {
		if _, err := ParseManifest(data); err == nil {
			t.Fatalf("invalid manifest accepted: %q", data[:min(len(data), 64)])
		}
	}
}
