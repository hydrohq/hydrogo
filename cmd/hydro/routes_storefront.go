package main

import (
	"fmt"
	"net/http"

	"github.com/hydrohq/hydrogo/internal/router"
	"github.com/hydrohq/hydrogo/internal/storefront"
)

func (a *serveApp) storefrontRoutes() []router.Route {
	if a.externalStorefront != nil {
		routes := []router.Route{
			{Pattern: "/", Access: router.Public, Handler: a.externalStorefront},
		}
		return append(routes, a.cloudStorefrontAPIRoutes()...)
	}
	if cloudRoutes := a.cloudStorefrontRoutes(); len(cloudRoutes) > 0 {
		return cloudRoutes
	}
	return []router.Route{
		{Pattern: "GET /{$}", Access: router.Public, Handler: http.HandlerFunc(a.rootWelcomeHandler)},
		{Pattern: "GET /welcome", Access: router.Public, Handler: http.HandlerFunc(a.rootWelcomeHandler)},
		{Pattern: "GET /robots.txt", Access: router.Public, Handler: http.HandlerFunc(a.robotsHandler)},
	}
}

func (a *serveApp) rootWelcomeHandler(w http.ResponseWriter, r *http.Request) {
	storefront.RenderWelcome(w, r)
}

func (a *serveApp) robotsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /admin\n\nSitemap: /sitemap.xml\n")
}
