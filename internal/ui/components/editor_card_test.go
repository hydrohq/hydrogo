package components

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestEditorCardEscapesTitleAndWrapsChildren(t *testing.T) {
	child := templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "child-body")
		return err
	})
	var out strings.Builder
	ctx := templ.WithChildren(context.Background(), child)
	if err := EditorCard(EditorCardProps{Title: "SEO & <Arama>", Tag: "YENİ", Status: "draft"}).Render(ctx, &out); err != nil {
		t.Fatal(err)
	}
	markup := out.String()
	for _, want := range []string{`<section class="editor-card">`, `role="heading" aria-level="2">SEO &amp; &lt;Arama&gt;</span>`, `<span class="card-tag">YENİ</span>`, `<span class="status">draft</span>`, "child-body"} {
		if !strings.Contains(markup, want) {
			t.Errorf("editor card missing %q: %s", want, markup)
		}
	}
	out.Reset()
	if err := EditorCard(EditorCardProps{Title: "Plain"}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "card-tag") || strings.Contains(out.String(), `class="status"`) {
		t.Errorf("optional badges rendered without values: %s", out.String())
	}
}
