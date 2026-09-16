package generator

import (
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
)

func TestAdminCollectionCarriesOptionalKits(t *testing.T) {
	spec := Spec{Name: "article", Table: "articles", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"read"}, Dialects: []string{"sqlite"}, Kits: []string{"seo", "search"}}
	c, err := AdminCollection(spec)
	if err != nil || len(c.Kits) != 2 || string(c.Kits[0]) != "search" || string(c.Kits[1]) != "seo" {
		t.Fatalf("kits = %#v, err=%v", c.Kits, err)
	}
}

func TestAdminCollectionCarriesResolvedKitDependencies(t *testing.T) {
	spec := Spec{Name: "order", Table: "orders", Fields: []Field{{Name: "total", Type: "decimal"}}, Permissions: []string{"read"}, Dialects: []string{"sqlite"}, Kits: []string{"commerce"}}
	collection, err := AdminCollection(spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(collection.Kits) != 2 || collection.Kits[0] != adminstudio.KitCommerce || collection.Kits[1] != adminstudio.KitForms {
		t.Fatalf("resolved collection kits = %#v", collection.Kits)
	}
}
