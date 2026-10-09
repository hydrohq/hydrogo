package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRichTextEditorRendersAccessibleQuillField(t *testing.T) {
	var output bytes.Buffer
	err := RichTextEditor(RichTextEditorProps{
		ID: "body", Name: "article_body", Label: "Article body", Value: `{"ops":[{"insert":"Hello"}]}`,
	}).Render(context.Background(), &output)
	if err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{
		`class="richtext-field" data-richtext`,
		`id="body-editor" class="richtext-editor ql-container ql-snow" role="textbox" aria-labelledby="body-label" aria-multiline="true"`,
		`class="richtext-value" name="article_body" hidden aria-hidden="true"`,
		`Article body araçları`,
		`{&#34;ops&#34;:[{&#34;insert&#34;:&#34;Hello&#34;}]}`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rich text control missing %q in %s", want, got)
		}
	}
}

func TestRichTextEditorAcceptsLocalizedToolbarLabels(t *testing.T) {
	got := render(t, RichTextEditor(RichTextEditorProps{
		ID: "body", Name: "body", Label: "Body", ToolbarLabel: "Rich text tools", BoldLabel: "Bold",
		ItalicLabel: "Italic", UnderlineLabel: "Underline", LinkLabel: "Add link", ImageLabel: "Add image",
		BlockQuoteLabel: "Block quote", OrderedListLabel: "Numbered list", BulletListLabel: "Bulleted list",
	}))
	for _, want := range []string{`aria-label="Rich text tools"`, `aria-label="Bold"`, `aria-label="Italic"`, `aria-label="Underline"`, `aria-label="Add link"`, `aria-label="Add image"`, `aria-label="Block quote"`, `aria-label="Numbered list"`, `aria-label="Bulleted list"`} {
		if !strings.Contains(got, want) {
			t.Errorf("rich text toolbar missing localized label %q: %s", want, got)
		}
	}
	if strings.Contains(got, "Kalın") || strings.Contains(got, "İtalik") {
		t.Errorf("rich text toolbar retained default Turkish copy: %s", got)
	}
}
