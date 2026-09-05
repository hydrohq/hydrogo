package main

import (
	"context"
	"fmt"
	"io"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/extension"
)

type MinimalTheme struct{}

func (t MinimalTheme) Name() string {
	return "minimal"
}

func (t MinimalTheme) RenderHeader(ctx context.Context, data extension.HeaderData) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<header class="min-header"><span>%s</span></header>`, data.SiteName)
		return err
	})
}

func (t MinimalTheme) RenderFooter(ctx context.Context, data extension.FooterData) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<footer class="min-footer"><small>%s</small></footer>`, data.Copyright)
		return err
	})
}

func (t MinimalTheme) RenderPage(ctx context.Context, data extension.PageViewData) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="%s">
<head>
    <meta charset="utf-8">
    <title>%s - Minimal</title>
    <style nonce="%s">
        body { font-family: monospace; padding: 1rem; color: #111; }
    </style>
</head>
<body>
    <article>
        <h2>%s</h2>
        <div>%s</div>
    </article>
</body>
</html>`, data.Locale, data.Title, data.Nonce, data.Title, data.HTMLContent)
		return err
	})
}
