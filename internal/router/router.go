package router

import (
	"errors"
	"fmt"
	"net/http"
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
		if route.Pattern == "" || route.Handler == nil {
			return nil, errors.New("route pattern and handler are required")
		}
		switch route.Access {
		case Public:
			mux.Handle(route.Pattern, route.Handler)
		case Protected:
			if authorize == nil {
				return nil, fmt.Errorf("protected route %q requires an authorizer", route.Pattern)
			}
			mux.Handle(route.Pattern, requireAuthorization(authorize, route.Handler))
		default:
			return nil, fmt.Errorf("route %q is not classified", route.Pattern)
		}
	}
	return mux, nil
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
