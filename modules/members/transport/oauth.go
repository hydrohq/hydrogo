package transport

import (
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

// OAuthHandler is the member-facing social sign-in surface. It reuses
// auth.OAuthFlow for the provider-facing PKCE/state/token-exchange and Apple
// id_token verification mechanics (see internal/auth/oauth.go) so that code
// is never duplicated between the admin and member surfaces, but resolves
// identities through a members.OAuthResolver (which auto-provisions a
// member when needed) and issues the member session cookie, never the admin
// one.
type OAuthHandler struct {
	auth.OAuthFlow
	Resolver members.OAuthResolver
	Service  members.Service
	Audit    auth.AuditRecorder
	Secure   bool
}

func (h OAuthHandler) Routes() []router.Route {
	if len(h.Providers) == 0 || h.Resolver == nil || len(h.StateKey) < 32 {
		return nil
	}
	return []router.Route{
		{Pattern: "GET /member/oauth/{provider}/start", Access: router.Public, Handler: http.HandlerFunc(h.OAuthFlow.Start)},
		{Pattern: "GET /member/oauth/{provider}/callback", Access: router.Public, Handler: http.HandlerFunc(h.callback)},
	}
}

func (h OAuthHandler) callback(w http.ResponseWriter, r *http.Request) {
	identity, returnTo, handled := h.HandleCallback(w, r)
	if handled {
		return
	}
	member, err := h.Resolver.ResolveOAuth(r.Context(), identity)
	if err != nil || member == nil || !member.Active {
		http.Error(w, "social login could not be completed", http.StatusUnauthorized)
		return
	}
	token, session, err := h.Service.CreateSession(r.Context(), member.ID, r.UserAgent(), requestmeta.ClientIP(r), 7*24*time.Hour)
	if err != nil {
		http.Error(w, "social login could not be completed", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: members.SessionCookieName, Value: token, Path: "/", Expires: session.ExpiresAt, HttpOnly: true, Secure: h.Secure, SameSite: http.SameSiteLaxMode})
	w.Header().Set("Cache-Control", "no-store")
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), "member.oauth.login", member.ID)
	}
	http.Redirect(w, r, returnTo, http.StatusSeeOther)
}
