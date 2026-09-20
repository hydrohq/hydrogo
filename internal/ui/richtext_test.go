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

func TestRichTextHTMLRejectsUnsafeLinkProtocols(t *testing.T) {
	got := RichTextHTML(`{"ops":[{"insert":"click","attributes":{"link":"javascript:alert(1)"}}]}`)
	if strings.Contains(got, "javascript:") || strings.Contains(got, "href=") {
		t.Fatalf("unsafe link rendered: %s", got)
	}
}

func TestRichTextHTMLEscapesEmbeddedMarkup(t *testing.T) {
	got := RichTextHTML(`{"ops":[{"insert":"<script>alert(1)</script>\n"}]}`)
	if strings.Contains(got, "<script>") || !strings.Contains(got, "&lt;script&gt;") {
		t.Fatalf("embedded markup was not escaped: %s", got)
	}
}

func TestRichTextHTMLRendersListsAndImages(t *testing.T) {
	got := RichTextHTML(`{"ops":[{"insert":"Madde 1"},{"attributes":{"list":"bullet"},"insert":"\n"},{"insert":{"image":"https://example.com/photo.jpg"}},{"insert":"\n"},{"insert":"Kod"},{"attributes":{"code":true},"insert":" ornegi"},{"insert":"\n"}]}`)
	if !strings.Contains(got, "<ul><li>Madde 1</li></ul>") {
		t.Fatalf("expected bullet list, got: %s", got)
	}
	if !strings.Contains(got, `<img src="https://example.com/photo.jpg"`) {
		t.Fatalf("expected image rendering, got: %s", got)
	}
	if !strings.Contains(got, "<code> ornegi</code>") {
		t.Fatalf("expected inline code rendering, got: %s", got)
	}
}

func TestRichTextHTMLRendersInlineSizeFontAndSafeColor(t *testing.T) {
	got := RichTextHTML(`{"ops":[{"insert":"Large","attributes":{"size":"large","font":"serif","color":"#123456"}},{"insert":"\n"}]}`)
	for _, want := range []string{`font-size:1.5em`, `font-family:Georgia,Times New Roman,serif`, `color:#123456`} {
		if !strings.Contains(got, want) {
			t.Fatalf("inline formatting missing %q in %s", want, got)
		}
	}
}

func TestRichTextHTMLRejectsUnsafeInlineColor(t *testing.T) {
	got := RichTextHTML(`{"ops":[{"insert":"Unsafe","attributes":{"color":"red; background:url(javascript:alert(1))"}},{"insert":"\n"}]}`)
	if strings.Contains(got, "javascript") || strings.Contains(got, "background") {
		t.Fatalf("unsafe inline style rendered: %s", got)
	}
}

func TestRichTextHTMLRejectsProtocolRelativeAndDataSVG(t *testing.T) {
	gotLink := RichTextHTML(`{"ops":[{"insert":"click","attributes":{"link":"//evil.com"}}]}`)
	if strings.Contains(gotLink, "evil.com") && strings.Contains(gotLink, "<a href=") {
		t.Fatalf("protocol-relative link should not be rendered: %s", gotLink)
	}

	gotImg := RichTextHTML(`{"ops":[{"insert":{"image":"data:image/svg+xml;utf8,<svg onload=alert(1)>"}}]}`)
	if strings.Contains(gotImg, "<img") || strings.Contains(gotImg, "onload") {
		t.Fatalf("unsafe SVG data URL rendered: %s", gotImg)
	}
}
