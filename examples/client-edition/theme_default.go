package main

import (
	"context"
	"fmt"
	"html/template"
	"io"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/extension"
)

type DefaultTheme struct{}

func (t DefaultTheme) Name() string {
	return "default"
}

func (t DefaultTheme) RenderHeader(ctx context.Context, data extension.HeaderData) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<header class="theme-default-header"><nav><h1>%s</h1></nav></header>`, template.HTMLEscapeString(data.SiteName))
		return err
	})
}

func (t DefaultTheme) RenderFooter(ctx context.Context, data extension.FooterData) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<footer class="theme-default-footer"><p>%s</p></footer>`, template.HTMLEscapeString(data.Copyright))
		return err
	})
}

func (t DefaultTheme) RenderPage(ctx context.Context, data extension.PageViewData) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="%s">
<head>
    <meta charset="utf-8">
    <title>%s</title>
    <style nonce="%s">
        body { font-family: sans-serif; margin: 0; padding: 2rem; background: #fafafa; }
        .page-content { max-width: 800px; margin: 0 auto; background: #fff; padding: 2rem; border-radius: 8px; }
    </style>
</head>
<body>
    <div class="page-content">
        <h1>%s</h1>
        <div class="content-body">%s</div>
    </div>
</body>
</html>`, template.HTMLEscapeString(data.Locale), template.HTMLEscapeString(data.Title), template.HTMLEscapeString(data.Nonce), template.HTMLEscapeString(data.Title), template.HTML(data.HTMLContent))
		return err
	})
}
