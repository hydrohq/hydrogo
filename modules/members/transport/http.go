package transport

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hydrohq/hydrogo/internal/api"
	"github.com/hydrohq/hydrogo/internal/auth"
	"github.com/hydrohq/hydrogo/internal/contracts/customerorders"
	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/internal/push"
	"github.com/hydrohq/hydrogo/internal/requestmeta"
	"github.com/hydrohq/hydrogo/internal/router"
	"github.com/hydrohq/hydrogo/internal/security"
	"github.com/hydrohq/hydrogo/modules/members"
)

type Handler struct {
	Service              members.Service
	Mobile               *members.MobileAuthService
	Secure               bool
	Limiter              security.RateLimiter
	Audit                auth.AuditRecorder
	Orders               customerorders.Reader
	DB                   database.Executor
	DefaultLocale        string
	Push                 *push.Dispatcher
	Verification         members.EmailVerificationService
	VerificationTx       database.Transactor
	VerificationNotifier auth.EmailVerificationNotifier
	VerificationLimiter  security.RateLimiter
	PublicOrigin         string
	VerificationEnabled  bool
}

func (h Handler) defaultLocale() string {
	if strings.TrimSpace(h.DefaultLocale) != "" {
		return strings.TrimSpace(h.DefaultLocale)
	}
	return "tr"
}

func (h Handler) Routes() []router.Route {
	routes := []router.Route{
		// Consumer/mobile authentication is intentionally separate from the
		// browser-session and staff mobile surfaces. These routes authenticate
		// with member bearer tokens and never read or write browser cookies.
		{Pattern: "POST /api/v1/mobile/member/auth/login", Access: router.Public, Handler: http.HandlerFunc(h.mobileLogin)},
		{Pattern: "POST /api/v1/mobile/member/auth/register", Access: router.Public, Handler: http.HandlerFunc(h.mobileRegister)},
		{Pattern: "POST /api/v1/mobile/member/auth/refresh", Access: router.Public, Handler: http.HandlerFunc(h.mobileRefresh)},
		{Pattern: "POST /api/v1/mobile/member/auth/revoke", Access: router.Public, Handler: http.HandlerFunc(h.mobileRevoke)},
		{Pattern: "POST /api/v1/mobile/member/auth/revoke-all", Access: router.Public, Handler: http.HandlerFunc(h.mobileRevokeAll)},
		{Pattern: "GET /api/v1/mobile/member/auth/devices", Access: router.Public, Handler: http.HandlerFunc(h.mobileDevices)},
		{Pattern: "GET /api/v1/mobile/member/me", Access: router.Public, Handler: http.HandlerFunc(h.mobileMe)},
		{Pattern: "GET /api/v1/mobile/member/orders", Access: router.Public, Handler: http.HandlerFunc(h.mobileOrders)},
		{Pattern: "GET /api/v1/mobile/member/orders/{orderID}", Access: router.Public, Handler: http.HandlerFunc(h.mobileOrder)},
		{Pattern: "POST /member/register", Access: router.Public, Handler: http.HandlerFunc(h.register)},
		{Pattern: "POST /member/login", Access: router.Public, Handler: http.HandlerFunc(h.login)},
		{Pattern: "POST /member/logout", Access: router.Public, Handler: http.HandlerFunc(h.logout)},
		{Pattern: "POST /member/delete", Access: router.Public, Handler: http.HandlerFunc(h.deleteAccount)},
		{Pattern: "POST /member/password", Access: router.Public, Handler: http.HandlerFunc(h.changePassword)},
		{Pattern: "POST /member/email", Access: router.Public, Handler: http.HandlerFunc(h.changeEmail)},
		{Pattern: "GET /member/me", Access: router.Public, Handler: http.HandlerFunc(h.me)},
		{Pattern: "GET /member/orders", Access: router.Public, Handler: http.HandlerFunc(h.memberOrders)},

		{Pattern: "GET /member/email/verify", Access: router.Public, Handler: http.HandlerFunc(h.verifyMemberEmail)},
	}
	if h.Push != nil {
		routes = append(routes,
			router.Route{Pattern: "POST /api/v1/mobile/member/push/subscription", Access: router.Public, Handler: http.HandlerFunc(h.mobilePushSubscribe)},
			router.Route{Pattern: "DELETE /api/v1/mobile/member/push/subscription", Access: router.Public, Handler: http.HandlerFunc(h.mobilePushUnsubscribe)},
		)
	}
	return append(routes, h.adminRoutes()...)
}

