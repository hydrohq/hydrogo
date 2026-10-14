package transport

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/contracts/customerorders"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/push"
	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/internal/xlsx"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/members"
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
	// Theme renders the public member pages; nil uses the built-in views.
	Theme *v1.ThemeBundle
}

// memberResponseWriter removes legacy client-side member state persistence
// from rendered auth pages. Session cookies and /member/me remain the only
// source of truth; member identity must not survive in browser storage.
type memberResponseWriter struct {
	http.ResponseWriter
	locale string
}

func (w memberResponseWriter) Write(payload []byte) (int, error) {
	output := string(payload)
	output = strings.ReplaceAll(output, "if(data&&data.member){try{localStorage.setItem('hydro_member_state',JSON.stringify(data.member));if(typeof BroadcastChannel!=='undefined'){new BroadcastChannel('hydro_member_auth').postMessage({type:'LOGIN',member:data.member});}}catch(e){}}", "")
	output = strings.ReplaceAll(output, "try{localStorage.removeItem('hydro_member_state');if(typeof BroadcastChannel!=='undefined'){new BroadcastChannel('hydro_member_auth').postMessage({type:'LOGOUT'});}}catch(e){}", "")
	output = strings.ReplaceAll(output, "Bağlantı hatası: Sunucuya ulaşılamadı ('+err.message+').", "Bağlantı hatası: Sunucuya ulaşılamadı.")
	locale := strings.TrimSpace(w.locale)
	if locale == "" {
		locale = "tr"
	}
	output = strings.ReplaceAll(output, `<html lang="tr"`, `<html lang="`+template.HTMLEscapeString(locale)+`"`)
	output = strings.ReplaceAll(output, "/public/pages/tr/", "/public/pages/"+template.HTMLEscapeString(locale)+"/")
	output = strings.ReplaceAll(output, "/public/news/tr/", "/public/news/"+template.HTMLEscapeString(locale)+"/")
	// Only extend a policy that is already set. Behind a capturing middleware
	// (custom code injection) the header map starts empty, and writing a
	// hash-only policy there replaced the site-wide CSP and blocked every
	// same-origin stylesheet and script on the page.
	if policy := w.Header().Get("Content-Security-Policy"); policy != "" {
		w.Header().Set("Content-Security-Policy", security.MergeCSP(policy, security.InlineCSPDirectives([]byte(output))))
	}
	return w.ResponseWriter.Write([]byte(output))
}

func sanitizeMemberResponse(w http.ResponseWriter) http.ResponseWriter {
	return memberResponseWriter{ResponseWriter: w, locale: "tr"}
}

