package main

import (
	"encoding/json"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type adminExtensionChannelRegistry interface {
	BindChannel(string, string, adminstudio.ChannelPrincipal) error
	RevokeChannel(string, adminstudio.ChannelPrincipal)
}

func adminExtensionChannelPrincipal(r *http.Request) (adminstudio.ChannelPrincipal, bool) {
	user, session := auth.UserFromContext(r.Context()), auth.SessionFromContext(r.Context())
	if user == nil || session == nil || !user.Active || user.Role != auth.RoleAdmin || session.UserID != user.ID {
		return adminstudio.ChannelPrincipal{}, false
	}
	principal := adminstudio.ChannelPrincipal{SessionID: session.ID, UserID: user.ID}
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		principal.TenantID = scope.TenantID
	}
	return principal, true
}

func adminExtensionChannelBindHandler(registry adminExtensionChannelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := adminExtensionChannelPrincipal(r)
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var input struct {
			ChannelID string `json:"channel_id"`
			Challenge string `json:"challenge"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if err := registry.BindChannel(input.ChannelID, input.Challenge, principal); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func adminExtensionChannelRevokeHandler(registry adminExtensionChannelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := adminExtensionChannelPrincipal(r)
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var input struct {
			ChannelID string `json:"channel_id"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 512)
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.ChannelID == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		// Deliberately idempotent: callers do not learn whether a channel existed.
		registry.RevokeChannel(input.ChannelID, principal)
		w.WriteHeader(http.StatusNoContent)
	}
}
