package generator

import "testing"

func TestAdminCollectionCarriesOptionalKits(t *testing.T) {
	spec := Spec{Name: "article", Table: "articles", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"read"}, Dialects: []string{"sqlite"}, Kits: []string{"seo", "search"}}
	c, err := AdminCollection(spec)
	if err != nil || len(c.Kits) != 2 || string(c.Kits[0]) != "seo" {
		t.Fatalf("kits = %#v, err=%v", c.Kits, err)
	}
}
