package main

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

func graphQLRoutes(handler http.Handler, limiter security.RateLimiter) []router.Route {
	if handler == nil || limiter == nil {
		return nil
	}
	return []router.Route{{Pattern: "POST /api/graphql", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow("graphql:" + requestmeta.ClientIP(r)) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		handler.ServeHTTP(w, r)
	})}}
}
