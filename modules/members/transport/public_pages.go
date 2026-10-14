package transport

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

// Member sign-in, registration and account pages hand the active theme a
// typed view model (extension/v1 MemberAuthView / MemberAccountView). The
// form behaviour ships as a module-owned script so no theme needs inline code.

const memberAuthScript = "/member/assets/member-auth.js"

//go:embed assets/member-auth.js
var memberAssets embed.FS

func serveMemberAuthScript(w http.ResponseWriter, r *http.Request) {
	data, err := memberAssets.ReadFile("assets/member-auth.js")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeContent(w, r, "member-auth.js", time.Time{}, bytes.NewReader(data))
}

func (h Handler) memberSiteView(locale, title string) v1.ThemeSiteView {
	return v1.ThemeSiteView{
		Locale:    locale,
		Direction: canonical.Direction(locale),
		Title:     title,
		Scripts:   []string{memberAuthScript},
		NewsURL:   "/public/news/" + url.PathEscape(locale) + "/html",
		PagesURL:  "/public/pages/" + url.PathEscape(locale) + "/",
	}
}

func (h Handler) writeMemberAuthPage(w http.ResponseWriter, r *http.Request, register bool) {
	locale := h.defaultLocale()
	w = sanitizeMemberResponseWithLocale(w, locale)
	returnTo := security.SafeReturnPath(r.URL.Query().Get("return_to"), "/member/account")
	view := v1.MemberAuthView{
		Register:             register,
		Action:               "/member/login",
		CSRF:                 security.CSRFTokenFromContext(r.Context()),
		ReturnTo:             returnTo,
		SwitchURL:            "/member/register",
		PasswordAutocomplete: "current-password",
	}
	if register {
		view.Action, view.SwitchURL, view.PasswordAutocomplete = "/member/register", "/member/login", "new-password"
		var strength bytes.Buffer
		if err := designsystem.PasswordStrengthMeter().Render(r.Context(), &strength); err == nil {
			view.PasswordStrength = template.HTML(strength.String())
		}
	}
	if returnTo != "/member/account" {
		view.SwitchURL += "?return_to=" + url.QueryEscape(returnTo)
	}
	view.Site = h.memberSiteView(locale, "")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := v1.RenderThemeView(h.Theme, w, v1.ViewMemberAuth, view); err != nil {
		http.Error(w, "member page unavailable", http.StatusInternalServerError)
	}
}

func (h Handler) accountPage(w http.ResponseWriter, r *http.Request) {
	locale := h.defaultLocale()
	w = sanitizeMemberResponseWithLocale(w, locale)
	cookie, err := r.Cookie(members.SessionCookieName)
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/member/login", http.StatusSeeOther)
		return
	}
	currentSession, member, err := h.Service.ValidateSession(r.Context(), cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/member/login", http.StatusSeeOther)
		return
	}
	sessions, _ := h.Service.Sessions.ListByMemberID(r.Context(), member.ID)

	view := v1.MemberAccountView{
		Email:         member.Email,
		Initial:       "U",
		EmailVerified: member.EmailVerifiedAt != nil,
		RegisteredAt:  member.CreatedAt.Format("02.01.2006"),
		CSRF:          security.CSRFTokenFromContext(r.Context()),
	}
	if member.Email != "" {
		view.Initial = strings.ToUpper(string([]rune(member.Email)[0]))
	}
	if member.CreatedAt.IsZero() {
		view.RegisteredAt = time.Now().Format("02.01.2006")
	}
	for _, s := range sessions {
		agent := strings.TrimSpace(s.UserAgent)
		if len(agent) > 42 {
			agent = agent[:40] + "…"
		}
		view.Sessions = append(view.Sessions, v1.MemberSessionView{
			UserAgent: agent,
			IPAddress: s.IPAddress,
			LastSeen:  s.LastSeenAt.Format("02.01 15:04"),
			Current:   currentSession != nil && s.ID == currentSession.ID,
		})
	}
	view.Site = h.memberSiteView(locale, member.Email)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	if err := v1.RenderThemeView(h.Theme, w, v1.ViewMemberAccount, view); err != nil {
		http.Error(w, "member page unavailable", http.StatusInternalServerError)
	}
}
