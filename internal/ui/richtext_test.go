package ui

import (
	"strings"
	"testing"
)

func TestPrepareArticleHTMLBuildsAccessibleNestedTOC(t *testing.T) {
	article, toc, err := PrepareArticleHTML(`<h2>Overview</h2><p>Read</p><h3>Details &amp; care</h3><h2>Overview</h2>`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`id="overview"`, `id="details-care"`, `id="overview-2"`} {
		if !strings.Contains(article, want) {
			t.Errorf("article missing %q: %s", want, article)
		}
	}
	for _, want := range []string{`aria-label="İçindekiler"`, `href="#overview"`, `href="#details-care"`, `Details &amp; care`} {
		if !strings.Contains(toc, want) {
			t.Errorf("table of contents missing %q: %s", want, toc)
		}
	}
}

func TestPrepareArticleHTMLWithoutHeadingsHasNoTOC(t *testing.T) {
	article, toc, err := PrepareArticleHTML(`<p>Plain article</p>`)
	if err != nil || toc != "" || !strings.Contains(article, "Plain article") {
		t.Fatalf("PrepareArticleHTML = (%q, %q, %v)", article, toc, err)
	}
}

func TestPublicArticleToolsAreAccessibleAndExposeNoRuntimeDependency(t *testing.T) {
	var output strings.Builder
	if err := PublicArticleTools().Render(t.Context(), &output); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`role="group"`, `aria-label="Okuma araçları"`, `data-font-size="up"`, `data-font-size="down"`, `data-print-article`} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("reading tools missing %q: %s", want, output.String())
		}
	}
	if strings.Contains(PublicArticleJS, "https://") || strings.Contains(PublicArticleCSS, "javascript:") {
		t.Fatal("reading tools include a remote runtime dependency")
	}
}

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

func TestRichTextEmbedsAllowlistedVideosWithoutScripts(t *testing.T) {
	for _,input:=range []string{
		`{"ops":[{"insert":"https://youtu.be/abc123\n"}]}`,
		`{"ops":[{"insert":{"video":"https://vimeo.com/12345"}}]}`,
	}{got:=RichTextHTML(input);if !strings.Contains(got,"<iframe")||!strings.Contains(got,"loading=\"lazy\"")||strings.Contains(got,"<script"){t.Fatalf("video output = %s",got)}}
	bad:=RichTextHTML(`{"ops":[{"insert":{"video":"https://evil.test/watch/123"}}]}`);if strings.Contains(bad,"<iframe"){t.Fatalf("untrusted video rendered: %s",bad)}
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