func (h Handler) memberOrders(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(members.SessionCookieName)
	if err != nil || cookie.Value == "" || h.Orders == nil || h.DB == nil {
		http.Error(w, "orders unavailable", http.StatusServiceUnavailable)
		return
	}
	_, member, err := h.Service.ValidateSession(r.Context(), cookie.Value)
	if err != nil || member == nil || !member.Active {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	orders, err := h.Orders.ListOrdersByUserID(r.Context(), h.DB, member.ID, 100)
	if err != nil {
		http.Error(w, "orders unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	_ = json.NewEncoder(w).Encode(orders)
}

func (h Handler) mobileOrders(w http.ResponseWriter, r *http.Request) {
	memberID, _, ok := h.memberMobilePrincipal(w, r)
	if !ok {
		return
	}
	if h.Orders == nil || h.DB == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "Orders are temporarily unavailable.", nil)
		return
	}
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			api.WriteError(w, r, http.StatusBadRequest, "Invalid order limit.", nil)
			return
		}
		limit = parsed
	}
	orders, err := h.Orders.ListOrdersByUserID(r.Context(), h.DB, memberID, limit)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "Orders are temporarily unavailable.", nil)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"orders": orders, "limit": limit})
}

func (h Handler) mobileOrder(w http.ResponseWriter, r *http.Request) {
	memberID, _, ok := h.memberMobilePrincipal(w, r)
	if !ok {
		return
	}
	reader, ok := h.Orders.(customerorders.DetailReader)
	if !ok || h.DB == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "Orders are temporarily unavailable.", nil)
		return
	}
	orderID := strings.TrimSpace(r.PathValue("orderID"))
	if orderID == "" || len(orderID) > 256 {
		api.WriteError(w, r, http.StatusBadRequest, "Invalid order.", nil)
		return
	}
	order, err := reader.GetOrderByUserID(r.Context(), h.DB, memberID, orderID)
	if err != nil {
		api.WriteError(w, r, http.StatusNotFound, "Order not found.", nil)
		return
	}
	api.WriteJSON(w, http.StatusOK, order)
}

const maxMemberFormBytes = 16 << 10

func (h Handler) register(w http.ResponseWriter, r *http.Request) {
	if h.Limiter != nil && !security.AllowForRequest(h.Limiter, r.Context(), "member-register:"+requestmeta.ClientIP(r)) {
		http.Error(w, "too many requests", 429)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxMemberFormBytes)
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	var email, password, returnTo string
	if strings.HasPrefix(contentType, "application/json") {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			ReturnTo string `json:"return_to"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "invalid request", 400)
			return
		}
		email, password, returnTo = req.Email, req.Password, req.ReturnTo
	} else {
		if err := r.ParseForm(); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "invalid request", 400)
			return
		}
		email, password, returnTo = r.FormValue("email"), r.FormValue("password"), r.FormValue("return_to")
	}
	m, err := h.Service.Register(r.Context(), email, password)
	if err != nil {
		if errors.Is(err, auth.ErrWeakPassword) {
			http.Error(w, "password too weak: choose a longer, less common password", http.StatusBadRequest)
			return
		}
		if errors.Is(err, members.ErrMemberExists) {
			http.Error(w, "email already registered", 409)
			return
		}
		if errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, "invalid credentials", 400)
			return
		}
		// Do not expose repository/provider details in a public response.
		http.Error(w, "registration unavailable", http.StatusBadRequest)
		return
	}
	h.issueSession(w, r, m, returnTo)
}

func (h Handler) login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxMemberFormBytes)
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	var email, password, returnTo string
	if strings.HasPrefix(contentType, "application/json") {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			ReturnTo string `json:"return_to"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "invalid request", 400)
			return
		}
		email, password, returnTo = req.Email, req.Password, req.ReturnTo
	} else {
		if err := r.ParseForm(); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "invalid request", 400)
			return
		}
		email, password, returnTo = r.FormValue("email"), r.FormValue("password"), r.FormValue("return_to")
	}
	if h.Limiter != nil && !security.AllowForRequest(h.Limiter, r.Context(), "member-login:"+requestmeta.ClientIP(r)+":"+auth.NormalizeEmail(email)) {
		http.Error(w, "too many requests", 429)
		return
	}
	m, err := h.Service.Authenticate(r.Context(), email, password)
	if err != nil {
		http.Error(w, "invalid credentials", 401)
		return
	}
	h.issueSession(w, r, m, returnTo)
}

func (h Handler) issueSession(w http.ResponseWriter, r *http.Request, m *members.Member, returnTo string) {
	token, s, err := h.Service.CreateSession(r.Context(), m.ID, r.UserAgent(), requestmeta.ClientIP(r), 7*24*time.Hour)
	if err != nil {
		http.Error(w, "authentication unavailable", 503)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: members.SessionCookieName, Value: token, Path: "/", Expires: s.ExpiresAt, HttpOnly: true, Secure: h.Secure, SameSite: http.SameSiteLaxMode}) // #nosec G124 -- Secure follows the secure_cookies setting (off only for plain-HTTP development); the cookie is HttpOnly and SameSite
	w.Header().Set("Cache-Control", "no-store")

	if returnTo == "" {
		returnTo = r.FormValue("return_to")
	}
	if returnTo == "" {
		returnTo = r.URL.Query().Get("return_to")
	}
	returnTo = security.SafeReturnPath(returnTo, "/member/account")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"authenticated": true,
		"redirect":      returnTo,
		"member":        map[string]any{"id": m.ID, "email": m.Email},
	})
}

