package main

import (
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/adminui"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/webhooks"
	newstransport "github.com/ardaozyildirim/hydrogo/modules/news/transport"
	pagetransport "github.com/ardaozyildirim/hydrogo/modules/pages/transport"
	sitetransport "github.com/ardaozyildirim/hydrogo/modules/site/transport"
)

// routes assembles the full route table. The order is the registration order
// the server sees; keep new groups next to their neighbours.
func (a *serveApp) routes() ([]router.Route, error) {
	routes := a.authRoutes()
	collectionRoutes, err := a.collectionRoutes()
	if err != nil {
		return nil, err
	}
	routes = append(routes, collectionRoutes...)
	routes = append(routes, a.pushRoutes()...)
	routes = append(routes, a.trashRoutes()...)
	routes = append(routes, a.searchRoutes()...)
	routes = append(routes, a.mobileAuthHTTP.RouterRoutes()...)
	routes = append(routes, a.pageRoutes()...)
	routes = append(routes, a.memberRoutes()...)
	routes = append(routes, a.analyticsRoutes()...)
	routes = append(routes, a.adminStudioRoutes()...)
	routes = append(routes, a.adminRoutes()...)
	routes = append(routes, a.contentTypeAdminRoutes()...)
	routes = append(routes, a.mediaRoutes()...)
	routes = append(routes, a.siteRoutes()...)
	routes = append(routes, a.newsRoutes()...)
	routes = append(routes, adminui.Routes()...)
	assetRoutes, err := a.assetRoutes()
	if err != nil {
		return nil, err
	}
	routes = append(routes, assetRoutes...)
	routes = append(routes, a.webhookRoutes()...)
	return routes, nil
}

// webhookRoutes collects the configured webhook endpoints (none in safe
// mode) for the outbox worker; it registers no routes.
func (a *serveApp) webhookRoutes() []router.Route {
	cfg := a.cfg
	webhookEndpoints := make([]webhooks.Endpoint, 0, len(cfg.Webhooks))
	if !cfg.SafeMode {
		for _, configured := range cfg.Webhooks {
			topics := make(map[string]struct{}, len(configured.Topics))
			for _, topic := range configured.Topics {
				topics[strings.TrimSpace(topic)] = struct{}{}
			}
			webhookEndpoints = append(webhookEndpoints, webhooks.Endpoint{ID: configured.ID, URL: configured.URL, Secret: configured.Secret, Topics: topics})
		}
	}
	a.webhookEndpoints = webhookEndpoints
	return nil
}

func pageRouterRoutes(routes []pagetransport.Route) []router.Route {
	result := make([]router.Route, 0, len(routes))
	for _, route := range routes {
		access := router.Protected
		if !route.Protected {
			access = router.Public
		}
		result = append(result, router.Route{Pattern: route.Pattern, Access: access, Handler: route.Handler})
	}
	return result
}

func mediaRouterRoutes(routes []media.HTTPRoute) []router.Route {
	result := make([]router.Route, 0, len(routes))
	for _, route := range routes {
		access := router.Protected
		if !route.Protected {
			access = router.Public
		}
		result = append(result, router.Route{Pattern: route.Pattern, Access: access, Handler: route.Handler})
	}
	return result
}

func siteRouterRoutes(routes []sitetransport.Route) []router.Route {
	result := make([]router.Route, 0, len(routes))
	for _, route := range routes {
		access := router.Protected
		if route.Public {
			access = router.Public
		}
		result = append(result, router.Route{Pattern: route.Pattern, Access: access, Handler: route.Handler})
	}
	return result
}

func newsRouterRoutes(routes []newstransport.Route) []router.Route {
	result := make([]router.Route, 0, len(routes))
	for _, route := range routes {
		access := router.Protected
		if !route.Protected {
			access = router.Public
		}
		result = append(result, router.Route{Pattern: route.Pattern, Access: access, Handler: route.Handler})
	}
	return result
}
