package main

import (
	"context"
	"fmt"
	"html/template"
	"io"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/extension"
)

// EditorialTheme is a second external theme using only typed extension slots.
type EditorialTheme struct{}

func (EditorialTheme) Name() string { return "editorial" }

func (EditorialTheme) RenderHeader(_ context.Context, data extension.HeaderData) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<header><nav><strong>%s</strong></nav></header>`, template.HTMLEscapeString(data.SiteName))
		return err
	})
}

func (EditorialTheme) RenderFooter(_ context.Context, data extension.FooterData) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<footer><small>%s</small></footer>`, template.HTMLEscapeString(data.Copyright))
		return err
	})
}

func (EditorialTheme) RenderPage(_ context.Context, data extension.PageViewData) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<article lang="%s"><h1>%s</h1><div>%s</div></article>`, template.HTMLEscapeString(data.Locale), template.HTMLEscapeString(data.Title), template.HTML(data.HTMLContent))
		return err
	})
}
