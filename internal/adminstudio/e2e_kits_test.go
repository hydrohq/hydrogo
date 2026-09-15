package adminstudio

import "testing"

func TestE2EManifestCarriesCollectionKits(t *testing.T) {
	m := E2EManifestForCollections([]Collection{{ID: "articles", Actions: []Action{{Label: "Articles", Href: "/admin/articles"}}, Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText}}, Kits: []Kit{KitSEO, KitSearch}}})
	if len(m.Admin.Collections) != 1 || len(m.Admin.Collections[0].Kits) != 2 {
		t.Fatalf("manifest kits = %#v", m.Admin.Collections)
	}
}
