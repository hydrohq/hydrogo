package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	auditstore "github.com/hydrohq/hydrogo/internal/audit/sqlstore"
	"github.com/hydrohq/hydrogo/internal/tenant"
)

func (h *Handler) toolSystemInfo(ctx context.Context) (ToolCallResult, error) {
	types := h.contentTypes(ctx)
	typeIDs := make([]string, 0, len(types))
	for _, t := range types {
		typeIDs = append(typeIDs, t.ID)
	}

	tenantID := ""
	if scope, err := tenant.FromContext(ctx); err == nil {
		tenantID = strings.TrimSpace(scope.TenantID)
	}

	info := map[string]any{
		"service":             "HydroGo",
		"version":             "2.0.0",
		"content_types_count": len(types),
		"content_types":       typeIDs,
		"tenant_scoped":       h.cfg.RequireTenantScope,
		"current_tenant":      tenantID,
		"auth_required":       h.cfg.RequireAuth,
		"audit_recorder":      h.cfg.AuditStore != nil,
		"audit_chain_enabled": auditstore.ChainEnabled(),
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: err.Error()}}}, nil
	}

	return ToolCallResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func (h *Handler) toolSystemVerifyAudit(ctx context.Context) (ToolCallResult, error) {
	sqlDB, ok := h.cfg.DB.(*sql.DB)
	if !ok || sqlDB == nil {
		res := map[string]any{
			"supported": false,
			"message":   "live SQL database connection not provided to MCP handler",
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		return ToolCallResult{Content: []ToolContent{{Type: "text", Text: string(data)}}}, nil
	}

	tenantID := ""
	if scope, err := tenant.FromContext(ctx); err == nil {
		tenantID = strings.TrimSpace(scope.TenantID)
	}

	store := auditstore.Store{
		DB:                 sqlDB,
		RequireTenantScope: h.cfg.RequireTenantScope,
	}

	report, err := store.VerifyChain(ctx, sqlDB, tenantID, nil)
	if err != nil {
		return ToolCallResult{
			IsError: true,
			Content: []ToolContent{{Type: "text", Text: "audit verification failed: " + err.Error()}},
		}, nil
	}

	res := map[string]any{
		"valid":         len(report.Problems) == 0,
		"tenant_id":     report.TenantID,
		"total_links":   report.Links,
		"legacy_events": report.Legacy,
		"head_seq":      report.Head.Seq,
		"head_hash":     report.Head.Hash,
		"problems":      report.Problems,
	}

	data, _ := json.MarshalIndent(res, "", "  ")
	return ToolCallResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}
