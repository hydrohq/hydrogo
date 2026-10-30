package main

import (
	"context"

	"github.com/ardaozyildirim/hydrogo/internal/audit"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/mcp"
	"github.com/ardaozyildirim/hydrogo/internal/router"
)

func (a *serveApp) mcpRoutes() []router.Route {
	var detailRecorder audit.DetailRecorder
	if dr, ok := a.auditRecorder.(audit.DetailRecorder); ok {
		detailRecorder = dr
	} else if a.db != nil {
		detailRecorder = newAuditRecorder(a.db, a.cfg.Database.Dialect, a.requireTenantScope)
	}

	handler := mcp.NewHandler(mcp.Config{
		CollectionService: a.collectionService,
		ContentTypesDir:   a.cfg.ContentTypesDir,
		ContentTypes: func() []contenttypes.Definition {
			return a.contentTypes
		},
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
