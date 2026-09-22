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

func TestFormAndInputRenderCSRFAndEscapedValues(t *testing.T) {
	out := render(t, Form(FormProps{Action: `/projects/<x>`, CSRF: `a"b`, Body: Input(InputProps{ID: "title", Name: "title", Label: "Title", Required: true})}))
	if !strings.Contains(out, `name="_csrf" value="a&#34;b"`) || !strings.Contains(out, `action="/projects/&lt;x&gt;"`) || !strings.Contains(out, `required="required"`) {
		t.Fatalf("unexpected form output: %s", out)
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

func TestSidebarEscapesModuleMetadataAndMarksActiveItem(t *testing.T) {
	out := render(t, Sidebar([]SidebarItem{{Route: "/admin/news", Label: `<News>`, Icon: `book"`, Group: "Content", Active: true}}))
	if !strings.Contains(out, `href="/admin/news"`) || !strings.Contains(out, `is-active`) || !strings.Contains(out, `&lt;News&gt;`) || strings.Contains(out, `<News>`) || strings.Contains(out, `book"`) {
		t.Fatalf("sidebar output was not safely rendered: %s", out)
	}
}

func TestBlockEditorUsesEditionAssetAndAllowlistedLivePreview(t *testing.T) {
	out := render(t, BlockEditor(BlockEditorProps{ID: "body", Name: "body", Label: "Body", Nonce: "n\"1", Value: `{"blocks":[]}`, PreviewURL: "/api/preview"}))
	for _, marker := range []string{`data-editorjs-mount="body"`, `hx-post="/api/preview"`, `hx-target="#body-preview"`, `hx-swap="innerHTML"`, `hx-trigger="keyup delay:300ms"`, `/vendor/editorjs/editorjs.umd.js`} {
		if !strings.Contains(out, marker) {
			t.Fatalf("block editor missing %q: %s", marker, out)
		}
	}
	if strings.Contains(out, "<!--") || strings.Contains(out, "n\"1") {
		t.Fatalf("block editor emitted unsafe/comment markup: %s", out)
	}
}