func sanitizeMemberResponseWithLocale(w http.ResponseWriter, locale string) http.ResponseWriter {
	return memberResponseWriter{ResponseWriter: w, locale: locale}
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
		{Pattern: "GET /member/assets/member-auth.js", Access: router.Public, Handler: http.HandlerFunc(serveMemberAuthScript)},
		{Pattern: "GET /member/login", Access: router.Public, Handler: http.HandlerFunc(h.loginPage)},
		{Pattern: "GET /member/register", Access: router.Public, Handler: http.HandlerFunc(h.registerPage)},
		{Pattern: "GET /member/account", Access: router.Public, Handler: http.HandlerFunc(h.accountPage)},
		{Pattern: "POST /member/register", Access: router.Public, Handler: http.HandlerFunc(h.register)},
		{Pattern: "POST /member/login", Access: router.Public, Handler: http.HandlerFunc(h.login)},
		{Pattern: "POST /member/logout", Access: router.Public, Handler: http.HandlerFunc(h.logout)},
		{Pattern: "GET /member/me", Access: router.Public, Handler: http.HandlerFunc(h.me)},
		{Pattern: "GET /member/orders", Access: router.Public, Handler: http.HandlerFunc(h.memberOrders)},

		// Admin routes (Protected, RoleAdmin checked)
		{Pattern: "GET /admin/members", Access: router.Protected, Handler: http.HandlerFunc(h.adminList)},
		{Pattern: "GET /admin/members/export.xlsx", Access: router.Protected, Handler: http.HandlerFunc(h.adminExportXLSX)},
		{Pattern: "GET /admin/members/export.csv", Access: router.Protected, Handler: http.HandlerFunc(h.adminExportCSV)},
		{Pattern: "GET /admin/members/{id}", Access: router.Protected, Handler: http.HandlerFunc(h.adminDetail)},
		{Pattern: "POST /admin/members/{id}", Access: router.Protected, Handler: http.HandlerFunc(h.adminSetActive)},
		{Pattern: "POST /admin/members/{id}/active", Access: router.Protected, Handler: http.HandlerFunc(h.adminSetActive)},
		{Pattern: "POST /admin/members/{id}/verify-email", Access: router.Protected, Handler: http.HandlerFunc(h.adminSetEmailVerified)},
		{Pattern: "POST /admin/members/{id}/resend-verification", Access: router.Protected, Handler: http.HandlerFunc(h.adminResendVerification)},
		{Pattern: "GET /member/email/verify", Access: router.Public, Handler: http.HandlerFunc(h.verifyMemberEmail)},
		{Pattern: "POST /admin/members/{id}/sessions/{session_id}/revoke", Access: router.Protected, Handler: http.HandlerFunc(h.adminRevokeSession)},
		{Pattern: "POST /admin/members/{id}/sessions/revoke-all", Access: router.Protected, Handler: http.HandlerFunc(h.adminRevokeAllSessions)},
	}
	if h.Push != nil {
		routes = append(routes,
			router.Route{Pattern: "POST /api/v1/mobile/member/push/subscription", Access: router.Public, Handler: http.HandlerFunc(h.mobilePushSubscribe)},
			router.Route{Pattern: "DELETE /api/v1/mobile/member/push/subscription", Access: router.Public, Handler: http.HandlerFunc(h.mobilePushUnsubscribe)},
		)
	}
	return routes
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

func (h Handler) loginPage(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(members.SessionCookieName); err == nil && cookie.Value != "" {
		if _, _, err := h.Service.ValidateSession(r.Context(), cookie.Value); err == nil {
			returnTo := security.SafeReturnPath(r.URL.Query().Get("return_to"), "/member/account")
			http.Redirect(w, r, returnTo, http.StatusSeeOther)
			return
		}
	}
	h.writeMemberAuthPage(w, r, false)
}

