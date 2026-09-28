package server

import (
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
)

type SessionPageHandler struct{ Service *auth.AuthService }

func (h SessionPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, current := auth.UserFromContext(r.Context()), auth.SessionFromContext(r.Context())
	if user == nil || current == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if r.Method == http.MethodPost {
		var err error
		if r.URL.Path == "/auth/me/sessions/manage/revoke-others" {
			err = h.Service.RevokeOtherUserSessions(r.Context(), user.ID, current.TokenHash)
		} else if sessionID := r.PathValue("id"); sessionID != "" {
			err = h.Service.RevokeUserSession(r.Context(), user.ID, sessionID, current.ID)
		} else {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "sessions could not be revoked", http.StatusServiceUnavailable)
			return
		}
		http.Redirect(w, r, "/auth/me/sessions/manage", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessions, err := h.Service.ListUserSessions(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "sessions unavailable", http.StatusServiceUnavailable)
		return
	}
	items := make([]ui.ActiveSessionItem, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, ui.ActiveSessionItem{ID: session.ID, UserAgent: session.UserAgent, IPAddress: session.IPAddress, LastSeen: session.LastSeenAt.Format(time.RFC822), Current: session.ID == current.ID})
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Active sessions", "Account security", security.NonceFromContext(r.Context()), "account"); err != nil {
		return
	}
	if err := ui.ActiveSessionsPage(items, security.CSRFTokenFromContext(r.Context())).Render(r.Context(), w); err != nil {
		http.Error(w, "sessions page unavailable", http.StatusInternalServerError)
		return
	}
	_ = ui.AdminEnd(w)
}