func (h Handler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie(members.SessionCookieName)
	if cookie != nil {
		_ = h.Service.Revoke(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: members.SessionCookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(0, 0), HttpOnly: true, Secure: h.Secure, SameSite: http.SameSiteLaxMode}) // #nosec G124 -- Secure follows the secure_cookies setting (off only for plain-HTTP development); the cookie is HttpOnly and SameSite
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(members.SessionCookieName)
	if err != nil || cookie.Value == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	_, member, err := h.Service.ValidateSession(r.Context(), cookie.Value)
	if err != nil || member == nil || !member.Active {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	// KVKK Soft Delete: deactivate member account and revoke all sessions
	if err := h.Service.Members.SetActive(r.Context(), member.ID, false); err != nil {
		http.Error(w, "failed to delete account", http.StatusInternalServerError)
		return
	}
	_ = h.Service.RevokeAllSessions(r.Context(), member.ID)
	http.SetCookie(w, &http.Cookie{Name: members.SessionCookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(0, 0), HttpOnly: true, Secure: h.Secure, SameSite: http.SameSiteLaxMode}) // #nosec G124
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true, "message": "Your account was deleted."})
}

func (h Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(members.SessionCookieName)
	if err != nil || cookie.Value == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sess, member, err := h.Service.ValidateSession(r.Context(), cookie.Value)
	if err != nil || member == nil || !member.Active {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.Limiter != nil && !security.AllowForRequest(h.Limiter, r.Context(), "member-chpw:"+requestmeta.ClientIP(r)+":"+member.ID) {
		http.Error(w, "Too many attempts. Please try again shortly.", http.StatusTooManyRequests)
		return
	}
	var payload struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if !api.DecodeJSON(w, r, &payload, maxMemberFormBytes) {
			return
		}
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, maxMemberFormBytes)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		payload.CurrentPassword = r.FormValue("current_password")
		payload.NewPassword = r.FormValue("new_password")
	}
	if err := h.Service.ChangeOwnPassword(r.Context(), member.ID, sess.ID, payload.CurrentPassword, payload.NewPassword); err != nil {
		if errors.Is(err, members.ErrInvalidCredentials) {
			http.Error(w, "Your current password is incorrect.", http.StatusUnauthorized)
			return
		}
		if errors.Is(err, auth.ErrWeakPassword) {
			http.Error(w, "The new password must be at least 6 characters.", http.StatusBadRequest)
			return
		}
		http.Error(w, "The password could not be updated.", http.StatusInternalServerError)
		return
	}
	if h.Mobile != nil {
		_ = h.Mobile.RevokeAll(r.Context(), member.ID)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"updated": true, "message": "Your password was updated."})
}

func (h Handler) changeEmail(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(members.SessionCookieName)
	if err != nil || cookie.Value == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	_, member, err := h.Service.ValidateSession(r.Context(), cookie.Value)
	if err != nil || member == nil || !member.Active {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.Limiter != nil && !security.AllowForRequest(h.Limiter, r.Context(), "member-chem:"+requestmeta.ClientIP(r)+":"+member.ID) {
		http.Error(w, "Too many attempts. Please try again shortly.", http.StatusTooManyRequests)
		return
	}
	var payload struct {
		CurrentPassword string `json:"current_password"`
		NewEmail        string `json:"new_email"`
	}
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if !api.DecodeJSON(w, r, &payload, maxMemberFormBytes) {
			return
		}
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, maxMemberFormBytes)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		payload.CurrentPassword = r.FormValue("current_password")
		payload.NewEmail = r.FormValue("new_email")
	}
	if err := h.Service.ChangeOwnEmail(r.Context(), member.ID, payload.CurrentPassword, payload.NewEmail); err != nil {
		if errors.Is(err, members.ErrInvalidCredentials) {
			http.Error(w, "Your current password is incorrect.", http.StatusUnauthorized)
			return
		}
		if errors.Is(err, members.ErrMemberExists) {
			http.Error(w, "This email address is used by another account.", http.StatusConflict)
			return
		}
		if errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, "Enter a valid email address.", http.StatusBadRequest)
			return
		}
		http.Error(w, "The email address could not be updated.", http.StatusInternalServerError)
		return
	}
	updated, _ := h.Service.Members.GetByID(r.Context(), member.ID)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"updated": true, "member": updated})
}

func (h Handler) me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(members.SessionCookieName)
	if err != nil || cookie.Value == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	_, m, err := h.Service.ValidateSession(r.Context(), cookie.Value)
	if err != nil {
		http.Error(w, "unauthorized", 401)
		return
	}
	_ = json.NewEncoder(w).Encode(m)
}

func (h Handler) verifyMemberEmail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if h.DB == nil {
		http.Error(w, "Verification is not available right now.", http.StatusServiceUnavailable)
		return
	}
	token := r.URL.Query().Get("token")
	err := h.VerificationTx.Within(r.Context(), func(tx database.Executor) error {
		return h.Verification.Verify(r.Context(), tx, token)
	})
	if err != nil {
		http.Error(w, "The verification link is invalid or has expired. Contact support for a new link.", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/member/login?email_verified=1", http.StatusSeeOther)
}
