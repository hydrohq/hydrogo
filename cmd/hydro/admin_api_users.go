package main

import (
	"errors"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

type adminUserItem struct {
	auth.User
	Permissions []string `json:"permissions"`
}

func (a *serveApp) adminUserItem(user *auth.User) adminUserItem {
	granted := make([]string, 0)
	for _, permission := range a.authSvc.AvailablePermissions() {
		if auth.Can(user, permission) {
			granted = append(granted, permission)
		}
	}
	return adminUserItem{User: *user, Permissions: granted}
}

func (a *serveApp) requireUsersManage(w http.ResponseWriter, r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	if !activePlatformAdmin(user) || !auth.Can(user, auth.PermissionUsersManage) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return false
	}
	return true
}

func (a *serveApp) adminUsersListHandler(w http.ResponseWriter, r *http.Request) {
	if !a.requireUsersManage(w, r) {
		return
	}
	if a.userAdmin == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "user directory unavailable", nil)
		return
	}
	users, err := a.userAdmin.List(r.Context())
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "user directory unavailable", nil)
		return
	}
	items := make([]adminUserItem, 0, len(users))
	for _, user := range users {
		items = append(items, a.adminUserItem(user))
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteData(w, http.StatusOK, items, nil)
}

func (a *serveApp) adminUserGetHandler(w http.ResponseWriter, r *http.Request) {
	if !a.requireUsersManage(w, r) {
		return
	}
	user, err := a.authSvc.UserByID(r.Context(), r.PathValue("id"))
	if errors.Is(err, auth.ErrUserNotFound) || (err == nil && user == nil) {
		api.WriteError(w, r, http.StatusNotFound, "user not found", nil)
		return
	}
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "user directory unavailable", nil)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteData(w, http.StatusOK, a.adminUserItem(user), nil)
}

type adminPermissionCatalog struct {
	Catalog      []string               `json:"catalog"`
	RoleDefaults map[auth.Role][]string `json:"role_defaults"`
	Roles        []auth.Role            `json:"roles"`
}

// adminPermissionsHandler lists the permission catalog and what each role
// grants by default, so the users screen can render override checkboxes.
func (a *serveApp) adminPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	if !a.requireUsersManage(w, r) {
		return
	}
	catalog := a.authSvc.AvailablePermissions()
	roles := []auth.Role{auth.RoleAdmin, auth.RoleEditor, auth.RoleViewer}
	defaults := make(map[auth.Role][]string, len(roles))
	for _, role := range roles {
		probe := &auth.User{Role: role, Active: true}
		granted := make([]string, 0)
		for _, permission := range catalog {
			if auth.Can(probe, permission) {
				granted = append(granted, permission)
			}
		}
		defaults[role] = granted
	}
	api.WriteData(w, http.StatusOK, adminPermissionCatalog{Catalog: catalog, RoleDefaults: defaults, Roles: roles}, nil)
}
