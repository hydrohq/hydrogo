package main

import (
	"time"

	pushservice "github.com/ardaozyildirim/hydrogo/internal/push"
	"github.com/ardaozyildirim/hydrogo/internal/router"
)

// authRoutes registers the admin auth API.
func (a *serveApp) authRoutes() []router.Route {
	return append([]router.Route{}, a.authHTTP.RouterRoutes()...)
}

func (a *serveApp) pushRoutes() []router.Route {
	// Safe mode leaves the dispatcher unset even when the module is enabled.
	if !a.modules["push"] || a.pushDispatcher == nil {
		return nil
	}
	pushDispatcher, newRateLimiter := a.pushDispatcher, a.newRateLimiter
	return (pushservice.Handler{Dispatcher: *pushDispatcher, Limiter: newRateLimiter(3, time.Minute), Audit: a.auditRecorder}).Routes()
}

func (a *serveApp) memberRoutes() []router.Route {
	if !enabledModuleSet(a.cfg.Modules)["members"] {
		return nil
	}
	routes := a.memberHTTP.Routes()
	if a.memberOAuthHTTP != nil {
		routes = append(routes, a.memberOAuthHTTP.Routes()...)
	}
	return routes
}
