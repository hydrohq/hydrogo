package v1

import (
	"context"

	"github.com/a-h/templ"
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
