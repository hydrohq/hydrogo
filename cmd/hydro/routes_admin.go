package main

import (
	"context"
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	adminsearch "github.com/ardaozyildirim/hydrogo/internal/search"
	sitetransport "github.com/ardaozyildirim/hydrogo/modules/site/transport"
)

// adminStudioRoutes registers the admin search API and the admin schema.
func (a *serveApp) adminStudioRoutes() []router.Route {
	cfg, db, modules := a.cfg, a.db, a.modules
	var routes []router.Route
	if modules["pages"] || modules["news"] || modules["members"] || modules["media"] || modules["site"] {
		routes = append(routes, router.Route{Pattern: "GET /api/v1/admin/search", Access: router.Protected, Handler: adminsearch.AdminSearchHandler{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope, Modules: modules}})
	}
	routes = append(routes, router.Route{Pattern: "GET /api/v1/admin/schema", Access: router.Protected, Handler: http.HandlerFunc(a.adminSchema)})
	// The sign-in screen needs the project's name, logo and colour before anyone is signed in.
	routes = append(routes, router.Route{Pattern: "GET /api/v1/admin/branding", Access: router.Public, Handler: http.HandlerFunc(a.adminBrandingHandler)})
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
		protected("GET /api/v1/admin/tenant/usage", a.adminTenantUsageHandler),
		protected("GET /api/v1/admin/orchestrator/tenants", a.adminOrchestratorListHandler),
		protected("POST /api/v1/admin/orchestrator/tenants", a.adminOrchestratorProvisionHandler),
		protected("GET /api/v1/admin/orchestrator/tenants/{id}", a.adminOrchestratorGetHandler),
		protected("PATCH /api/v1/admin/orchestrator/tenants/{id}", a.adminOrchestratorUpdateHandler),
		protected("POST /api/v1/admin/orchestrator/tenants/{id}/impersonate", a.adminOrchestratorImpersonateHandler),
		protected("POST /api/v1/admin/orchestrator/tenants/{id}/archive", a.adminOrchestratorArchiveHandler),
		protected("POST /api/v1/admin/ai/copilot", a.rateLimitedAI("copilot", 20, time.Minute, a.adminAICopilotHandler)),
		protected("POST /api/v1/admin/ai/generate", a.rateLimitedAI("sitegen", 10, time.Minute, a.adminAIGenerateSiteHandler)),
		protected("GET /api/v1/admin/ai/actions", a.adminAIActionsListHandler),
		protected("POST /api/v1/admin/ai/actions/{id}/undo", a.adminAIActionUndoHandler),
		protected("GET /api/v1/admin/dashboard", a.adminDashboardHandler),
		protected("GET /api/v1/admin/reports", a.adminReportsHandler),
		protected("GET /api/v1/admin/backups", a.adminBackupStatusHandler),
		protected("POST /api/v1/admin/backups", a.adminBackupDownloadHandler(a.newRateLimiter(3, time.Minute))),
		protected("GET /api/v1/admin/security/ip-bans", a.adminIPBanListHandler),
		protected("POST /api/v1/admin/security/ip-bans", a.adminIPBanMutationHandler(false)),
		protected("DELETE /api/v1/admin/security/ip-bans", a.adminIPBanMutationHandler(true)),
		protected("GET /api/v1/admin/users", a.adminUsersListHandler),
		protected("GET /api/v1/admin/users/{id}", a.adminUserGetHandler),
		protected("GET /api/v1/admin/roles", a.adminRolesListHandler),
		protected("POST /api/v1/admin/roles", a.adminRoleSaveHandler),
		protected("GET /api/v1/admin/roles/{id}", a.adminRoleGetHandler),
		protected("PUT /api/v1/admin/roles/{id}", a.adminRoleSaveHandler),
		protected("DELETE /api/v1/admin/roles/{id}", a.adminRoleDeleteHandler),
		protected("GET /api/v1/admin/permissions", a.adminPermissionsHandler),
		protected("PUT /api/v1/admin/account/locale", a.adminAccountLocaleHandler),
		protected("GET /api/v1/admin/i18n/{locale}", a.adminI18nHandler),
		protected("GET /api/v1/admin/system/info", a.adminSystemInfoHandler),
		protected("GET /api/v1/admin/system/metrics", a.adminSystemMetricsHandler),
		protected("GET /api/v1/admin/system/health", a.adminSystemHealthHandler(a.readiness)),
		protected("POST /api/v1/admin/client-errors", a.adminClientErrorHandler(a.newRateLimiter(10, time.Minute))),
		// Jobs are core-owned (see docs/decisions/0010-generic-reports-and-jobs.md),
		// so this is always registered, independent of which modules are enabled.
		protected("GET /api/v1/admin/jobs", a.adminJobsHandler),
		protected("GET /api/v1/admin/jobs/dead-letters", a.adminJobDeadLettersHandler),
		protected("POST /api/v1/admin/jobs/dead-letters/{id}/requeue", a.adminJobDeadLetterRequeueHandler),
		protected("DELETE /api/v1/admin/jobs/dead-letters/{id}", a.adminJobDeadLetterDeleteHandler),
	}
	if len(a.cfg.Hooks) > 0 {
		routes = append(routes, protected("GET /api/v1/admin/system/services", a.adminServicesHandler))
	}
	// The audit log is the core's (audit_events is created by core, and every
	// admin write is recorded whichever modules are on), so its explorer must not
	// wait for pages: crm and erp-lite have none and still show the Audit button.
	routes = append(routes, protected("GET /api/v1/admin/audit", a.adminAuditHandler))
	routes = append(routes, protected("POST /api/v1/admin/audit/verify", a.adminAuditVerifyHandler))
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
	Branding     adminBranding            `json:"branding"`
}

