package auth

import (
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
)

func (h HTTPHandler) me(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	session := SessionFromContext(r.Context())
	expiresAt := time.Time{}
	if session != nil {
		expiresAt = session.ExpiresAt
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"id": user.ID, "email": user.Email, "role": user.Role, "active": user.Active, "session_expires_at": expiresAt})
}

func (h HTTPHandler) listSessions(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	session := SessionFromContext(r.Context())
	if user == nil || session == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	items, err := h.Service.ListUserSessions(r.Context(), user.ID)
	if err != nil {
		writeAuthDependencyError(w, "sessions unavailable", err, http.StatusServiceUnavailable)
		return
	}
	type item struct {
		ID, UserAgent, IPAddress         string
		CreatedAt, LastSeenAt, ExpiresAt time.Time
		Current                          bool
	}
	out := make([]item, 0, len(items))
	for _, x := range items {
		out = append(out, item{ID: x.ID, UserAgent: x.UserAgent, IPAddress: x.IPAddress, CreatedAt: x.CreatedAt, LastSeenAt: x.LastSeenAt, ExpiresAt: x.ExpiresAt, Current: x.ID == session.ID})
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteJSON(w, http.StatusOK, map[string]any{"sessions": out})
}

func (h HTTPHandler) revokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	session := SessionFromContext(r.Context())
	if user == nil || session == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := h.Service.RevokeOtherUserSessions(r.Context(), user.ID, session.TokenHash); err != nil {
		writeAuthDependencyError(w, "sessions could not be revoked", err, http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.FormValue("_csrf") != "" {
		http.Redirect(w, r, "/auth/me/sessions/manage", http.StatusSeeOther)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h HTTPHandler) extendSession(w http.ResponseWriter, r *http.Request) {
	noStoreAuthResponse(w)
	user, current := UserFromContext(r.Context()), SessionFromContext(r.Context())
	oldToken := sessionToken(r)
	if user == nil || !user.Active || current == nil || oldToken == "" || h.Service == nil {
		http.Error(w, "session extension unavailable", http.StatusUnauthorized)
		return
	}
	token, next, err := h.Service.CreateSession(r.Context(), user.ID, r.UserAgent(), clientIP(r), DefaultSessionTTL)
	if err != nil {
		writeAuthDependencyError(w, "session extension failed", err, http.StatusInternalServerError)
		return
	}
	if err := h.Service.RevokeSession(r.Context(), oldToken); err != nil {
		_ = h.Service.RevokeSession(r.Context(), token)
		writeAuthDependencyError(w, "session extension failed", err, http.StatusInternalServerError)
		return
	}
	SetSessionCookie(w, token, next.ExpiresAt, h.Secure)
	api.WriteJSON(w, http.StatusOK, map[string]any{"session_expires_at": next.ExpiresAt})
}
