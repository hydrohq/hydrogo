package extension

import (
	"context"

	"github.com/a-h/templ"
)

// PageViewData holds structured public page presentation data for themes.
type PageViewData struct {
	Title       string
	Slug        string
	HTMLContent string
	Nonce       string
	Locale      string
}

// HeaderData holds site-level header data.
type HeaderData struct {
	SiteName string
	Nonce    string
}

// FooterData holds site-level footer data.
type FooterData struct {
	Copyright string
	Nonce     string
}

// Theme defines the contract for client-edition UI customization without Core modification.
//
// Deprecated: use extension/v1.Theme for new integrations.
type Theme interface {
	Name() string
	RenderPage(ctx context.Context, data PageViewData) templ.Component
	RenderHeader(ctx context.Context, data HeaderData) templ.Component
	RenderFooter(ctx context.Context, data FooterData) templ.Component
}
