package adminstudio

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderListShowsOnlyConfiguredColumnsAndEscapesRows(t *testing.T) {
	collection := Collection{ID: "products", Label: "Products", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}, {ID: "internal", Label: "Internal", Kind: FieldText}, {ID: "status", Label: "Status", Kind: FieldStatus, VisibleInList: true}}, Actions: []Action{{Label: "Catalog", Href: "/admin/commerce/products"}}}
	var out bytes.Buffer
	err := RenderList(&out, ListData{Collection: collection, Rows: []Row{{Values: map[string]string{"title": `<img src=x>`, "internal": "hidden", "status": "published"}, Actions: []Action{{Label: "Variants", Href: "/admin/commerce/products/p1/variants"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{"<th>Title</th>", "<th>Status</th>", "Variants", "&lt;img src=x&gt;"} {
		if !strings.Contains(html, want) {
			t.Errorf("list missing %q: %s", want, html)
		}
	}
	if strings.Contains(html, "Internal") || strings.Contains(html, "hidden") || strings.Contains(html, "<img src=x>") {
		t.Fatalf("list leaked hidden or unsafe data: %s", html)
	}
}
