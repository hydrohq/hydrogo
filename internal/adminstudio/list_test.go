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
	for _, want := range []string{"<caption class=\"visually-hidden\">Products listesi</caption>", `<th scope="col">Title</th>`, `<th scope="col">Status</th>`, "Variants", "&lt;img src=x&gt;"} {
		if !strings.Contains(html, want) {
			t.Errorf("list missing %q: %s", want, html)
		}
	}
	if strings.Contains(html, "Internal") || strings.Contains(html, ">hidden<") || strings.Contains(html, "<img src=x>") {
		t.Fatalf("list leaked hidden or unsafe data: %s", html)
	}
}

func TestRenderListProvidesAccessibleEmptyState(t *testing.T) {
	collection := Collection{ID: "products", Label: "Products", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}}, Actions: []Action{{Label: "Yeni", Href: "/admin/products/new"}}}
	var out bytes.Buffer
	if err := RenderList(&out, ListData{Collection: collection, EmptyMessage: "Henüz ürün eklenmedi."}); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{`colspan="2"`, `class="empty-state"`, `role="status"`, "Henüz ürün eklenmedi."} {
		if !strings.Contains(html, want) {
			t.Errorf("empty state missing %q: %s", want, html)
		}
	}
}

func TestRenderListUsesSafeDefaultEmptyMessage(t *testing.T) {
	var out bytes.Buffer
	if err := RenderList(&out, ListData{Collection: Collection{ID: "products", Label: "Products", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}}, Actions: []Action{{Label: "Yeni", Href: "/admin/products/new"}}}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Bu koleksiyonda henüz kayıt bulunmuyor.") {
		t.Fatalf("default empty message missing: %s", out.String())
	}
}

func TestRenderListOptInBulkSelection(t *testing.T) {
	collection := Collection{ID: "articles", Label: "Articles", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}}, Actions: []Action{{Label: "Yeni", Href: "/admin/articles/new"}}, BulkActions: []Action{{Label: "Yayınla", Href: "/admin/articles/bulk/publish"}}}
	var out strings.Builder
	if err := RenderList(&out, ListData{Collection: collection, Rows: []Row{{ID: "a1", Values: map[string]string{"title": "Hello"}}}}); err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`data-bulk-select-all`, `value="a1"`, `data-bulk-action="Yayınla"`} {
		if !strings.Contains(out.String(), marker) {
			t.Fatalf("bulk list missing %q: %s", marker, out.String())
		}
	}
}

func TestRenderListBulkSelectionRequiresRowID(t *testing.T) {
	collection := Collection{ID: "articles", Label: "Articles", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}}, Actions: []Action{{Label: "Yeni", Href: "/admin/articles/new"}}, BulkActions: []Action{{Label: "Yayınla", Href: "/admin/articles/bulk/publish"}}}
	if err := RenderList(&strings.Builder{}, ListData{Collection: collection, Rows: []Row{{Values: map[string]string{"title": "Hello"}}}}); err != ErrInvalidForm {
		t.Fatalf("missing row ID error = %v", err)
	}
}
