package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var out bytes.Buffer
	if err := c.Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestButtonEscapesAndAllowlistsHTMX(t *testing.T) {
	out := render(t, Button(ButtonProps{Label: `<script>alert(1)</script>`, HTMX: HTMX{Post: "/save", Target: "#result", Swap: "outerHTML"}}))
	if !strings.Contains(out, `hx-post="/save"`) || !strings.Contains(out, `hx-target="#result"`) || !strings.Contains(out, `hx-swap="outerHTML"`) {
		t.Fatalf("missing allowlisted attributes: %s", out)
	}
	if strings.Contains(out, "<script>") || strings.Contains(out, "hx-on") {
		t.Fatalf("unsafe or non-allowlisted output: %s", out)
	}
}

func TestMediaPickerEscapesValuesAndUsesAdminPickerContract(t *testing.T) {
	out := render(t, MediaPicker(MediaPickerProps{ID: `cover"`, Name: "cover_asset_id", Label: "Cover image", Value: `<script>`}))
	for _, marker := range []string{`data-media-target="cover&#34;"`, `name="cover_asset_id"`, `href="/admin/media?select=cover%22"`, `media-select-clear`, `value="&lt;script&gt;"`} {
		if !strings.Contains(out, marker) {
			t.Fatalf("media picker missing %q: %s", marker, out)
		}
	}
	if strings.Contains(out, `<script>`) {
		t.Fatalf("media value was rendered as markup: %s", out)
	}
}

func TestItemListEditorEscapesValuesAndUsesSharedAdminContract(t *testing.T) {
	out := render(t, ItemListEditor(ItemListEditorProps{ID: "keywords", Name: "keywords", Label: "Keywords", Value: `["one","two & <three>"]`}))
	for _, marker := range []string{`data-item-list`, `data-field-name="keywords"`, `aria-label="Keywords: madde ekle"`, `name="keywords"`, `&#34;two &amp; &lt;three&gt;&#34;`} {
		if !strings.Contains(out, marker) {
			t.Fatalf("item list editor missing %q: %s", marker, out)
		}
	}
	var outWithoutID bytes.Buffer
	if err := ItemListEditor(ItemListEditorProps{Label: "Keywords"}).Render(context.Background(), &outWithoutID); err == nil {
		t.Fatal("item list editor accepted a missing id")
	}
}

func TestTypedLinksRequireLocalPathsAndEscapeLabels(t *testing.T) {
	out := render(t, Link(LinkProps{Href: "/admin/site/seo", Label: "<SEO>"}))
	if !strings.Contains(out, `href="/admin/site/seo"`) || !strings.Contains(out, "&lt;SEO&gt;") {
		t.Fatalf("typed link output=%s", out)
	}
	for _, href := range []string{"https://example.test", "//example.test", "/path?next=evil"} {
		if err := Link(LinkProps{Href: href, Label: "open"}).Render(context.Background(), &bytes.Buffer{}); err == nil {
			t.Errorf("unsafe link %q accepted", href)
		}
	}
}

func TestFormAndInputRenderCSRFAndEscapedValues(t *testing.T) {
	out := render(t, Form(FormProps{Action: `/projects/<x>`, Class: "studio-form", CSRF: `a"b`, Body: Input(InputProps{ID: "title", Name: "title", Label: "Title", Required: true})}))
	if !strings.Contains(out, `name="_csrf" value="a&#34;b"`) || !strings.Contains(out, `action="/projects/&lt;x&gt;"`) || !strings.Contains(out, `class="studio-form"`) || !strings.Contains(out, `required="required"`) {
		t.Fatalf("unexpected form output: %s", out)
	}
}

func TestInputsRenderTypedLengthConstraints(t *testing.T) {
	out := render(t, Input(InputProps{ID: "email", Name: "email", Label: "Email", Type: "email", MinLength: 4, HasMinLength: true, MaxLength: 64, HasMaxLength: true}))
	for _, marker := range []string{`type="email"`, `minlength="4"`, `maxlength="64"`} {
		if !strings.Contains(out, marker) {
			t.Fatalf("missing %q: %s", marker, out)
		}
	}
	out = render(t, Textarea(TextareaProps{ID: "bio", Name: "bio", Label: "Bio", MinLength: 2, HasMinLength: true, MaxLength: 10, HasMaxLength: true}))
	if !strings.Contains(out, `minlength="2"`) || !strings.Contains(out, `maxlength="10"`) {
		t.Fatalf("textarea constraints missing: %s", out)
	}
}

