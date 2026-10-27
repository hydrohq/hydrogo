package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/ardaozyildirim/hydrogo/internal/richtext"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"golang.org/x/sync/errgroup"
)

// contentTypeRoutes loads the YAML content types in dir, syncs their schemas
// into collection storage (once per tenant scope when tenancy is on) and
// returns the public read routes and the admin JSON API for them. A missing
// or empty directory yields no routes. Rich text is returned as stored;
// rendering it is the frontend's job. publicLimiter bounds the public
// list/get routes per caller IP; nil leaves them unbounded (only used by
// the test helper below).
func contentTypeRoutes(ctx context.Context, dir string, service collections.Service, scopes []tenant.Scope, locales contenttypes.Locales, presence *realtime.Presence, hook contenttypes.HookCaller, publicLimiter security.RateLimiter, typesProvider ...func() []contenttypes.Definition) ([]contenttypes.Definition, []router.Route, error) {
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
	var group errgroup.Group
	for _, syncCtx := range syncContexts {
		group.Go(func() error { return service.SyncSchemas(syncCtx, schemas) })
	}
	if err := group.Wait(); err != nil {
		return nil, nil, fmt.Errorf("content types: sync schemas: %w", err)
	}
	if _, err := adminstudio.NewRegistry(contenttypes.AdminCollections(types, locales.Enabled())); err != nil {
		return nil, nil, fmt.Errorf("content types: admin registry: %w", err)
	}
	var provider func() []contenttypes.Definition
	if len(typesProvider) > 0 {
		provider = typesProvider[0]
	}
	// Clean on read as well as on write: records saved before rich text was
	// sanitized on save are still served to storefronts that use innerHTML.
	public := contenttypes.PublicAPI{Service: service, Types: types, TypesFunc: provider, Locales: locales, RenderRichText: richtext.Clean}
	admin := contenttypes.AdminAPI{Service: service, Types: types, TypesFunc: provider, Locales: locales, Presence: presence, Actor: contentTypeActor, Subject: contentTypeActor, Hook: hook, Authorize: contentTypeAccess, Can: contentTypeCan}
	routes := []router.Route{
		{Pattern: "GET /api/v1/content/{type}", Access: router.Public, Handler: rateLimitedPublicContent(publicLimiter, http.HandlerFunc(public.List))},
		{Pattern: "GET /api/v1/content/{type}/{id}", Access: router.Public, Handler: rateLimitedPublicContent(publicLimiter, http.HandlerFunc(public.Get))},
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

// rateLimitedPublicContent bounds the public content API per caller IP.
// GET /api/v1/content/{type} accepts a search= parameter that forces a full
// table scan on SQLite/MySQL (no index can serve a LIKE '%...%'); without a
// limit here one unauthenticated caller can pin a core looping that query.
func rateLimitedPublicContent(limiter security.RateLimiter, next http.Handler) http.Handler {
	if limiter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !security.AllowForRequest(limiter, r.Context(), "public-content:"+requestmeta.ClientIP(r)) {
			security.WriteRateLimitedResponse(w, "too many requests", time.Minute)
			return
		}
		next.ServeHTTP(w, r)
	})
}
