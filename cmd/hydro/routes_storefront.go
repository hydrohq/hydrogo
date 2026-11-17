package main

import (
	"fmt"
	"net/http"

	"github.com/hydrohq/hydrogo/internal/router"
	"github.com/hydrohq/hydrogo/internal/storefront"
)

func (a *serveApp) storefrontRoutes() []router.Route {
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
	fmt.Fprintf(w, "User-agent: *\nAllow: /\n\nSitemap: /sitemap.xml\n")
}