func TestTypedDataComponentsEscapeValuesAndRenderHTMX(t *testing.T) {
	out := render(t, Select(SelectProps{ID: "status", Name: "status", Label: "Status", Value: "<x>", Options: []string{"<x>", "published"}, HTMX: HTMX{Get: "/admin/articles", Target: "#list", Swap: "innerHTML"}}))
	if !strings.Contains(out, `value="&lt;x&gt;" selected="selected"`) || !strings.Contains(out, `hx-target="#list"`) || strings.Contains(out, "<x>") {
		t.Fatalf("select output was not safely rendered: %s", out)
	}
	out = render(t, Table(TableProps{Caption: "Articles", Columns: []string{"Title"}, Rows: [][]string{{`<script>alert(1)</script>`}}}))
	if strings.Contains(out, "<script>") || !strings.Contains(out, "&lt;script&gt;") {
		t.Fatalf("table output was not escaped: %s", out)
	}
}

func TestTableRowActionIsLocalAndCSRFProtected(t *testing.T) {
	out := render(t, Table(TableProps{Caption: "Redirects", Columns: []string{"Path"}, Rows: [][]string{{"/old"}}, RowActions: [][]TableAction{{{Label: "Remove", Action: "/admin/redirects/id/delete", CSRF: "token", Variant: Danger}}}}))
	for _, marker := range []string{`<th scope="col">İşlem</th>`, `method="post"`, `name="_csrf" value="token"`, `>Remove</button>`} {
		if !strings.Contains(out, marker) {
			t.Fatalf("table action missing %q: %s", marker, out)
		}
	}
	if strings.Contains(out, `action="https://`) {
		t.Fatalf("table action allowed an external path: %s", out)
	}
	bad := Table(TableProps{Rows: [][]string{{"row"}}, RowActions: [][]TableAction{{{Label: "Remove", Action: "//example.test"}}}})
	if err := bad.Render(context.Background(), &bytes.Buffer{}); err == nil {
		t.Fatal("external row action path accepted")
	}
}

func TestSidebarEscapesModuleMetadataAndMarksActiveItem(t *testing.T) {
	out := render(t, Sidebar([]SidebarItem{{Route: "/admin/news", Label: `<News>`, Icon: `book"`, Group: "Content", Active: true}}))
	if !strings.Contains(out, `href="/admin/news"`) || !strings.Contains(out, `is-active`) || !strings.Contains(out, `&lt;News&gt;`) || strings.Contains(out, `<News>`) || strings.Contains(out, `book"`) {
		t.Fatalf("sidebar output was not safely rendered: %s", out)
	}
}

func TestBlockEditorUsesEditionAssetAndAllowlistedLivePreview(t *testing.T) {
	out := render(t, BlockEditor(BlockEditorProps{ID: "body", Name: "body", Label: "Body", Value: `{"blocks":[]}`, PreviewURL: "/api/preview"}))
	for _, marker := range []string{`data-editorjs-mount="body"`, `data-editorjs-script="/vendor/editorjs/editorjs.umd.js"`, `hx-post="/api/preview"`, `hx-target="#body-preview"`, `hx-swap="innerHTML"`, `hx-trigger="keyup delay:300ms"`} {
		if !strings.Contains(out, marker) {
			t.Fatalf("block editor missing %q: %s", marker, out)
		}
	}
	if strings.Contains(out, "<!--") || strings.Contains(out, "<script") {
		t.Fatalf("block editor emitted unsafe/comment markup: %s", out)
	}
}

func TestBlockEditorDoesNotInventPreviewRoute(t *testing.T) {
	out := render(t, BlockEditor(BlockEditorProps{ID: "body", Name: "body", Label: "Body", Value: `{"blocks":[]}`}))
	if strings.Contains(out, "hx-post") || strings.Contains(out, "/api/preview") {
		t.Fatalf("block editor invented an Edition preview endpoint: %s", out)
	}
}

func TestMediaPickerEscapesReferenceAndKeepsChooserNonAuthorizing(t *testing.T) {
	out := render(t, MediaPicker(MediaPickerProps{ID: "cover", Name: "cover", Label: `<Cover>`, Value: `a"b`}))
	if !strings.Contains(out, `type="hidden"`) || !strings.Contains(out, `value="a&#34;b"`) || !strings.Contains(out, `/admin/media?select=cover`) {
		t.Fatalf("media picker output missing contract: %s", out)
	}
	if strings.Contains(out, "<Cover>") || strings.Contains(out, "javascript:") {
		t.Fatalf("media picker output was not safely rendered: %s", out)
	}
}
