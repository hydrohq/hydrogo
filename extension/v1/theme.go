package v1

import (
	"context"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

type PageViewData struct {
	Title       string
	Slug        string
	HTMLContent string
	Nonce       string
	Locale      string
}

type HeaderData struct {
	SiteName string
	Nonce    string
}

type FooterData struct {
	Copyright string
	Nonce     string
}

type Theme interface {
	Name() string
	RenderPage(context.Context, PageViewData) templ.Component
	RenderHeader(context.Context, HeaderData) templ.Component
	RenderFooter(context.Context, FooterData) templ.Component
}

// BlockTheme is an optional extension of Theme. Keeping block rendering
// separate preserves compatibility with existing themes while allowing a
// selected edition to opt into structured page blocks.
type BlockTheme interface {
	Theme
	RenderBlock(context.Context, pages.Block) (templ.Component, error)
}
