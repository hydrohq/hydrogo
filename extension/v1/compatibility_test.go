package v1

import (
	"context"
	"net/http"
	"testing"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/extension"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

// These assertions intentionally fail at compile time when a public contract
// changes. They are fixtures rather than runtime registration, so they add no
// routes, workers, or other module overhead.
type compatibilityModule struct{}

func (compatibilityModule) Name() string { return "compatibility" }
func (compatibilityModule) Routes() []ModuleRoute {
	return []ModuleRoute{{Pattern: "GET /health", Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}}
}

type compatibilityTheme struct{}

func (compatibilityTheme) Name() string { return "compatibility" }
func (compatibilityTheme) RenderPage(context.Context, PageViewData) templ.Component {
	return nil
}
func (compatibilityTheme) RenderHeader(context.Context, HeaderData) templ.Component {
	return nil
}
func (compatibilityTheme) RenderFooter(context.Context, FooterData) templ.Component {
	return nil
}

type compatibilityBlockTheme struct{ compatibilityTheme }

func (compatibilityBlockTheme) RenderBlock(context.Context, pages.Block) (templ.Component, error) {
	return nil, nil
}

type legacyCompatibilityTheme struct{}

func (legacyCompatibilityTheme) Name() string { return "legacy-compatibility" }
func (legacyCompatibilityTheme) RenderPage(context.Context, extension.PageViewData) templ.Component {
	return nil
}
func (legacyCompatibilityTheme) RenderHeader(context.Context, extension.HeaderData) templ.Component {
	return nil
}
func (legacyCompatibilityTheme) RenderFooter(context.Context, extension.FooterData) templ.Component {
	return nil
}

var (
	_ Module           = compatibilityModule{}
	_ extension.Module = compatibilityModule{}
	_ Theme            = compatibilityTheme{}
	_ BlockTheme       = compatibilityBlockTheme{}
	_ extension.Theme  = legacyCompatibilityTheme{}
)

func TestPublicContractsRemainCompileTimeCompatible(t *testing.T) {
	// Keep a runtime test so this file is visible in ordinary package test runs.
	if (ModuleRoute{}).Public {
		t.Fatal("zero-value route unexpectedly public")
	}
}
