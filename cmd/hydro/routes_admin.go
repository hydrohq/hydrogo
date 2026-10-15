package main

import (
	"encoding/json"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	adminsearch "github.com/ardaozyildirim/hydrogo/internal/search"
)

// adminStudioRoutes registers the admin search API and the studio manifest.
func (a *serveApp) adminStudioRoutes() []router.Route {
	cfg, db, modules := a.cfg, a.db, a.modules
	var routes []router.Route
	if modules["pages"] || modules["news"] || modules["members"] || modules["media"] || modules["site"] {
		routes = append(routes, router.Route{Pattern: "GET /admin/api/search", Access: router.Protected, Handler: adminsearch.AdminSearchHandler{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope, Modules: modules}})
	}
	routes = append(routes, router.Route{Pattern: "GET /admin/studio/manifest", Access: router.Protected, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if !activePlatformAdmin(user) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(adminstudio.E2EManifestForCollections(a.activeStudioCollections()))
	})})
	return routes
}

// adminRoutes registers the report export.
func (a *serveApp) adminRoutes() []router.Route {
	return []router.Route{{Pattern: "GET /admin/reports/export", Access: router.Protected, Handler: http.HandlerFunc(a.adminReportsExport)}}
}

func activePlatformAdmin(user *auth.User) bool {
	return user != nil && user.Active && user.Role == auth.RoleAdmin
}
