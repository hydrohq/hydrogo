package router

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

type Access uint8

const (
	Unclassified Access = iota
	Public
	Protected
)

type Route struct {
	Pattern string
	Access  Access
	Handler http.Handler
	// Permission, on a Protected route, is the staff permission the caller
	// must hold on top of being signed in. Empty means any active staff user
	// (the handler then checks for itself).
	Permission string
}

type Authorizer func(*http.Request) bool

// PermissionChecker reports whether the signed-in caller holds permission.
type PermissionChecker func(r *http.Request, permission string) bool

func New(routes []Route, authorize Authorizer) (http.Handler, error) {
	return NewWithPermissions(routes, authorize, nil)
}

// NewWithPermissions is New plus route-level permissions. A route that
// declares a Permission with no checker configured is a startup error, so a
// permission can never be silently skipped.
func NewWithPermissions(routes []Route, authorize Authorizer, can PermissionChecker) (http.Handler, error) {
	mux := http.NewServeMux()
	seen := make(map[string]struct{}, len(routes)*2)
	for _, route := range routes {
		if route.Pattern == "" || guards.IsNil(route.Handler) {
			return nil, errors.New("route pattern and handler are required")
		}
		for _, pattern := range apiVersionedPatterns(route.Pattern) {
			if _, exists := seen[pattern]; exists {
				return nil, fmt.Errorf("duplicate route pattern %q", pattern)
			}
			seen[pattern] = struct{}{}
			handler := route.Handler
			if isLegacyAPIPath(pattern) {
				handler = legacyAPIHandler(handler)
			}
			switch route.Access {
			case Public:
				if route.Permission != "" {
					return nil, fmt.Errorf("public route %q cannot require permission %q", route.Pattern, route.Permission)
				}
				mux.Handle(pattern, handler)
			case Protected:
				if authorize == nil {
					return nil, fmt.Errorf("protected route %q requires an authorizer", route.Pattern)
				}
				if route.Permission != "" {
					if can == nil {
						return nil, fmt.Errorf("route %q requires permission %q but no permission checker is configured", route.Pattern, route.Permission)
					}
					handler = requirePermission(can, route.Permission, handler)
				}
				mux.Handle(pattern, requireAuthorization(authorize, handler))
			default:
				return nil, fmt.Errorf("route %q is not classified", route.Pattern)
			}
		}
	}
	return mux, nil
}

func apiVersionedPatterns(pattern string) []string {
	method, path, ok := strings.Cut(pattern, " ")
	if !ok || !strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/api/v1/") {
		return []string{pattern}
	}
	return []string{pattern, method + " /api/v1" + strings.TrimPrefix(path, "/api")}
}

func isLegacyAPIPath(pattern string) bool {
	_, path, ok := strings.Cut(pattern, " ")
	return ok && strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/api/v1/")
}

func legacyAPIHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Deprecation", "true")
		w.Header().Set("Link", "</api/v1>; rel=\"successor-version\"")
		next.ServeHTTP(w, r)
	})
}

func requireAuthorization(authorize Authorizer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !authorize(r) {
			w.Header().Set("Cache-Control", "no-store")
			api.WriteError(w, r, http.StatusUnauthorized, "authentication required", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requirePermission(can PermissionChecker, permission string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !can(r, permission) {
			w.Header().Set("Cache-Control", "no-store")
			api.WriteError(w, r, http.StatusForbidden, "permission required: "+permission, nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
