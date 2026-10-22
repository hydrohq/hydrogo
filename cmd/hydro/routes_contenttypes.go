package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"gopkg.in/yaml.v3"
)

// contentTypeAdminRoutes registers the content types schema management API.
func (a *serveApp) contentTypeAdminRoutes() []router.Route {
	protected := func(pattern string, handler http.HandlerFunc) router.Route {
		return router.Route{Pattern: pattern, Access: router.Protected, Handler: handler}
	}
	return []router.Route{
		protected("GET /api/v1/admin/content-types", a.adminListContentTypes),
		protected("POST /api/v1/admin/content-types", a.adminCreateContentType),
		protected("DELETE /api/v1/admin/content-types/{id}", a.adminDeleteContentType),
	}
}

func (a *serveApp) syncContentTypes(ctx context.Context) error {
	types, err := contenttypes.LoadDir(a.cfg.ContentTypesDir)
	if err != nil {
		return fmt.Errorf("reload content types: %w", err)
	}
	schemas, err := contenttypes.Schemas(types)
	if err != nil {
		return fmt.Errorf("compile schemas: %w", err)
	}
	syncContexts := []context.Context{ctx}
	if len(a.tenantScopes) > 0 {
		syncContexts = syncContexts[:0]
		for _, scope := range a.tenantScopes {
			syncContexts = append(syncContexts, tenant.WithScope(ctx, scope))
		}
	}
	for _, syncCtx := range syncContexts {
		if err := a.collectionService.SyncSchemas(syncCtx, schemas); err != nil {
			return fmt.Errorf("sync schemas: %w", err)
		}
	}
	a.contentTypes = types
	return nil
}

func (a *serveApp) adminListContentTypes(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !auth.Can(user, auth.PermissionContentRead) && !activePlatformAdmin(user) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	api.WriteData(w, http.StatusOK, a.contentTypes, nil)
}

func (a *serveApp) adminCreateContentType(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !auth.Can(user, auth.PermissionContentWrite) && !activePlatformAdmin(user) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}

	var def contenttypes.Definition
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	def.ID = strings.TrimSpace(strings.ToLower(def.ID))
	def.Label = strings.TrimSpace(def.Label)

	if err := def.Validate(); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, err.Error(), nil)
		return
	}

	for _, existing := range a.contentTypes {
		if existing.ID == def.ID {
			api.WriteError(w, r, http.StatusConflict, fmt.Sprintf("content type %q already exists", def.ID), nil)
			return
		}
	}

	yamlBytes, err := yaml.Marshal(&def)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "failed to serialize definition", nil)
		return
	}

	dir := a.cfg.ContentTypesDir
	if dir == "" {
		dir = "content-types"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil { // #nosec G301 -- content-types directory of the project
		api.WriteError(w, r, http.StatusInternalServerError, "failed to prepare content types directory", nil)
		return
	}

	filePath := filepath.Join(dir, def.ID+".yaml")
	if err := os.WriteFile(filePath, yamlBytes, 0o644); err != nil { // #nosec G306 -- content type definition: a project source file with no secrets
		api.WriteError(w, r, http.StatusInternalServerError, "failed to write definition file", nil)
		return
	}

	if err := a.syncContentTypes(r.Context()); err != nil {
		_ = os.Remove(filePath)
		api.WriteError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to sync schema: %v", err), nil)
		return
	}

	api.WriteData(w, http.StatusCreated, def, nil)
}

func (a *serveApp) adminDeleteContentType(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !auth.Can(user, auth.PermissionContentWrite) && !activePlatformAdmin(user) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}

	id := strings.TrimSpace(strings.ToLower(r.PathValue("id")))
	if id == "" {
		api.WriteError(w, r, http.StatusBadRequest, "missing content type id", nil)
		return
	}

	found := false
	for _, existing := range a.contentTypes {
		if existing.ID == id {
			found = true
			break
		}
	}
	if !found {
		api.WriteError(w, r, http.StatusNotFound, "content type not found", nil)
		return
	}

	dir := a.cfg.ContentTypesDir
	if dir == "" {
		dir = "content-types"
	}
	filePath := filepath.Join(dir, id+".yaml")
	if err := os.Remove(filePath); err != nil && !errors.Is(err, os.ErrNotExist) { // #nosec G703 -- id equals the ID of an already loaded content type
		api.WriteError(w, r, http.StatusInternalServerError, "failed to delete definition file", nil)
		return
	}

	if err := a.syncContentTypes(r.Context()); err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to sync after deletion: %v", err), nil)
		return
	}

	api.WriteData(w, http.StatusOK, map[string]bool{"deleted": true}, nil)
}
