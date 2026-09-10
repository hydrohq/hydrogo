package adminstudio

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderCreateFormUsesCollectionMetadataAndEscapesValues(t *testing.T) {
	collection := Collection{ID: "products", Label: "Products", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, Required: true}, {ID: "price_cents", Label: "Price", Kind: FieldNumber, Required: true}, {ID: "status", Label: "Status", Kind: FieldStatus, Options: []string{"draft", "published"}}}, Actions: []Action{{Label: "Catalog", Href: "/admin/commerce/products"}}}
	var out bytes.Buffer
	err := RenderCreateForm(&out, FormData{Collection: collection, Action: "/admin/commerce/products", CSRFToken: "csrf", Values: map[string]string{"title": `<script>alert(1)</script>`}})
	if err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{`name="title"`, `type="number"`, `name="status"`, `value="published"`, `name="_csrf" value="csrf"`} {
		if !strings.Contains(html, want) {
			t.Errorf("form missing %q: %s", want, html)
		}
	}
	if strings.Contains(html, `<script>`) || !strings.Contains(html, "&lt;script&gt;") {
		t.Fatalf("form did not escape value: %s", html)
	}
}

func TestRenderCreateFormRejectsUnsafeAction(t *testing.T) {
	collection := Collection{ID: "products", Label: "Products", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText}}, Actions: []Action{{Label: "Catalog", Href: "/admin/commerce/products"}}}
	if err := RenderCreateForm(&bytes.Buffer{}, FormData{Collection: collection, Action: "https://example.test", CSRFToken: "csrf"}); err == nil {
		t.Fatal("unsafe action accepted")
	}
}
