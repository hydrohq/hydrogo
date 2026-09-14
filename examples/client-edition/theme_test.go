package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/extension"
)

func TestExampleThemesEscapeChromeAndRenderSanitizedContent(t *testing.T) {
	data := extension.PageViewData{Locale: "tr", Title: "<Başlık>", HTMLContent: `<p data-safe="1">İçerik</p>`}
	for _, theme := range []extension.Theme{DefaultTheme{}, MinimalTheme{}} {
		var out bytes.Buffer
		if err := theme.RenderPage(context.Background(), data).Render(context.Background(), &out); err != nil {
			t.Fatalf("%s render: %v", theme.Name(), err)
		}
		got := out.String()
		if !strings.Contains(got, "&lt;Başlık&gt;") || !strings.Contains(got, data.HTMLContent) {
			t.Fatalf("%s output did not preserve escaped chrome and sanitized HTML: %s", theme.Name(), got)
		}
	}
}
