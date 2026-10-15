package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// contentTypeRoutes loads the YAML content types in dir, syncs their schemas
// into collection storage (once per tenant scope when tenancy is on) and
// returns the public read routes for them. A missing or empty directory
// yields no routes. Rich text is returned as stored; rendering it is the
// frontend's job.
func contentTypeRoutes(ctx context.Context, dir string, service collections.Service, scopes []tenant.Scope, locales contenttypes.Locales) ([]router.Route, error) {
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
	if _, err := adminstudio.NewRegistry(contenttypes.AdminCollections(types, locales.Enabled())); err != nil {
		return nil, fmt.Errorf("content types: admin registry: %w", err)
	}
	public := contenttypes.PublicAPI{Service: service, Types: types, Locales: locales}
	return []router.Route{
		{Pattern: "GET /api/v1/content/{type}", Access: router.Public, Handler: http.HandlerFunc(public.List)},
		{Pattern: "GET /api/v1/content/{type}/{id}", Access: router.Public, Handler: http.HandlerFunc(public.Get)},
	}, nil
}

func isActiveAdmin(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return user != nil && user.Active && user.Role == auth.RoleAdmin
}

// contentTypeActor names the signed-in user in content-type revisions.
func contentTypeActor(r *http.Request) string {
	if user := auth.UserFromContext(r.Context()); user != nil {
		return user.Email
	}
	return ""
}
