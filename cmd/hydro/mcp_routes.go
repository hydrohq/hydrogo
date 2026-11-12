package main

import (
	"context"

	"github.com/hydrohq/hydrogo/internal/audit"
	"github.com/hydrohq/hydrogo/internal/contenttypes"
	"github.com/hydrohq/hydrogo/internal/mcp"
	"github.com/hydrohq/hydrogo/internal/router"
)

func (a *serveApp) mcpRoutes() []router.Route {
	var detailRecorder audit.DetailRecorder
	if dr, ok := a.auditRecorder.(audit.DetailRecorder); ok {
		detailRecorder = dr
	} else if a.db != nil {
		detailRecorder = newAuditRecorder(a.db, a.cfg.Database.Dialect, a.requireTenantScope)
	}

	handler := mcp.NewHandler(mcp.Config{
		CollectionService:      a.collectionService,
		ContentTypesDir:        a.cfg.ContentTypesDir,
		RequireTenantScope:     a.requireTenantScope,
		RequireAuth:            a.authSvc != nil,
		ContentTypes:           a.getContentTypes,
		ContentTypesForContext: a.contentTypesForContext,
		SyncSchema: func(ctx context.Context, _ contenttypes.Definition) error {
			return a.syncContentTypes(ctx)
		},
		AuditStore: detailRecorder,
		DB:         a.db,
	})

	return []router.Route{
		{Pattern: "GET /api/v1/mcp", Access: router.Public, Handler: handler},
		{Pattern: "POST /api/v1/mcp", Access: router.Public, Handler: handler},
	}
}