func (h Handler) registerPage(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(members.SessionCookieName); err == nil && cookie.Value != "" {
		if _, _, err := h.Service.ValidateSession(r.Context(), cookie.Value); err == nil {
			returnTo := security.SafeReturnPath(r.URL.Query().Get("return_to"), "/member/account")
			http.Redirect(w, r, returnTo, http.StatusSeeOther)
			return
		}
	}
	h.writeMemberAuthPage(w, r, true)
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
	http.SetCookie(w, &http.Cookie{Name: members.SessionCookieName, Value: token, Path: "/", Expires: s.ExpiresAt, HttpOnly: true, Secure: h.Secure, SameSite: http.SameSiteLaxMode})
	w.Header().Set("Cache-Control", "no-store")

	if returnTo == "" {
		returnTo = r.FormValue("return_to")
	}
	if returnTo == "" {
		returnTo = r.URL.Query().Get("return_to")
	}
	returnTo = security.SafeReturnPath(returnTo, "/member/account")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]any{
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
	http.SetCookie(w, &http.Cookie{Name: members.SessionCookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(0, 0), HttpOnly: true, Secure: h.Secure, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
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
	json.NewEncoder(w).Encode(m)
}

func (h Handler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	u := auth.UserFromContext(r.Context())
	if !auth.Can(u, auth.PermissionMembersManage) {
		http.Error(w, "forbidden", 403)
		return false
	}
	return true
}

func (h Handler) adminList(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 200 {
		q = q[:200]
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	allowedStatus := map[string]bool{"all": true, "active": true, "inactive": true, "verified": true, "unverified": true}
	if !allowedStatus[status] {
		status = "all"
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	if page > 10000 {
		page = 10000
	}
	limit := 20
	offset := (page - 1) * limit

	filter := members.MemberFilter{
		Query:  q,
		Status: status,
		Limit:  limit,
		Offset: offset,
	}

	result, err := h.Service.Members.Search(r.Context(), filter)
	if err != nil {
		http.Error(w, "member list unavailable", 503)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Üyeler", "", "members"); err != nil {
		return
	}

	totalPages := (result.TotalCount + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}

	var body strings.Builder
	body.WriteString(`<section class="studio-card">`)

	// Search & Filter Toolbar
	body.WriteString(`<form class="table-toolbar" method="get" action="/admin/members">`)
	fmt.Fprintf(&body, `<div class="search-wrap"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg><input type="search" name="q" value="%s" placeholder="E-posta ile üye ara…"></div>`, template.HTMLEscapeString(q))
	body.WriteString(`<select name="status">`)
	for _, opt := range []struct{ id, label string }{
		{"all", "Tüm Durumlar"},
		{"active", "Sadece Aktifler"},
		{"inactive", "Sadece Pasifler"},
		{"verified", "E-postası Doğrulanmış"},
		{"unverified", "E-postası Doğrulanmamış"},
	} {
		selected := ""
		if status == opt.id {
			selected = " selected"
		}
		fmt.Fprintf(&body, `<option value="%s"%s>%s</option>`, opt.id, selected, opt.label)
	}
	body.WriteString(`</select>`)
	body.WriteString(`<button type="submit" class="button">Filtrele</button>`)
	if q != "" || status != "all" {
		body.WriteString(`<a href="/admin/members" class="button button-quiet">Temizle</a>`)
	}
	body.WriteString(`</form>`)
	exportURL := "/admin/members/export.xlsx?q=" + url.QueryEscape(q) + "&status=" + url.QueryEscape(status)
	var exportLinks bytes.Buffer
	if err := designsystem.ExportLinks([]designsystem.ExportLink{{Label: "CSV dışa aktar", URL: strings.Replace(exportURL, "/export.xlsx", "/export.csv", 1)}, {Label: "Excel dışa aktar", URL: exportURL}}).Render(r.Context(), &exportLinks); err == nil {
		body.WriteString(exportLinks.String())
	}

	// Table view
	body.WriteString(`<div class="table-wrap"><table><thead><tr><th>E-posta</th><th>Durum</th><th>E-posta Doğrulama</th><th>Son Giriş</th><th>Kayıt Tarihi</th><th>İşlem</th></tr></thead><tbody>`)
	if len(result.Members) == 0 {
		body.WriteString(`<tr><td colspan="6" class="empty">Eşleşen üye kaydı bulunamadı.</td></tr>`)
	} else {
		for _, member := range result.Members {
			statusClass, statusText := "status-badge", "Pasif"
			if member.Active {
				statusClass, statusText = "status-badge status-badge-success", "Aktif"
			}
			verifyBadge := `<span style="color:var(--muted);font-size:0.8rem;">Doğrulanmamış</span>`
			if member.EmailVerifiedAt != nil {
				verifyBadge = fmt.Sprintf(`<span style="color:var(--green);font-weight:600;font-size:0.8rem;" title="%s">✓ Doğrulanmış</span>`, member.EmailVerifiedAt.Format("2006-01-02 15:04"))
			}
			actions := fmt.Sprintf(`<a href="/admin/members/%s" class="button button-quiet" style="min-height:28px;padding:4px 12px;font-size:0.78rem;font-weight:600;display:inline-flex;align-items:center;">Detay</a>`, template.HTMLEscapeString(member.ID))
			if h.VerificationEnabled && member.EmailVerifiedAt == nil {
				var resend bytes.Buffer
				if err := designsystem.EmailVerificationResendForm("/admin/members/"+url.PathEscape(member.ID)+"/resend-verification", security.CSRFTokenFromContext(r.Context())).Render(r.Context(), &resend); err == nil {
					actions += resend.String()
				}
			}
			lastLoginText := `<span style="color:var(--muted);font-size:0.8rem;">—</span>`
			if member.LastLoginAt != nil {
				lastLoginText = fmt.Sprintf(`<span style="font-size:0.84rem;font-family:var(--font-mono);">%s</span>`, member.LastLoginAt.Format("2006-01-02 15:04"))
			}
			fmt.Fprintf(&body, `<tr><td><strong><a href="/admin/members/%s" style="color:var(--ink);text-decoration:none;">%s</a></strong></td><td><span class="%s">%s</span></td><td>%s</td><td>%s</td><td style="font-family:var(--font-mono);font-size:0.84rem;">%s</td><td><div style="display:inline-flex;gap:6px;">%s</div></td></tr>`,
				template.HTMLEscapeString(member.ID),
				template.HTMLEscapeString(member.Email),
				statusClass,
				statusText,
				verifyBadge,
				lastLoginText,
				member.CreatedAt.Format("2006-01-02"),
				actions,
			)
		}
	}
	body.WriteString(`</tbody></table></div>`)

	// Pagination controls
	if totalPages > 1 {
		body.WriteString(`<div class="pagination" style="display:flex;justify-content:space-between;align-items:center;margin-top:20px;padding-top:14px;border-top:1px solid var(--line);font-size:0.86rem;">`)
		fmt.Fprintf(&body, `<span class="muted">Toplam %d üye (Sayfa %d / %d)</span>`, result.TotalCount, page, totalPages)
		body.WriteString(`<div style="display:flex;gap:6px;">`)
		if page > 1 {
			prevURL := fmt.Sprintf("/admin/members?q=%s&status=%s&page=%d", url.QueryEscape(q), url.QueryEscape(status), page-1)
			fmt.Fprintf(&body, `<a href="%s" class="button button-quiet" style="min-height:32px;padding:4px 12px;">← Önceki</a>`, prevURL)
		}
		if page < totalPages {
			nextURL := fmt.Sprintf("/admin/members?q=%s&status=%s&page=%d", url.QueryEscape(q), url.QueryEscape(status), page+1)
			fmt.Fprintf(&body, `<a href="%s" class="button button-quiet" style="min-height:32px;padding:4px 12px;">Sonraki →</a>`, nextURL)
		}
		body.WriteString(`</div></div>`)
	}

	body.WriteString(`</section>`)
	_, _ = w.Write([]byte(body.String()))
	_ = ui.AdminEnd(w)
}

func (h Handler) adminExportXLSX(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	rows, err := h.memberExportRows(r)
	if err != nil {
		writeMemberExportError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="members.xlsx"`)
	if err := xlsx.Write(w, rows); err != nil {
		return
	}
}

func (h Handler) adminExportCSV(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	rows, err := h.memberExportRows(r)
	if err != nil {
		writeMemberExportError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="members.csv"`)
	writer := csv.NewWriter(w)
	for index, row := range rows {
		if index > 0 && len(row) > 0 {
			row[0] = csvSafeMember(row[0])
		}
		if err := writer.Write(row); err != nil {
			return
		}
	}
	writer.Flush()
}

func csvSafeMember(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}

func (h Handler) memberExportRows(r *http.Request) ([][]string, error) {
	const pageSize, maxRows = 500, 20_000
	filter := members.MemberFilter{Query: strings.TrimSpace(r.URL.Query().Get("q")), Status: r.URL.Query().Get("status"), Limit: pageSize}
	first, err := h.Service.Members.Search(r.Context(), filter)
	if err != nil {
		return nil, err
	}
	if first.TotalCount > maxRows {
		return nil, errMemberExportTooLarge
	}
	rows := make([][]string, 1, first.TotalCount+1)
	rows[0] = []string{"Email", "Active", "Email verified at", "Last login at", "Created at"}
	appendMembers := func(values []members.Member) {
		for _, member := range values {
			verified, lastLogin := "", ""
			if member.EmailVerifiedAt != nil {
				verified = member.EmailVerifiedAt.UTC().Format(time.RFC3339)
			}
			if member.LastLoginAt != nil {
				lastLogin = member.LastLoginAt.UTC().Format(time.RFC3339)
			}
			rows = append(rows, []string{member.Email, strconv.FormatBool(member.Active), verified, lastLogin, member.CreatedAt.UTC().Format(time.RFC3339)})
		}
	}
	appendMembers(first.Members)
	for offset := len(first.Members); offset < first.TotalCount; offset += pageSize {
		filter.Offset = offset
		page, err := h.Service.Members.Search(r.Context(), filter)
		if err != nil {
			return nil, err
		}
		appendMembers(page.Members)
		if len(page.Members) == 0 {
			break
		}
	}
	return rows, nil
}

var errMemberExportTooLarge = errors.New("member export exceeds row limit")

func writeMemberExportError(w http.ResponseWriter, err error) {
	if errors.Is(err, errMemberExportTooLarge) {
		http.Error(w, "export is too large; narrow the member filter", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, "member export unavailable", http.StatusServiceUnavailable)
}

func (h Handler) adminDetail(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id := r.PathValue("id")
	member, err := h.Service.Members.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, members.ErrMemberNotFound) {
			http.Error(w, "member not found", 404)
			return
		}
		http.Error(w, "member lookup failed", 500)
		return
	}

	var sessions []members.Session
	if h.Service.Sessions != nil {
		sessions, _ = h.Service.Sessions.ListByMemberID(r.Context(), id)
	}
	var auditEvents []members.AuditEvent
	if h.Service.Audit != nil {
		auditEvents, _ = h.Service.Audit.ListByTargetID(r.Context(), id, 20)
	}

	csrf := security.CSRFTokenFromContext(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Üye Detayı", "", "members"); err != nil {
		return
	}

	var body strings.Builder
	body.WriteString(`<div style="display:flex;flex-direction:column;gap:20px;">`)

	// Top actions & breadcrumbs
	body.WriteString(`<div style="display:flex;justify-content:space-between;align-items:center;"><a href="/admin/members" class="button button-quiet" style="min-height:36px;">← Üye Listesine Dön</a><div style="display:flex;gap:8px;">`)
	if member.Active {
		fmt.Fprintf(&body, `<form method="post" action="/admin/members/%s/active"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="active" value="false"><button type="submit" class="button button-danger" style="min-height:36px;">Hesabı Devre Dışı Bırak</button></form>`, template.HTMLEscapeString(member.ID), template.HTMLEscapeString(csrf))
	} else {
		fmt.Fprintf(&body, `<form method="post" action="/admin/members/%s/active"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="active" value="true"><button type="submit" class="button button-success" style="min-height:36px;">Hesabı Etkinleştir</button></form>`, template.HTMLEscapeString(member.ID), template.HTMLEscapeString(csrf))
	}
	body.WriteString(`</div></div>`)

	// Profile & Stats Card
	body.WriteString(`<section class="studio-card"><div class="card-title-group"><h2>Üye Bilgileri</h2></div><div class="kpi-grid" style="margin-bottom:20px;">`)
	fmt.Fprintf(&body, `<div class="kpi-card"><span class="kpi-label">E-POSTA</span><strong style="font-size:1.15rem;margin-top:6px;display:block;">%s</strong></div>`, template.HTMLEscapeString(member.Email))
	statusText := "Aktif"
	if !member.Active {
		statusText = "Pasif"
	}
	fmt.Fprintf(&body, `<div class="kpi-card"><span class="kpi-label">DURUM</span><strong style="font-size:1.15rem;margin-top:6px;display:block;">%s</strong></div>`, statusText)
	verifyText := "Doğrulanmamış"
	if member.EmailVerifiedAt != nil {
		verifyText = member.EmailVerifiedAt.Format("2006-01-02 15:04")
	}
	fmt.Fprintf(&body, `<div class="kpi-card"><span class="kpi-label">DOĞRULAMA</span><strong style="font-size:1.15rem;margin-top:6px;display:block;">%s</strong></div>`, verifyText)
	lastLoginStr := "Giriş Yok"
	if member.LastLoginAt != nil {
		lastLoginStr = member.LastLoginAt.Format("2006-01-02 15:04")
	}
	fmt.Fprintf(&body, `<div class="kpi-card"><span class="kpi-label">SON GİRİŞ</span><strong style="font-size:1.15rem;margin-top:6px;display:block;">%s</strong></div>`, lastLoginStr)
	body.WriteString(`</div>`)

	// Quick Management Form
	body.WriteString(`<div style="display:flex;gap:12px;padding-top:14px;border-top:1px solid var(--line);align-items:center;flex-wrap:wrap;">`)
	if member.EmailVerifiedAt == nil {
		fmt.Fprintf(&body, `<form method="post" action="/admin/members/%s/verify-email"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="verified" value="true"><button type="submit" class="button" style="min-height:36px;">E-postayı Doğrulandı Olarak İşaretle</button></form>`, template.HTMLEscapeString(member.ID), template.HTMLEscapeString(csrf))
	} else {
		fmt.Fprintf(&body, `<form method="post" action="/admin/members/%s/verify-email"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="verified" value="false"><button type="submit" class="button button-quiet" style="min-height:36px;">Doğrulama Durumunu Kaldır</button></form>`, template.HTMLEscapeString(member.ID), template.HTMLEscapeString(csrf))
	}
	body.WriteString(`</div></section>`)

	// Active Sessions Card
	body.WriteString(`<section class="studio-card"><div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px;"><div class="card-title-group"><h2>Aktif Oturumlar</h2></div>`)
	if len(sessions) > 0 {
		fmt.Fprintf(&body, `<form method="post" action="/admin/members/%s/sessions/revoke-all"><input type="hidden" name="_csrf" value="%s"><button type="submit" class="button button-quiet button-danger" style="min-height:32px;font-size:0.8rem;">Tüm Oturumları Kapat</button></form>`, template.HTMLEscapeString(member.ID), template.HTMLEscapeString(csrf))
	}
	body.WriteString(`</div><div class="table-wrap"><table><thead><tr><th>IP Adresi</th><th>Tarayıcı / Cihaz</th><th>Son Görülme</th><th>Son Geçerlilik</th><th>İşlem</th></tr></thead><tbody>`)
	if len(sessions) == 0 {
		body.WriteString(`<tr><td colspan="5" class="empty">Aktif oturum bulunmuyor.</td></tr>`)
	} else {
		for _, s := range sessions {
			fmt.Fprintf(&body, `<tr><td style="font-family:var(--font-mono);font-size:0.84rem;">%s</td><td style="font-size:0.84rem;max-width:240px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">%s</td><td style="font-family:var(--font-mono);font-size:0.84rem;">%s</td><td style="font-family:var(--font-mono);font-size:0.84rem;">%s</td><td><form method="post" action="/admin/members/%s/sessions/%s/revoke"><input type="hidden" name="_csrf" value="%s"><button type="submit" class="button button-quiet button-danger" style="min-height:28px;padding:2px 8px;font-size:0.75rem;">Sonlandır</button></form></td></tr>`,
				template.HTMLEscapeString(s.IPAddress),
				template.HTMLEscapeString(s.UserAgent),
				s.LastSeenAt.Format("2006-01-02 15:04:05"),
				s.ExpiresAt.Format("2006-01-02 15:04"),
				template.HTMLEscapeString(member.ID),
				template.HTMLEscapeString(s.ID),
				template.HTMLEscapeString(csrf),
			)
		}
	}
	body.WriteString(`</tbody></table></div></section>`)

	// Audit Trail Card
	body.WriteString(`<section class="studio-card"><div class="card-title-group"><h2>Güvenlik ve İşlem Kayıtları (Audit)</h2></div><div class="table-wrap"><table><thead><tr><th>Olay / Aksiyon</th><th>İstek ID</th></tr></thead><tbody>`)
	if len(auditEvents) == 0 {
		body.WriteString(`<tr><td colspan="2" class="empty">Kayıtlı audit olayı bulunmuyor.</td></tr>`)
	} else {
		for _, ev := range auditEvents {
			fmt.Fprintf(&body, `<tr><td><strong>%s</strong></td><td style="font-family:var(--font-mono);font-size:0.82rem;color:var(--muted);">%s</td></tr>`,
				template.HTMLEscapeString(ev.Action),
				template.HTMLEscapeString(ev.RequestID),
			)
		}
	}
	body.WriteString(`</tbody></table></div></section>`)

	body.WriteString(`</div>`)
	_, _ = w.Write([]byte(body.String()))
	_ = ui.AdminEnd(w)
}

func (h Handler) adminSetActive(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	active := strings.EqualFold(r.FormValue("active"), "true") || r.FormValue("active") == "1"
	id := r.PathValue("id")
	if err := h.Service.Members.SetActive(r.Context(), id, active); err != nil {
		http.Error(w, "member update failed", 400)
		return
	}
	if !active {
		if err := h.Service.RevokeAllSessions(r.Context(), id); err != nil {
			http.Error(w, "member session update failed", http.StatusServiceUnavailable)
			return
		}
		if h.Mobile != nil {
			if err := h.Mobile.RevokeAll(r.Context(), id); err != nil {
				http.Error(w, "member session update failed", http.StatusServiceUnavailable)
				return
			}
		}
	}
	ref := r.Header.Get("Referer")
	if ref != "" {
		// Referer is attacker-controlled. Restrict redirects to an in-origin
		// path so a forged header cannot turn this mutation into an open redirect.
		if safeRef := security.SafeReturnPath(ref, ""); safeRef != "" {
			http.Redirect(w, r, safeRef, http.StatusSeeOther)
			return
		}
	}
	if id != "" {
		http.Redirect(w, r, "/admin/members/"+url.PathEscape(id), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/members", http.StatusSeeOther)
}

func (h Handler) adminSetEmailVerified(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	verified := strings.EqualFold(r.FormValue("verified"), "true") || r.FormValue("verified") == "1"
	id := r.PathValue("id")
	if err := h.Service.Members.SetEmailVerified(r.Context(), id, verified); err != nil {
		http.Error(w, "member verification update failed", 400)
		return
	}
	http.Redirect(w, r, "/admin/members/"+url.PathEscape(id), http.StatusSeeOther)
}

func (h Handler) adminResendVerification(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	admin := auth.UserFromContext(r.Context())
	targetID := strings.TrimSpace(r.PathValue("id"))
	if !h.VerificationEnabled || h.VerificationNotifier == nil || strings.TrimSpace(h.PublicOrigin) == "" || h.DB == nil {
		http.Error(w, "verification email is unavailable", http.StatusServiceUnavailable)
		return
	}
	if targetID == "" || len(targetID) > 128 {
		http.Error(w, "member not found", http.StatusNotFound)
		return
	}
	if h.VerificationLimiter != nil && !security.AllowForRequest(h.VerificationLimiter, r.Context(), "member-verification:"+admin.ID+":"+targetID) {
		security.WriteRateLimitedResponse(w, "verification email temporarily unavailable", time.Hour)
		return
	}
	member, err := h.Service.Members.GetByID(r.Context(), targetID)
	if err != nil || member == nil {
		http.Error(w, "member not found", http.StatusNotFound)
		return
	}
	if member.EmailVerifiedAt != nil {
		http.Redirect(w, r, "/admin/members?email_verified=1", http.StatusSeeOther)
		return
	}
	var token string
	err = h.VerificationTx.Within(r.Context(), func(tx database.Executor) error {
		var issueErr error
		token, issueErr = h.Verification.Issue(r.Context(), tx, member, 24*time.Hour)
		return issueErr
	})
	if err != nil || token == "" {
		http.Error(w, "verification email could not be sent", http.StatusServiceUnavailable)
		return
	}
	link := strings.TrimRight(h.PublicOrigin, "/") + "/member/email/verify?token=" + url.QueryEscape(token)
	if err := h.VerificationNotifier.SendEmailVerification(r.Context(), member.Email, link); err != nil {
		http.Error(w, "verification email could not be sent", http.StatusServiceUnavailable)
		return
	}
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), "member.email_verification.resent", member.ID)
	}
	http.Redirect(w, r, "/admin/members?verification_sent=1", http.StatusSeeOther)
}

func (h Handler) verifyMemberEmail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if h.DB == nil {
		http.Error(w, "Doğrulama şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	token := r.URL.Query().Get("token")
	err := h.VerificationTx.Within(r.Context(), func(tx database.Executor) error {
		return h.Verification.Verify(r.Context(), tx, token)
	})
	if err != nil {
		http.Error(w, "Doğrulama bağlantısı geçersiz veya süresi dolmuş. Yeni bağlantı için destek ekibiyle iletişime geçin.", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/member/login?email_verified=1", http.StatusSeeOther)
}

func (h Handler) adminRevokeSession(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	memberID := r.PathValue("id")
	sessionID := r.PathValue("session_id")
	if err := h.Service.RevokeSessionByID(r.Context(), memberID, sessionID); err != nil {
		if errors.Is(err, members.ErrMemberNotFound) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		http.Error(w, "session revoke failed", 400)
		return
	}
	http.Redirect(w, r, "/admin/members/"+url.PathEscape(memberID), http.StatusSeeOther)
}

func (h Handler) adminRevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	memberID := r.PathValue("id")
	if err := h.Service.RevokeAllSessions(r.Context(), memberID); err != nil {
		http.Error(w, "all sessions revoke failed", 400)
		return
	}
	http.Redirect(w, r, "/admin/members/"+url.PathEscape(memberID), http.StatusSeeOther)
}
