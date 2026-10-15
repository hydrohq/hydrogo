package main

import (
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	adminsearch "github.com/ardaozyildirim/hydrogo/internal/search"
)

// adminStudioRoutes registers the admin search API and the admin schema.
func (a *serveApp) adminStudioRoutes() []router.Route {
	cfg, db, modules := a.cfg, a.db, a.modules
	var routes []router.Route
	if modules["pages"] || modules["news"] || modules["members"] || modules["media"] || modules["site"] {
		routes = append(routes, router.Route{Pattern: "GET /api/v1/admin/search", Access: router.Protected, Handler: adminsearch.AdminSearchHandler{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope, Modules: modules}})
	}
	routes = append(routes, router.Route{Pattern: "GET /api/v1/admin/schema", Access: router.Protected, Handler: http.HandlerFunc(a.adminSchema)})
	return routes
}

// adminRoutes registers the cross-cutting admin API: dashboard, reports,
// audit explorer, backups, IP bans, users and permissions, account
// preferences, system information and the admin i18n catalog.
func (a *serveApp) adminRoutes() []router.Route {
	protected := func(pattern string, handler http.HandlerFunc) router.Route {
		return router.Route{Pattern: pattern, Access: router.Protected, Handler: handler}
	}
	routes := []router.Route{
		protected("GET /api/v1/admin/dashboard", a.adminDashboardHandler),
		protected("GET /api/v1/admin/reports", a.adminReportsHandler),
		protected("GET /api/v1/admin/backups", a.adminBackupStatusHandler),
		protected("POST /api/v1/admin/backups", a.adminBackupDownloadHandler(a.newRateLimiter(3, time.Minute))),
		protected("GET /api/v1/admin/security/ip-bans", a.adminIPBanListHandler),
		protected("POST /api/v1/admin/security/ip-bans", a.adminIPBanMutationHandler(false)),
		protected("DELETE /api/v1/admin/security/ip-bans", a.adminIPBanMutationHandler(true)),
		protected("GET /api/v1/admin/users", a.adminUsersListHandler),
		protected("GET /api/v1/admin/users/{id}", a.adminUserGetHandler),
		protected("GET /api/v1/admin/permissions", a.adminPermissionsHandler),
		protected("PUT /api/v1/admin/account/locale", a.adminAccountLocaleHandler),
		protected("GET /api/v1/admin/i18n/{locale}", a.adminI18nHandler),
		protected("GET /api/v1/admin/system/info", a.adminSystemInfoHandler),
		protected("GET /api/v1/admin/system/metrics", a.adminSystemMetricsHandler),
		protected("GET /api/v1/admin/system/health", a.adminSystemHealthHandler(a.readiness)),
		protected("POST /api/v1/admin/client-errors", a.adminClientErrorHandler(a.newRateLimiter(10, time.Minute))),
	}
	if a.modules["pages"] {
		routes = append(routes, protected("GET /api/v1/admin/audit", a.adminAuditHandler))
	}
	return routes
}

func activePlatformAdmin(user *auth.User) bool {
	return user != nil && user.Active && user.Role == auth.RoleAdmin
}

// adminSchema describes what the admin can show: every collection with its
// fields and API path, the content types behind them, the enabled modules,
// the site locales and the permission catalog.
type adminSchema struct {
	Version      int                      `json:"version"`
	Resources    []adminstudio.Collection `json:"resources"`
	ContentTypes []adminSchemaContentType `json:"content_types"`
	Modules      map[string]bool          `json:"modules"`
	Locales      adminSchemaLocales       `json:"locales"`
	Permissions  []string                 `json:"permissions"`
}

type adminSchemaContentType struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Description  string `json:"description"`
	Public       bool   `json:"public"`
	Drafts       bool   `json:"drafts"`
	Revisions    bool   `json:"revisions"`
	Localization bool   `json:"localization"`
	TitleField   string `json:"title_field,omitempty"`
}

type adminSchemaLocales struct {
	Enabled bool     `json:"enabled"`
	Default string   `json:"default"`
	All     []string `json:"all"`
}

func (a *serveApp) adminSchema(w http.ResponseWriter, r *http.Request) {
	if user := auth.UserFromContext(r.Context()); user == nil || !user.Active {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusOK, a.buildAdminSchema(), nil)
}

func (a *serveApp) buildAdminSchema() adminSchema {
	localized := a.contentLocales.Enabled()
	resources := contenttypes.AdminCollections(a.contentTypes, localized)
	resources = append(resources, a.activeStudioCollections()...)
	schema := adminSchema{
		Version:      1,
		Resources:    resources,
		ContentTypes: make([]adminSchemaContentType, 0, len(a.contentTypes)),
		Modules:      make(map[string]bool, len(a.modules)),
		Locales:      adminSchemaLocales{Enabled: localized, Default: a.contentLocales.Default, All: append([]string{}, a.contentLocales.All...)},
		Permissions:  a.authSvc.AvailablePermissions(),
	}
	for _, definition := range a.contentTypes {
		schema.ContentTypes = append(schema.ContentTypes, adminSchemaContentType{
			ID: definition.ID, Label: definition.Label, Description: definition.Description, Public: definition.Public,
			Drafts: definition.Drafts, Revisions: definition.Revisions, Localization: localized && definition.Localized(),
			TitleField: definition.TitleField,
		})
	}
	for module, enabled := range a.modules {
		if enabled {
			schema.Modules[module] = true
		}
	}
	return schema
}
