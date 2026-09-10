package ui

import (
	"strings"
	"testing"
)

func TestRichTextHTMLRendersDeltaAndEscapesMarkup(t *testing.T) {
	got := RichTextHTML(`{"ops":[{"insert":"Hello","attributes":{"bold":true}},{"insert":" world\n"}]}`)
	if !strings.Contains(got, "<strong>Hello</strong>") || !strings.Contains(got, "<p>") {
		t.Fatalf("rendered rich text = %s", got)
	}
	if strings.Contains(got, "<script") {
		t.Fatal("rich text renderer emitted an unsafe tag")
	}
}

func TestRichTextHTMLKeepsLegacyPlainTextReadable(t *testing.T) {
	got := RichTextHTML("First line\nSecond line")
	if got != "<p>First line</p><p>Second line</p>" {
		t.Fatalf("legacy render = %s", got)
	}
}
