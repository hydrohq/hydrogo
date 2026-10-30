package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

func (a *serveApp) adminRolesListHandler(w http.ResponseWriter, r *http.Request) {
	if !a.requireUsersManage(w, r) {
		return
	}
	if a.roleStore == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "roles store unavailable", nil)
		return
	}
	roles, err := a.roleStore.ListRoles(r.Context())
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	if roles == nil {
		roles = []auth.CustomRole{}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteData(w, http.StatusOK, roles, nil)
}

func (a *serveApp) adminRoleGetHandler(w http.ResponseWriter, r *http.Request) {
	if !a.requireUsersManage(w, r) {
		return
	}
	if a.roleStore == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "roles store unavailable", nil)
		return
	}
	id := r.PathValue("id")
	role, err := a.roleStore.GetRole(r.Context(), id)
	if err != nil {
		if errors.Is(err, auth.ErrRoleNotFound) {
			api.WriteError(w, r, http.StatusNotFound, "role not found", nil)
			return
		}
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteData(w, http.StatusOK, role, nil)
}

type adminRoleInput struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Permissions []auth.RolePermission `json:"permissions"`
}

func (a *serveApp) adminRoleSaveHandler(w http.ResponseWriter, r *http.Request) {
	if !a.requireUsersManage(w, r) {
		return
	}
	if a.roleStore == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "roles store unavailable", nil)
		return
	}

	var input adminRoleInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&input); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	if pathID := r.PathValue("id"); pathID != "" {
		input.ID = pathID
	}
	input.ID = strings.TrimSpace(strings.ToLower(input.ID))
	if !auth.ValidCustomRoleName(input.ID) {
		api.WriteError(w, r, http.StatusBadRequest, "role id must be 2-64 characters matching [a-z][a-z0-9_]*", nil)
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		api.WriteError(w, r, http.StatusBadRequest, "role name is required", nil)
		return
	}

	role := auth.CustomRole{
		ID:          input.ID,
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		Permissions: input.Permissions,
	}

	if err := a.roleStore.SaveRole(r.Context(), role); err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	saved, err := a.roleStore.GetRole(r.Context(), role.ID)
	if err != nil {
		api.WriteData(w, http.StatusOK, role, nil)
		return
	}
	api.WriteData(w, http.StatusOK, saved, nil)
}

func (a *serveApp) adminRoleDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !a.requireUsersManage(w, r) {
		return
	}
	if a.roleStore == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "roles store unavailable", nil)
		return
	}

	id := r.PathValue("id")
	if err := a.roleStore.DeleteRole(r.Context(), id); err != nil {
		if errors.Is(err, auth.ErrRoleNotFound) {
			api.WriteError(w, r, http.StatusNotFound, "role not found", nil)
			return
		}
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	api.WriteData(w, http.StatusOK, map[string]bool{"deleted": true}, nil)
}
