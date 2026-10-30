package main

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/audit"
	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type adminAuditEntry struct {
	ID         string `json:"id"`
	At         string `json:"at"`
	Actor      string `json:"actor"`
	ActorID    string `json:"actor_id,omitempty"`
	Action     string `json:"action"`
	Resource   string `json:"resource"`
	Target     string `json:"target"`
	RequestID  string `json:"request_id,omitempty"`
	BeforeJSON string `json:"before_json,omitempty"`
	AfterJSON  string `json:"after_json,omitempty"`
}

// parseAdminAuditFilter reads the explorer query. Dates are whole days: from
// is inclusive, until is exclusive of the following day.
func parseAdminAuditFilter(r *http.Request) (auditstore.ExplorerFilter, map[string]string) {
	query := r.URL.Query()
	filter := auditstore.ExplorerFilter{Actor: query.Get("actor"), Action: query.Get("action"), Resource: query.Get("resource"), Target: query.Get("target"), Limit: 50}
	fields := map[string]string{}
	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			fields["limit"] = "invalid"
		} else {
			filter.Limit = limit
		}
	}
	if raw := query.Get("offset"); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 || offset > 1_000_000 {
			fields["offset"] = "invalid"
		} else {
			filter.Offset = offset
		}
	}
	for name, value := range map[string]string{"actor": filter.Actor, "action": filter.Action, "resource": filter.Resource, "target": filter.Target} {
		if len(value) > 256 {
			fields[name] = "invalid"
		}
	}
	if raw := query.Get("from"); raw != "" {
		if day, err := time.Parse("2006-01-02", raw); err != nil {
			fields["from"] = "invalid"
		} else {
			filter.From = day.UTC().Format("2006-01-02T15:04:05.000000000Z")
		}
	}
	if raw := query.Get("until"); raw != "" {
		if day, err := time.Parse("2006-01-02", raw); err != nil {
			fields["until"] = "invalid"
		} else {
			filter.Until = day.AddDate(0, 0, 1).UTC().Format("2006-01-02T15:04:05.000000000Z")
		}
	}
	return filter, fields
}

func (a *serveApp) adminAuditHandler(w http.ResponseWriter, r *http.Request) {
	if !activePlatformAdmin(auth.UserFromContext(r.Context())) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	filter, fields := parseAdminAuditFilter(r)
	if len(fields) > 0 {
		api.WriteValidation(w, r, fields)
		return
	}
	explorer := auditstore.Store{DB: a.db, Dialect: a.cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope}
	result, err := explorer.Explore(r.Context(), filter)
	if err != nil {
		slog.Error("admin audit explorer unavailable", "error", err)
		api.WriteError(w, r, http.StatusServiceUnavailable, "audit history is temporarily unavailable", nil)
		return
	}
	entries := make([]adminAuditEntry, 0, len(result.Events))
	for _, event := range result.Events {
		actor := event.ActorEmail
		if actor == "" {
			actor = "system"
			if event.ActorID != "" {
				actor = "unavailable"
			}
		}
		entries = append(entries, adminAuditEntry{ID: event.ID, At: event.CreatedAt, Actor: actor, ActorID: event.ActorID, Action: event.Action, Resource: event.ResourceType, Target: event.TargetID, RequestID: event.RequestID, BeforeJSON: event.BeforeJSON, AfterJSON: event.AfterJSON})
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteList(w, entries, api.PaginationMeta{Limit: filter.Limit, Offset: filter.Offset, Total: result.Total, HasMore: filter.Offset+len(entries) < result.Total})
}

func (a *serveApp) adminAuditVerifyHandler(w http.ResponseWriter, r *http.Request) {
	if !activePlatformAdmin(auth.UserFromContext(r.Context())) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	tenantID := ""
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	} else if a.requireTenantScope {
		api.WriteError(w, r, http.StatusBadRequest, "tenant scope is required", nil)
		return
	}
	store := auditstore.Store{DB: a.db, Dialect: a.cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope}
	service := audit.Service{Store: store, DB: a.db, Dialect: a.cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope}
	res, err := service.VerifyAuditChain(r.Context(), tenantID)
	if err != nil {
		slog.Error("admin audit verify failed", "error", err)
		api.WriteError(w, r, http.StatusInternalServerError, "failed to verify audit chain", nil)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteData(w, http.StatusOK, res, nil)
}
