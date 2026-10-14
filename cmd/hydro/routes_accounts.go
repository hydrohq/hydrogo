package main

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	pushservice "github.com/ardaozyildirim/hydrogo/internal/push"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/server"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

// authRoutes registers the admin auth API plus, with the auth module, the
// invitation acceptance page and token-lifetime settings.
func (a *serveApp) authRoutes() []router.Route {
	cfg, modules, authHTTP := a.cfg, a.modules, a.authHTTP
	routes := append([]router.Route{}, authHTTP.RouterRoutes()...)
	if modules["auth"] {
		routes = append(routes, router.Route{Pattern: "GET /auth/invites/accept", Access: router.Public, Handler: http.HandlerFunc(inviteAcceptPage)})
		tokenSettingsHandler := adminTokenLifetimeSettingsHandler{
			Settings: authHTTP.TokenLifetimeSettings,
			Defaults: auth.TokenLifetimes{PasswordReset: cfg.PasswordResetDuration(), Invitation: cfg.InvitationDuration()},
			Audit:    a.auditRecorder,
		}
		routes = append(routes,
			router.Route{Pattern: "GET /admin/security/token-lifetimes", Access: router.Protected, Handler: tokenSettingsHandler},
			router.Route{Pattern: "POST /admin/security/token-lifetimes", Access: router.Protected, Handler: tokenSettingsHandler},
		)
	}
	return routes
}

func (a *serveApp) pushRoutes() []router.Route {
	if !a.modules["push"] {
		return nil
	}
	pushDispatcher, newRateLimiter := a.pushDispatcher, a.newRateLimiter
	return (pushservice.Handler{Dispatcher: *pushDispatcher, Limiter: newRateLimiter(3, time.Minute), Audit: a.auditRecorder}).Routes()
}

func (a *serveApp) sessionRoutes() []router.Route {
	sessionPage := server.SessionPageHandler{Service: a.authSvc}
	return []router.Route{{Pattern: "GET /auth/me/sessions/manage", Access: router.Protected, Handler: sessionPage}, {Pattern: "POST /auth/me/sessions/manage/revoke-others", Access: router.Protected, Handler: sessionPage}, {Pattern: "POST /auth/me/sessions/manage/{id}/revoke", Access: router.Protected, Handler: sessionPage}}
}

func (a *serveApp) memberRoutes() []router.Route {
	if !enabledModuleSet(a.cfg.Modules)["members"] {
		return nil
	}
	return a.memberHTTP.Routes()
}

// publicAccountRoutes registers the password-reset and admin login pages.
func (a *serveApp) publicAccountRoutes() []router.Route {
	return []router.Route{
		{Pattern: "GET /password-reset", Access: router.Public, Handler: http.HandlerFunc(passwordResetPage)},
		{Pattern: "GET /login", Access: router.Public, Handler: http.HandlerFunc(a.adminLoginPage)},
	}
}

// accountTokenPageState validates the token and error query parameters shared
// by the invitation and password-reset pages.
func accountTokenPageState(r *http.Request) (token, locale, message string) {
	token = r.URL.Query().Get("token")
	validToken := len(token) == 64
	if validToken {
		_, err := hex.DecodeString(token)
		validToken = err == nil
	}
	locale = "en"
	if r.URL.Query().Get("lang") == "tr" {
		locale = "tr"
	}
	message = r.URL.Query().Get("error")
	switch message {
	case "mismatch", "weak", "rate", "unavailable":
	default:
		if !validToken {
			message = "invalid"
		} else {
			message = ""
		}
	}
	return token, locale, message
}

func setAccountTokenPageHeaders(w http.ResponseWriter, locale string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Language", locale)
}

func inviteAcceptPage(w http.ResponseWriter, r *http.Request) {
	token, locale, message := accountTokenPageState(r)
	setAccountTokenPageHeaders(w, locale)
	var page bytes.Buffer
	props := designsystem.InviteAcceptFormProps{Token: token, CSRF: security.CSRFTokenFromContext(r.Context()), Locale: locale, Error: message}
	if err := designsystem.InviteAcceptForm(props).Render(r.Context(), &page); err != nil {
		http.Error(w, "Invitation page is temporarily unavailable.", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Security-Policy", security.MergeCSP(w.Header().Get("Content-Security-Policy"), security.InlineCSPDirectives(page.Bytes())))
	_, _ = w.Write(page.Bytes())
}

func passwordResetPage(w http.ResponseWriter, r *http.Request) {
	token, locale, message := accountTokenPageState(r)
	setAccountTokenPageHeaders(w, locale)
	var page bytes.Buffer
	props := designsystem.PasswordResetFormProps{Token: token, CSRF: security.CSRFTokenFromContext(r.Context()), Locale: locale, Error: message}
	if err := designsystem.PasswordResetForm(props).Render(r.Context(), &page); err != nil {
		http.Error(w, "Password reset is temporarily unavailable.", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Security-Policy", security.MergeCSP(w.Header().Get("Content-Security-Policy"), security.InlineCSPDirectives(page.Bytes())))
	_, _ = w.Write(page.Bytes())
}
