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

func TestRenderCreateFormSupportsTextareaAndReadonlyFields(t *testing.T) {
	collection := Collection{ID: "news", Label: "News", Fields: []Field{{ID: "body", Label: "Body", Kind: FieldText, Widget: "textarea", Required: true}, {ID: "status", Label: "Status", Kind: FieldStatus, ReadOnly: true, Options: []string{"draft"}}}, Actions: []Action{{Label: "News", Href: "/admin/news/item"}}}
	var out bytes.Buffer
	if err := RenderCreateForm(&out, FormData{Collection: collection, Action: "/admin/news/item", CSRFToken: "csrf", SubmitLabel: "Save draft", Values: map[string]string{"body": "<strong>safe</strong>", "status": "draft"}}); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{`class="richtext-field"`, `name="body"`, "&lt;strong&gt;safe&lt;/strong&gt;", "Save draft"} {
		if !strings.Contains(html, want) {
			t.Errorf("form missing %q: %s", want, html)
		}
	}
	if strings.Contains(html, `name="status"`) {
		t.Fatalf("readonly field rendered as writable: %s", html)
	}
}

func TestRenderCreateFormCanLeaveSubmitToModuleExtension(t *testing.T) {
	collection := Collection{ID: "pages", Label: "Pages", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, Required: true}}, Actions: []Action{{Label: "Pages", Href: "/admin/pages/demo"}}}
	var out bytes.Buffer
	if err := RenderCreateForm(&out, FormData{Collection: collection, Action: "/admin/pages/demo", CSRFToken: "csrf", HideSubmit: true}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), `<button type="submit">`) || !strings.Contains(out.String(), `name="_csrf" value="csrf"`) {
		t.Fatalf("extension form output = %s", out.String())
	}
}

func TestRenderCreateFormSupportsRichTextWithNonce(t *testing.T) {
	collection := Collection{ID: "commerce", Label: "Commerce", Fields: []Field{{ID: "description", Label: "Description", Kind: FieldText, Widget: "richtext"}}, Actions: []Action{{Label: "Catalog", Href: "/admin/commerce/products"}}}
	var out bytes.Buffer
	if err := RenderCreateForm(&out, FormData{Collection: collection, Action: "/admin/commerce/products", CSRFToken: "csrf", Nonce: "nonce", Values: map[string]string{"description": "hello"}}); err != nil {
		t.Fatalf("rich text form render failed: %v", err)
	}
	if !strings.Contains(out.String(), "/vendor/quill/quill.js") {
		t.Fatalf("rich text assets missing: %s", out.String())
	}
}

func TestRenderCreateFormSupportsHiddenWidgetAndOptionsSelect(t *testing.T) {
	collection := Collection{
		ID:    "news",
		Label: "News",
		Fields: []Field{
			{ID: "locale", Label: "Dil", Kind: FieldText, Widget: "hidden"},
			{ID: "category", Label: "Kategori", Kind: FieldText, Options: []string{"tech", "business"}},
			{ID: "title", Label: "Başlık", Kind: FieldText, Required: true},
		},
		Actions: []Action{{Label: "News", Href: "/admin/news"}},
	}
	var out bytes.Buffer
	if err := RenderCreateForm(&out, FormData{
		Collection: collection,
		Action:     "/admin/news",
		CSRFToken:  "csrf",
		Values:     map[string]string{"locale": "tr", "category": "tech"},
	}); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	if !strings.Contains(html, `<input type="hidden" name="locale" value="tr">`) {
		t.Fatalf("hidden input missing: %s", html)
	}
	if !strings.Contains(html, `<select name="category"`) || !strings.Contains(html, `<option value="tech" selected>tech</option>`) {
		t.Fatalf("select options missing: %s", html)
	}
}
