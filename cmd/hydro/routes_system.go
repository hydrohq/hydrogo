package main

import (
	"fmt"
	"io/fs"
	"net/http"

	formsruntime "github.com/ardaozyildirim/hydrogo/internal/forms"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	staticassets "github.com/ardaozyildirim/hydrogo/static"
)

// assetRoutes registers embedded and theme static assets, the CSRF token
// endpoint and fonts for the modules that need them.
func (a *serveApp) assetRoutes() ([]router.Route, error) {
	cfg, modules, selectedTheme := a.cfg, a.modules, a.selectedTheme
	var routes []router.Route
	if modules["pages"] || modules["forms"] || modules["news"] {
		routes = append(routes, router.Route{Pattern: "GET /designsystem/static/{path...}", Access: router.Public, Handler: http.HandlerFunc(designsystem.ServeAsset)})
	}
	if modules["pages"] || modules["news"] {
		routes = append(routes, router.Route{Pattern: "GET /csrf-token", Access: router.Public, Handler: http.HandlerFunc(security.TokenHandler)})
	}
	if modules["pages"] {
		fontsDir := cfg.SEO.FontsDir
		if fontsDir == "" {
			fontsDir = "public/fonts"
		}
		routes = append(routes, router.Route{Pattern: "GET /fonts/{path...}", Access: router.Public, Handler: designsystem.FontAssets(fontsDir)})
	}
	if modules["forms"] {
		routes = append(routes, router.Route{Pattern: "GET /forms/static/{path...}", Access: router.Public, Handler: http.HandlerFunc(formsruntime.ServeAsset)})
	}
	if selectedTheme != nil {
		routes = append(routes, router.Route{Pattern: "GET /theme/static/{path...}", Access: router.Public, Handler: http.HandlerFunc(selectedTheme.ServeAsset)})
	}
	// Local editor assets are embedded in the release binary. Uploaded media and
	// themes remain external persistent resources by design.
	vendorFS, err := fs.Sub(staticassets.FS, "vendor")
	if err != nil {
		return nil, fmt.Errorf("prepare embedded vendor assets: %w", err)
	}
	routes = append(routes, router.Route{Pattern: "GET /vendor/{path...}", Access: router.Public, Handler: http.StripPrefix("/vendor/", http.FileServer(http.FS(vendorFS)))}, adminAssetsRoute())
	return routes, nil
}
