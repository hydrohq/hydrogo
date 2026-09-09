package router

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

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
}

type Authorizer func(*http.Request) bool

func New(routes []Route, authorize Authorizer) (http.Handler, error) {
	mux := http.NewServeMux()
	for _, route := range routes {
		if route.Pattern == "" || guards.IsNil(route.Handler) {
			return nil, errors.New("route pattern and handler are required")
		}
		for _, pattern := range apiVersionedPatterns(route.Pattern) {
			handler := route.Handler
			if isLegacyAPIPath(pattern) {
				handler = legacyAPIHandler(handler)
			}
			switch route.Access {
			case Public:
				mux.Handle(pattern, handler)
			case Protected:
				if authorize == nil {
					return nil, fmt.Errorf("protected route %q requires an authorizer", route.Pattern)
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
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
