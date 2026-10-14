package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"

	"github.com/a-h/templ"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
)

// contentTypeRoutes loads the YAML content types in dir, syncs their schemas
// into collection storage (once per tenant scope when tenancy is on) and
// returns the admin and public routes for them. A missing or empty directory
// yields no routes.
func contentTypeRoutes(ctx context.Context, dir string, service collections.Service, scopes []tenant.Scope) ([]router.Route, error) {
	types, err := contenttypes.LoadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("content types: %w", err)
	}
	if len(types) == 0 {
		return nil, nil
	}
	schemas, err := contenttypes.Schemas(types)
	if err != nil {
		return nil, err
	}
	syncContexts := []context.Context{ctx}
	if len(scopes) > 0 {
		syncContexts = syncContexts[:0]
		for _, scope := range scopes {
			syncContexts = append(syncContexts, tenant.WithScope(ctx, scope))
		}
	}
	for _, syncCtx := range syncContexts {
		if err := service.SyncSchemas(syncCtx, schemas); err != nil {
			return nil, fmt.Errorf("content types: sync schemas: %w", err)
		}
	}
	registry, err := adminstudio.NewRegistry(contenttypes.AdminCollections(types))
	if err != nil {
		return nil, fmt.Errorf("content types: admin registry: %w", err)
	}
	api := contenttypes.AdminAPI{Service: service, Types: types}
	admin := adminstudio.GeneratedAdminHandler{Registry: registry, Options: adminstudio.GeneratedAdminOptions{
		API:             api,
		Authorize:       isActiveAdmin,
		CanTransition:   api.CanTransition,
		Transition:      api.Transition,
		CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
		CSRFToken:       func(r *http.Request) string { return security.CSRFTokenFromContext(r.Context()) },
		ValidateCSRF: func(r *http.Request) bool {
			expected := security.CSRFTokenFromContext(r.Context())
			return expected != "" && r.ParseForm() == nil && subtle.ConstantTimeCompare([]byte(r.FormValue("_csrf")), []byte(expected)) == 1
		},
		RenderPage: renderContentTypePage,
	}}
	// Register validates the options; the routes themselves go through the
	// application router so they get the standard protection and logging.
	if err := admin.Register(http.NewServeMux()); err != nil {
		return nil, fmt.Errorf("content types: admin: %w", err)
	}
	public := contenttypes.PublicAPI{Service: service, Types: types, RenderRichText: ui.RichTextHTML}
	// Patterns carry methods: a method-less prefix would conflict with the
	// public "GET /{locale}/" route in net/http's mux.
	return []router.Route{
		{Pattern: "GET " + contenttypes.AdminBasePath, Access: router.Protected, Handler: admin},
		{Pattern: "GET " + contenttypes.AdminBasePath + "/", Access: router.Protected, Handler: admin},
		{Pattern: "POST " + contenttypes.AdminBasePath + "/", Access: router.Protected, Handler: admin},
		{Pattern: "GET /api/v1/content/{type}", Access: router.Public, Handler: http.HandlerFunc(public.List)},
		{Pattern: "GET /api/v1/content/{type}/{id}", Access: router.Public, Handler: http.HandlerFunc(public.Get)},
	}, nil
}

func isActiveAdmin(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return user != nil && user.Active && user.Role == auth.RoleAdmin
}

func renderContentTypePage(w http.ResponseWriter, r *http.Request, title string, body templ.Component) error {
	if err := ui.AdminStartForRequest(w, r, title, "", "content-types"); err != nil {
		return err
	}
	if err := body.Render(r.Context(), w); err != nil {
		return err
	}
	return ui.AdminEnd(w)
}
