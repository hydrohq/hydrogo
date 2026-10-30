package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
		protected("PUT /api/v1/admin/content-types/{id}", a.adminUpdateContentType),
		protected("DELETE /api/v1/admin/content-types/{id}", a.adminDeleteContentType),
	}
}

func (a *serveApp) contentTypesDirForContext(ctx context.Context) string {
	dir := a.cfg.ContentTypesDir
	if dir == "" {
		dir = "content-types"
	}
	if a.requireTenantScope && ctx != nil {
		if scope, err := tenant.FromContext(ctx); err == nil && scope.TenantID != "" {
			return filepath.Join(dir, "tenants", scope.TenantID)
		}
	}
	return dir
}

func (a *serveApp) syncContentTypes(ctx context.Context) error {
	if a.requireTenantScope && ctx != nil {
		if scope, err := tenant.FromContext(ctx); err == nil && scope.TenantID != "" {
			tDir := a.contentTypesDirForContext(ctx)
			tTypes, loadErr := contenttypes.LoadDir(tDir)
			if loadErr != nil {
				return fmt.Errorf("reload tenant content types: %w", loadErr)
			}
			a.stateMu.Lock()
			if a.tenantContentTypes == nil {
				a.tenantContentTypes = make(map[string][]contenttypes.Definition)
			}
			a.tenantContentTypes[scope.TenantID] = tTypes
			a.stateMu.Unlock()

			merged := a.contentTypesForContext(ctx)
			schemas, schemaErr := contenttypes.Schemas(merged)
			if schemaErr != nil {
				return fmt.Errorf("compile tenant schemas: %w", schemaErr)
			}
			if err := a.collectionService.SyncSchemas(ctx, schemas); err != nil {
				return fmt.Errorf("sync tenant schemas: %w", err)
			}
			return nil
		}
	}

	dir := a.cfg.ContentTypesDir
	if dir == "" {
		dir = "content-types"
	}
	types, err := contenttypes.LoadDir(dir)
	if err != nil {
		return fmt.Errorf("reload content types: %w", err)
	}
	schemas, err := contenttypes.Schemas(types)
	if err != nil {
		return fmt.Errorf("compile schemas: %w", err)
	}
	scopes := a.getTenantScopes()
	syncContexts := []context.Context{ctx}
	if len(scopes) > 0 {
		syncContexts = syncContexts[:0]
		for _, scope := range scopes {
			syncContexts = append(syncContexts, tenant.WithScope(ctx, scope))
		}
	}
	for _, syncCtx := range syncContexts {
		if err := a.collectionService.SyncSchemas(syncCtx, schemas); err != nil {
			return fmt.Errorf("sync schemas: %w", err)
		}
	}
	a.stateMu.Lock()
	a.contentTypes = types
	a.stateMu.Unlock()
	return nil
}

func (a *serveApp) adminListContentTypes(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !auth.Can(user, auth.PermissionContentRead) && !activePlatformAdmin(user) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	api.WriteData(w, http.StatusOK, a.contentTypesForContext(r.Context()), nil)
}

func (a *serveApp) adminCreateContentType(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !auth.Can(user, auth.PermissionContentWrite) && !activePlatformAdmin(user) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}

	bodyBytes, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "failed to read request body", nil)
		return
	}

	var def contenttypes.Definition
	trimmed := strings.TrimSpace(string(bodyBytes))
	if strings.HasPrefix(trimmed, "{") {
		if err := json.Unmarshal(bodyBytes, &def); err != nil {
			api.WriteError(w, r, http.StatusBadRequest, "invalid JSON payload", nil)
			return
		}
	} else {
		if err := yaml.Unmarshal(bodyBytes, &def); err != nil {
			api.WriteError(w, r, http.StatusBadRequest, "invalid YAML payload", nil)
			return
		}
	}

	def.ID = strings.TrimSpace(strings.ToLower(def.ID))
	def.Label = strings.TrimSpace(def.Label)

	if err := def.Validate(); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, err.Error(), nil)
		return
	}

	for _, existing := range a.contentTypesForContext(r.Context()) {
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

	dir := a.contentTypesDirForContext(r.Context())
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

func (a *serveApp) adminUpdateContentType(w http.ResponseWriter, r *http.Request) {
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
	for _, existing := range a.contentTypesForContext(r.Context()) {
		if existing.ID == id {
			found = true
			break
		}
	}
	if !found {
		api.WriteError(w, r, http.StatusNotFound, "content type not found", nil)
		return
	}

	bodyBytes, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "failed to read request body", nil)
		return
	}

	var def contenttypes.Definition
	trimmed := strings.TrimSpace(string(bodyBytes))
	if strings.HasPrefix(trimmed, "{") {
		if err := json.Unmarshal(bodyBytes, &def); err != nil {
			api.WriteError(w, r, http.StatusBadRequest, "invalid JSON payload", nil)
			return
		}
	} else {
		if err := yaml.Unmarshal(bodyBytes, &def); err != nil {
			api.WriteError(w, r, http.StatusBadRequest, "invalid YAML payload", nil)
			return
		}
	}

	if def.ID == "" {
		def.ID = id
	}
	def.ID = strings.TrimSpace(strings.ToLower(def.ID))
	if def.ID != id {
		api.WriteError(w, r, http.StatusBadRequest, "definition ID does not match route path", nil)
		return
	}
	def.Label = strings.TrimSpace(def.Label)

	if err := def.Validate(); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, err.Error(), nil)
		return
	}

	yamlBytes, err := yaml.Marshal(&def)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "failed to serialize definition", nil)
		return
	}

	dir := a.contentTypesDirForContext(r.Context())
	if err := os.MkdirAll(dir, 0o755); err != nil { // #nosec G301 -- content-types directory of the project
		api.WriteError(w, r, http.StatusInternalServerError, "failed to prepare content types directory", nil)
		return
	}
	filePath := filepath.Join(dir, id+".yaml")

	oldBytes, _ := os.ReadFile(filePath)

	if err := os.WriteFile(filePath, yamlBytes, 0o644); err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "failed to write definition file", nil)
		return
	}

	if err := a.syncContentTypes(r.Context()); err != nil {
		if len(oldBytes) > 0 {
			_ = os.WriteFile(filePath, oldBytes, 0o644)
			_ = a.syncContentTypes(r.Context())
		}
		api.WriteError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to sync schema: %v", err), nil)
		return
	}

	api.WriteData(w, http.StatusOK, def, nil)
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
	for _, existing := range a.contentTypesForContext(r.Context()) {
		if existing.ID == id {
			found = true
			break
		}
	}
	if !found {
		api.WriteError(w, r, http.StatusNotFound, "content type not found", nil)
		return
	}

	dir := a.contentTypesDirForContext(r.Context())
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
