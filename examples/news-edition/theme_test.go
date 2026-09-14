package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/extension"
)

func TestEditorialThemeEscapesChromeAndRendersSanitizedContent(t *testing.T) {
	data := extension.PageViewData{Locale: "tr", Title: "<Başlık>", HTMLContent: `<p data-safe="1">İçerik</p>`}
	var out bytes.Buffer
	if err := (EditorialTheme{}).RenderPage(context.Background(), data).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "&lt;Başlık&gt;") || !strings.Contains(got, data.HTMLContent) {
		t.Fatalf("output did not preserve escaped chrome and sanitized HTML: %s", got)
	}
}
