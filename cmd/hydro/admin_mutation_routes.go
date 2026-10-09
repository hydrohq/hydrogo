package main

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/router"
)

func adminBackupRoutes(handler http.Handler) []router.Route {
	return []router.Route{
		{Pattern: "GET /admin/backup", Access: router.Protected, Handler: handler},
		{Pattern: "POST /admin/backup", Access: router.Protected, Handler: handler},
	}
}

func adminSecurityBanRoutes(create, remove http.Handler) []router.Route {
	return []router.Route{
		{Pattern: "POST /admin/security/bans", Access: router.Protected, Handler: create},
		{Pattern: "POST /admin/security/bans/remove", Access: router.Protected, Handler: remove},
	}
}

func activePlatformAdmin(user *auth.User) bool {
	return user != nil && user.Active && user.Role == auth.RoleAdmin
}
