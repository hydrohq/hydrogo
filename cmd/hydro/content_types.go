package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// contentTypeRoutes loads the YAML content types in dir, syncs their schemas
// into collection storage (once per tenant scope when tenancy is on) and
// returns the public read routes and the admin JSON API for them. A missing
// or empty directory yields no routes. Rich text is returned as stored;
// rendering it is the frontend's job.
func contentTypeRoutes(ctx context.Context, dir string, service collections.Service, scopes []tenant.Scope, locales contenttypes.Locales, presence *realtime.Presence, typesProvider ...func() []contenttypes.Definition) ([]contenttypes.Definition, []router.Route, error) {
	types, err := contenttypes.LoadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("content types: %w", err)
	}
	if len(types) == 0 {
		return nil, nil, nil
	}
	schemas, err := contenttypes.Schemas(types)
	if err != nil {
		return nil, nil, err
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
			return nil, nil, fmt.Errorf("content types: sync schemas: %w", err)
		}
	}
	if _, err := adminstudio.NewRegistry(contenttypes.AdminCollections(types, locales.Enabled())); err != nil {
		return nil, nil, fmt.Errorf("content types: admin registry: %w", err)
	}
	var provider func() []contenttypes.Definition
	if len(typesProvider) > 0 {
		provider = typesProvider[0]
	}
	public := contenttypes.PublicAPI{Service: service, Types: types, TypesFunc: provider, Locales: locales}
	admin := contenttypes.AdminAPI{Service: service, Types: types, TypesFunc: provider, Locales: locales, Presence: presence, Actor: contentTypeActor, Authorize: contentTypeAccess, Can: contentTypeCan}
	routes := []router.Route{
		{Pattern: "GET /api/v1/content/{type}", Access: router.Public, Handler: http.HandlerFunc(public.List)},
		{Pattern: "GET /api/v1/content/{type}/{id}", Access: router.Public, Handler: http.HandlerFunc(public.Get)},
	}
	return types, append(routes, admin.Routes()...), nil
}

// contentTypeAccess maps the admin content API onto the content permissions:
// reads need admin.content.read, changes admin.content.write.
func contentTypeAccess(r *http.Request, write bool) bool {
	user := auth.UserFromContext(r.Context())
	if write {
		return auth.Can(user, auth.PermissionContentWrite)
	}
	return auth.Can(user, auth.PermissionContentRead)
}

// contentTypeActor names the signed-in user in content-type revisions.
func contentTypeActor(r *http.Request) string {
	if user := auth.UserFromContext(r.Context()); user != nil {
		return user.Email
	}
	return ""
}

// contentTypeCan answers permission checks of state transitions.
func contentTypeCan(r *http.Request, permission string) bool {
	return auth.Can(auth.UserFromContext(r.Context()), permission)
}
