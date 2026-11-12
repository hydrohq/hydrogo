//go:build !hydrocloud

package main

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/router"
)

// The hosted multi-tenant platform (HydroCloud: orchestrator, public onboarding, billing,
// superadmin console) is not part of this repository. Builds without the `hydrocloud` tag get
// these stand-ins: the routes answer 404 and the platform features report themselves as off.
// The real implementations live in the private HydroCloud overlay, built with `-tags hydrocloud`.

func (a *serveApp) superadminRoutes() []router.Route { return nil }
func (a *serveApp) billingRoutes() []router.Route    { return nil }

func (a *serveApp) isOrchestratorRequest(*http.Request) bool { return false }
func isOnboardEnabled() bool                                 { return false }

func (a *serveApp) adminTenantUsageHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
func (a *serveApp) adminAIGenerateSiteHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
func (a *serveApp) adminOrchestratorListHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
func (a *serveApp) adminOrchestratorProvisionHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
func (a *serveApp) adminOrchestratorGetHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
func (a *serveApp) adminOrchestratorUpdateHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
func (a *serveApp) adminOrchestratorImpersonateHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
func (a *serveApp) adminOrchestratorArchiveHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
func (a *serveApp) systemDomainCheckHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
func (a *serveApp) internalAIProxyHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
