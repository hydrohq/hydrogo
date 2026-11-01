package main

import (
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// rateLimitedAI bounds an AI endpoint per caller (tenant + authenticated user,
// falling back to the client IP) so one workspace cannot drain the platform's
// AI budget with request spam.
func (a *serveApp) rateLimitedAI(name string, limit int, window time.Duration, next http.HandlerFunc) http.HandlerFunc {
	limiter := a.newRateLimiter(limit, window)
	return func(w http.ResponseWriter, r *http.Request) {
		if !security.AllowForRequest(limiter, r.Context(), "ai:"+name+":"+aiCallerKey(r)) {
			security.WriteRateLimitedResponse(w, "too many AI requests", time.Minute)
			return
		}
		next(w, r)
	}
}

func aiCallerKey(r *http.Request) string {
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil && scope.TenantID != "" {
		tenantID = scope.TenantID
	}
	if user := auth.UserFromContext(r.Context()); user != nil {
		return tenantID + ":" + user.ID
	}
	return tenantID + ":" + requestmeta.ClientIP(r)
}
