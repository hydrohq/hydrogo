package main

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

// assetRoutes registers the CSRF token endpoint browser clients read before
// their first mutation.
func (a *serveApp) assetRoutes() ([]router.Route, error) {
	return []router.Route{
		{Pattern: "GET /csrf-token", Access: router.Public, Handler: http.HandlerFunc(security.TokenHandler)},
		{Pattern: "GET /api/v1/system/domain-check", Access: router.Public, Handler: http.HandlerFunc(a.systemDomainCheckHandler)},
		{Pattern: "GET /api/v1/internal/check-domain", Access: router.Public, Handler: http.HandlerFunc(a.systemDomainCheckHandler)},
	}, nil
}
