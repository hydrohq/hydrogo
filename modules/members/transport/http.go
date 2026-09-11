package transport

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

type Handler struct {
	Service members.Service
	Secure  bool
	Limiter interface{ Allow(string) bool }
}

func (h Handler) Routes() []router.Route {
	return []router.Route{
		{Pattern: "POST /member/register", Access: router.Public, Handler: http.HandlerFunc(h.register)},
		{Pattern: "POST /member/login", Access: router.Public, Handler: http.HandlerFunc(h.login)},
		{Pattern: "POST /member/logout", Access: router.Public, Handler: http.HandlerFunc(h.logout)},
		{Pattern: "GET /member/me", Access: router.Public, Handler: http.HandlerFunc(h.me)},

		// Admin routes (Protected, RoleAdmin checked)
		{Pattern: "GET /admin/members", Access: router.Protected, Handler: http.HandlerFunc(h.adminList)},
		{Pattern: "GET /admin/members/{id}", Access: router.Protected, Handler: http.HandlerFunc(h.adminDetail)},
		{Pattern: "POST /admin/members/{id}", Access: router.Protected, Handler: http.HandlerFunc(h.adminSetActive)},
		{Pattern: "POST /admin/members/{id}/active", Access: router.Protected, Handler: http.HandlerFunc(h.adminSetActive)},
		{Pattern: "POST /admin/members/{id}/verify-email", Access: router.Protected, Handler: http.HandlerFunc(h.adminSetEmailVerified)},
		{Pattern: "POST /admin/members/{id}/sessions/{session_id}/revoke", Access: router.Protected, Handler: http.HandlerFunc(h.adminRevokeSession)},
		{Pattern: "POST /admin/members/{id}/sessions/revoke-all", Access: router.Protected, Handler: http.HandlerFunc(h.adminRevokeAllSessions)},
	}
}

func (h Handler) register(w http.ResponseWriter, r *http.Request) {
	if h.Limiter != nil && !h.Limiter.Allow("member-register:"+r.RemoteAddr) {
		http.Error(w, "too many requests", 429)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	m, err := h.Service.Register(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		if errors.Is(err, members.ErrMemberExists) {
			http.Error(w, "email already registered", 409)
			return
		}
		http.Error(w, "registration failed", 400)
		return
	}
	h.issueSession(w, r, m)
}

func (h Handler) login(w http.ResponseWriter, r *http.Request) {
	if h.Limiter != nil && !h.Limiter.Allow("member-login:"+r.RemoteAddr+":"+auth.NormalizeEmail(r.FormValue("email"))) {
		http.Error(w, "too many requests", 429)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	m, err := h.Service.Authenticate(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		http.Error(w, "invalid credentials", 401)
		return
	}
	h.issueSession(w, r, m)
}

func (h Handler) issueSession(w http.ResponseWriter, r *http.Request, m *members.Member) {
	token, s, err := h.Service.CreateSession(r.Context(), m.ID, r.UserAgent(), r.RemoteAddr, 7*24*time.Hour)
	if err != nil {
		http.Error(w, "authentication unavailable", 503)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: members.SessionCookieName, Value: token, Path: "/", Expires: s.ExpiresAt, HttpOnly: true, Secure: h.Secure, SameSite: http.SameSiteLaxMode})
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{"authenticated": true, "member": map[string]any{"id": m.ID, "email": m.Email}})
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
	if u == nil || !u.Active || u.Role != auth.RoleAdmin {
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
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = "all"
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
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
	if err := ui.AdminStart(w, "Üyeler", "", security.NonceFromContext(r.Context()), "members"); err != nil {
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
			lastLoginText := `<span style="color:var(--muted);font-size:0.8rem;">—</span>`
			if member.LastLoginAt != nil {
				lastLoginText = fmt.Sprintf(`<span style="font-size:0.84rem;font-family:var(--font-mono);">%s</span>`, member.LastLoginAt.Format("2006-01-02 15:04"))
			}
			fmt.Fprintf(&body, `<tr><td><strong><a href="/admin/members/%s" style="color:var(--ink);text-decoration:none;">%s</a></strong></td><td><span class="%s">%s</span></td><td>%s</td><td>%s</td><td style="font-family:var(--font-mono);font-size:0.84rem;">%s</td><td><div style="display:inline-flex;gap:6px;"><a href="/admin/members/%s" class="button button-quiet" style="min-height:30px;padding:4px 10px;font-size:0.78rem;">Detay</a></div></td></tr>`,
				template.HTMLEscapeString(member.ID),
				template.HTMLEscapeString(member.Email),
				statusClass,
				statusText,
				verifyBadge,
				lastLoginText,
				member.CreatedAt.Format("2006-01-02"),
				template.HTMLEscapeString(member.ID),
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

	sessions, _ := h.Service.Sessions.ListByMemberID(r.Context(), id)
	var auditEvents []members.AuditEvent
	if h.Service.Audit != nil {
		auditEvents, _ = h.Service.Audit.ListByTargetID(r.Context(), id, 20)
	}

	csrf := security.CSRFTokenFromContext(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStart(w, "Üye Detayı", "", security.NonceFromContext(r.Context()), "members"); err != nil {
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
	ref := r.Header.Get("Referer")
	if ref != "" {
		http.Redirect(w, r, ref, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/members/"+id, http.StatusSeeOther)
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
	http.Redirect(w, r, "/admin/members/"+id, http.StatusSeeOther)
}

func (h Handler) adminRevokeSession(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	memberID := r.PathValue("id")
	sessionID := r.PathValue("session_id")
	if err := h.Service.RevokeSessionByID(r.Context(), sessionID); err != nil {
		http.Error(w, "session revoke failed", 400)
		return
	}
	http.Redirect(w, r, "/admin/members/"+memberID, http.StatusSeeOther)
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
	http.Redirect(w, r, "/admin/members/"+memberID, http.StatusSeeOther)
}