// adminBranding is the name, logo and accent colour an administrator set in
// Site settings; empty fields mean the HydroGo defaults.
type adminBranding struct {
	Name    string `json:"name"`
	LogoURL string `json:"logo_url"`
	Color   string `json:"color"`
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
	// AdminDefault is the admin panel language an administrator chose for the
	// project in Site settings; empty until they do.
	AdminDefault string `json:"admin_default"`
}

func (a *serveApp) adminSchema(w http.ResponseWriter, r *http.Request) {
	if user := auth.UserFromContext(r.Context()); user == nil || !user.Active {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	defs := a.contentTypesForContext(r.Context())
	schema := a.buildAdminSchemaForTypes(defs)
	schema.Modules["orchestrator"] = a.isOrchestratorRequest(r)
	schema.Resources = contenttypes.HideUnreadableFields(schema.Resources, defs, func(permission string) bool { return contentTypeCan(r, permission) })
	schema.Locales.AdminDefault = a.adminDefaultLocale(r.Context())
	schema.Branding = a.adminBranding(r.Context())
	api.WriteData(w, http.StatusOK, schema, nil)
}

// adminDefaultLocale reads the project-wide admin language setting. A missing
// site module, setting or an unsupported value all mean "no project default".
func (a *serveApp) adminDefaultLocale(ctx context.Context) string {
	if !a.modules["site"] {
		return ""
	}
	setting, found, err := a.siteStore.GetSetting(ctx, a.db, "admin.default_locale")
	if err != nil || !found || !i18n.IsSupportedCatalogLocale(setting.Value) {
		return ""
	}
	return setting.Value
}

// adminBrandingHandler serves the branding and default language without
// authentication; both are already public site settings.
func (a *serveApp) adminBrandingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	api.WriteData(w, http.StatusOK, struct {
		adminBranding
		DefaultLocale string `json:"default_locale"`
	}{a.adminBranding(r.Context()), a.adminDefaultLocale(r.Context())}, nil)
}

func (a *serveApp) adminBranding(ctx context.Context) adminBranding {
	if !a.modules["site"] {
		return adminBranding{}
	}
	get := func(key string) string {
		setting, found, err := a.siteStore.GetSetting(ctx, a.db, key)
		if err != nil || !found {
			return ""
		}
		return setting.Value
	}
	branding := adminBranding{Name: get("admin.brand_name"), LogoURL: get("admin.brand_logo_url"), Color: get("admin.brand_color")}
	if !sitetransport.ValidBrandColor(branding.Color) {
		branding.Color = ""
	}
	return branding
}

func (a *serveApp) buildAdminSchema() adminSchema {
	return a.buildAdminSchemaForTypes(a.getContentTypes())
}

func (a *serveApp) buildAdminSchemaForTypes(defs []contenttypes.Definition) adminSchema {
	localized := a.contentLocales.Enabled()
	resources := contenttypes.AdminCollections(defs, localized)
	resources = append(resources, a.activeStudioCollections()...)
	schema := adminSchema{
		Version:      1,
		Resources:    resources,
		ContentTypes: make([]adminSchemaContentType, 0, len(defs)),
		Modules:      make(map[string]bool, len(a.modules)),
		Locales:      adminSchemaLocales{Enabled: localized, Default: a.contentLocales.Default, All: append([]string{}, a.contentLocales.All...)},
		Permissions:  a.authSvc.AvailablePermissions(),
	}
	for _, definition := range defs {
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
	// Not a module: it tells the admin that hook services are configured, so it
	// asks for them (the endpoint does not exist otherwise) instead of getting a 404.
	if len(a.cfg.Hooks) > 0 {
		schema.Modules["hooks"] = true
	}
	return schema
}
