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
	"github.com/ardaozyildirim/hydrogo/internal/contracts/customerorders"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

type Handler struct {
	Service       members.Service
	Mobile        *members.MobileAuthService
	Secure        bool
	Limiter       interface{ Allow(string) bool }
	Audit         auth.AuditRecorder
	Orders        customerorders.Reader
	DB            database.Executor
	DefaultLocale string
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
	return []router.Route{
		// Consumer/mobile authentication is intentionally separate from the
		// browser-session and staff mobile surfaces. These routes authenticate
		// with member bearer tokens and never read or write browser cookies.
		{Pattern: "POST /api/v1/mobile/member/auth/login", Access: router.Public, Handler: http.HandlerFunc(h.mobileLogin)},
		{Pattern: "POST /api/v1/mobile/member/auth/refresh", Access: router.Public, Handler: http.HandlerFunc(h.mobileRefresh)},
		{Pattern: "POST /api/v1/mobile/member/auth/revoke", Access: router.Public, Handler: http.HandlerFunc(h.mobileRevoke)},
		{Pattern: "POST /api/v1/mobile/member/auth/revoke-all", Access: router.Public, Handler: http.HandlerFunc(h.mobileRevokeAll)},
		{Pattern: "GET /api/v1/mobile/member/orders", Access: router.Public, Handler: http.HandlerFunc(h.mobileOrders)},
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
		{Pattern: "GET /admin/members/{id}", Access: router.Protected, Handler: http.HandlerFunc(h.adminDetail)},
		{Pattern: "POST /admin/members/{id}", Access: router.Protected, Handler: http.HandlerFunc(h.adminSetActive)},
		{Pattern: "POST /admin/members/{id}/active", Access: router.Protected, Handler: http.HandlerFunc(h.adminSetActive)},
		{Pattern: "POST /admin/members/{id}/verify-email", Access: router.Protected, Handler: http.HandlerFunc(h.adminSetEmailVerified)},
		{Pattern: "POST /admin/members/{id}/sessions/{session_id}/revoke", Access: router.Protected, Handler: http.HandlerFunc(h.adminRevokeSession)},
		{Pattern: "POST /admin/members/{id}/sessions/revoke-all", Access: router.Protected, Handler: http.HandlerFunc(h.adminRevokeAllSessions)},
	}
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
		writeMemberMobileError(w, http.StatusServiceUnavailable, "ORDERS_UNAVAILABLE", "Orders are temporarily unavailable.")
		return
	}
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			writeMemberMobileError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid order limit.")
			return
		}
		limit = parsed
	}
	orders, err := h.Orders.ListOrdersByUserID(r.Context(), h.DB, memberID, limit)
	if err != nil {
		writeMemberMobileError(w, http.StatusServiceUnavailable, "ORDERS_UNAVAILABLE", "Orders are temporarily unavailable.")
		return
	}
	writeMemberMobileJSON(w, http.StatusOK, map[string]any{"orders": orders, "limit": limit})
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

func (h Handler) writeMemberAuthPage(w http.ResponseWriter, r *http.Request, register bool) {
	w = sanitizeMemberResponse(w)
	returnTo := security.SafeReturnPath(r.URL.Query().Get("return_to"), "/member/account")
	title, heading, endpoint, submit, prompt, linkText, linkHref := "Giriş yap", "Hesabınıza giriş yapın", "/member/login", "Giriş yap", "Hesabınız yok mu?", "Üye olun", "/member/register"
	if register {
		title, heading, endpoint, submit, prompt, linkText, linkHref = "Üye ol", "Yeni hesabınızı oluşturun", "/member/register", "Üye ol", "Zaten hesabınız var mı?", "Giriş yapın", "/member/login"
	}
	if returnTo != "/member/account" {
		linkHref += "?return_to=" + url.QueryEscape(returnTo)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	nonce := template.HTMLEscapeString(security.NonceFromContext(r.Context()))
	csrf := template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context()))
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>%s · HydroGo</title><link rel="stylesheet" href="/theme/static/theme.css?v=d986ae3d"></head><body class="theme-corporate-body"><main data-testid="member-auth" class="member-auth-page"><a class="member-auth-brand" href="/"><span class="member-auth-brand-icon"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg></span>HYDRO<b>GO</b></a><section class="member-auth-card"><div class="section-badge"><span class="badge-dot"></span> ÜYELİK MERKEZİ</div><h1>%s</h1><p class="member-auth-lead">Siparişlerinizi ve tercihlerinizi tek bir güvenli hesaptan yönetin.</p><form class="member-auth-form" method="post" data-member-form action="%s"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="return_to" value="%s"><label class="auth-field"><span class="field-label">E-posta Adresi</span><div class="field-input-wrap"><svg class="field-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"></path><polyline points="22,6 12,13 2,6"></polyline></svg><input name="email" type="email" autocomplete="email" required placeholder="ornek@site.com"></div></label><label class="auth-field"><span class="field-label">Şifre</span><div class="field-input-wrap"><svg class="field-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg><input name="password" type="password" autocomplete="%s" minlength="8" required placeholder="En az 8 karakter"></div></label><button class="btn-primary auth-submit-btn" type="submit"><span>%s</span><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"></line><polyline points="12 5 19 12 12 19"></polyline></svg></button><div class="member-auth-status" data-member-status role="alert"></div></form><div class="auth-trust-badge"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg><span>Güvenli bağlantı ve KVKK uyumu</span></div><p class="member-auth-switch">%s <a href="%s" class="auth-back-link">%s →</a></p></section></main><script nonce="%s">(function(){var form=document.querySelector('[data-member-form]'),status=document.querySelector('[data-member-status]');function showMsg(type,text){status.className='member-auth-status status-'+type;status.textContent=text;status.style.display='block';}form.addEventListener('submit',async function(event){event.preventDefault();var button=form.querySelector('button');var originalContent=button.innerHTML;button.disabled=true;button.innerHTML='<span>İşleniyor…</span>';status.style.display='none';status.textContent='';try{var csrfInput=form.querySelector('input[name="_csrf"]');var csrfVal=csrfInput?csrfInput.value:'';var formData=new FormData(form);var body=new URLSearchParams(formData);var response=await fetch(form.action,{method:'POST',headers:{'Content-Type':'application/x-www-form-urlencoded','X-CSRF-Token':csrfVal,'Accept':'application/json'},body:body});var rawText=await response.text();var data=null;try{data=JSON.parse(rawText);}catch(e){}if(!response.ok){var errMsg=(data&&data.error)?data.error:rawText;var lower=(errMsg||'').toLowerCase();if(lower.indexOf('already registered')!==-1||response.status===409){showMsg('error','Bu e-posta adresi zaten kayıtlı. Giriş yapmayı deneyin.');}else if(lower.indexOf('invalid credentials')!==-1||response.status===401){showMsg('error','E-posta veya şifre hatalı. Lütfen bilgilerinizi kontrol edin.');}else if(lower.indexOf('too many')!==-1||response.status===429){showMsg('error','Çok fazla deneme yapıldı. Lütfen biraz bekleyip tekrar deneyin.');}else if(lower.indexOf('csrf')!==-1||response.status===403){showMsg('error','Güvenlik doğrulaması başarısız. Lütfen sayfayı yenileyin.');}else{showMsg('error',errMsg||'İşlem tamamlanamadı. Bilgilerinizi kontrol edin.');}button.disabled=false;button.innerHTML=originalContent;return;}if(data&&data.member){try{localStorage.setItem('hydro_member_state',JSON.stringify(data.member));if(typeof BroadcastChannel!=='undefined'){new BroadcastChannel('hydro_member_auth').postMessage({type:'LOGIN',member:data.member});}}catch(e){}}var target=(data&&data.redirect)?data.redirect:'%s';showMsg('success','İşlem başarılı! Yönlendiriliyorsunuz…');setTimeout(function(){window.location.href=target;},200);}catch(err){showMsg('error','Bağlantı hatası: Sunucuya ulaşılamadı ('+err.message+').');button.disabled=false;button.innerHTML=originalContent;}});})();</script></body></html>`, title, heading, endpoint, csrf, template.HTMLEscapeString(returnTo), map[bool]string{true: "new-password", false: "current-password"}[register], submit, prompt, linkHref, linkText, nonce, template.HTMLEscapeString(returnTo))
}

func (h Handler) accountPage(w http.ResponseWriter, r *http.Request) {
	w = sanitizeMemberResponseWithLocale(w, h.defaultLocale())
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	nonce := template.HTMLEscapeString(security.NonceFromContext(r.Context()))
	csrf := template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context()))

	initial := "U"
	if len(member.Email) > 0 {
		initial = strings.ToUpper(string([]rune(member.Email)[0]))
	}

	verifiedBadge := `<span class="member-badge member-badge-muted">Doğrulanmamış E-posta</span>`
	if member.EmailVerifiedAt != nil {
		verifiedBadge = `<span class="member-badge member-badge-success">✓ E-posta Doğrulandı</span>`
	}

	regDate := member.CreatedAt.Format("02.01.2006")
	if member.CreatedAt.IsZero() {
		regDate = time.Now().Format("02.01.2006")
	}

	var sessionRows strings.Builder
	for _, s := range sessions {
		isCurrent := ""
		if currentSession != nil && s.ID == currentSession.ID {
			isCurrent = `<span class="member-session-current">Mevcut Oturum</span>`
		}
		ua := s.UserAgent
		if len(ua) > 42 {
			ua = ua[:40] + "…"
		}
		if strings.TrimSpace(ua) == "" {
			ua = "Standart Tarayıcı"
		}
		fmt.Fprintf(&sessionRows, `<tr><td><strong>%s</strong>%s<br><small style="color:var(--theme-muted);font-family:'JetBrains Mono',monospace;font-size:0.78rem;">IP: %s</small></td><td style="font-family:'JetBrains Mono',monospace;font-size:0.8rem;">%s</td></tr>`,
			template.HTMLEscapeString(ua),
			isCurrent,
			template.HTMLEscapeString(s.IPAddress),
			s.LastSeenAt.Format("02.01 15:04"),
		)
	}

	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Hesabım · HydroGo</title><link rel="stylesheet" href="/theme/static/theme.css?v=d986ae3d"></head><body class="theme-corporate-body"><header class="news-header"><div class="header-container"><a class="news-logo" href="/" aria-label="HydroGo Ana Sayfa"><div class="logo-mark"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="12 2 22 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg></div><div class="logo-text"><span>HYDROGO</span></div></a><button class="nav-toggle" type="button" aria-expanded="false" aria-controls="primary-nav" aria-label="Menüyü aç"><span class="hamburger-bar"></span><span class="hamburger-bar"></span><span class="hamburger-bar"></span></button><nav id="primary-nav" class="news-nav" aria-label="Ana navigasyon"><a href="/" class="nav-item">Ana Sayfa</a><a href="/public/pages/tr/showcase/html" class="nav-item">Showcase</a><a href="/public/pages/tr/about/html" class="nav-item">Hakkında</a><a href="/public/news/tr/html" class="nav-item">Haberler</a><a href="/shop" class="nav-item">Mağaza</a><a href="/member/account" class="nav-member-badge is-active"><span class="nav-member-icon">👤</span> %s</a><button class="nav-admin-btn" style="cursor:pointer;" data-member-logout type="button">Çıkış ↗</button></nav></div></header><main class="member-portal"><div class="member-hero"><div class="member-hero-left"><div class="member-avatar">%s</div><div class="member-hero-info"><h1>Hoş Geldiniz</h1><div class="member-hero-email">%s</div><div class="member-badges-row"><span class="member-badge member-badge-success">✓ Aktif Üyelik</span>%s<span class="member-badge member-badge-info">🛡️ Güvenli Oturum</span></div></div></div><div style="display:flex;gap:10px;align-items:center;"><button class="btn-secondary" data-member-logout type="button" style="padding:10px 18px;font-size:0.9rem;">Güvenli Çıkış Yap</button></div></div><div class="member-kpi-grid"><div class="member-kpi-card"><span class="member-kpi-label">HESAP GÜVENLİĞİ</span><div class="member-kpi-value" style="color:#166534;">Aktif &amp; Korumalı</div><span class="member-kpi-sub">Güvenli hesap erişimi</span></div><div class="member-kpi-card"><span class="member-kpi-label">AKTİF CİHAZLAR</span><div class="member-kpi-value">%d Cihaz</div><span class="member-kpi-sub">Yetkilendirilmiş güvenli oturum</span></div><div class="member-kpi-card"><span class="member-kpi-label">KAYIT TARİHİ</span><div class="member-kpi-value" style="font-size:1.15rem;font-family:'JetBrains Mono',monospace;">%s</div><span class="member-kpi-sub">Platform katılım tarihi</span></div></div><div class="member-dashboard-grid"><div class="member-left-col"><div class="member-card"><div class="member-card-header"><h2 class="member-card-title">🛍️ Siparişlerim &amp; Dijital Lisanslarım</h2></div><p class="member-card-desc">HydroGo Commerce üzerinden verdiğiniz tüm siparişler, dijital yazılım lisansları ve e-faturalar bu hesapta toplanır.</p><div class="member-action-list"><a class="member-action-item" href="/shop"><span>📦 Mağaza Kataloğuna Göz At</span><span style="color:var(--theme-primary);">İncele →</span></a><a class="member-action-item" href="/shop/cart"><span>🛒 Alışveriş Sepetim</span><span style="color:var(--theme-primary);">Sepete Git →</span></a></div></div><div class="member-card"><div class="member-card-header"><h2 class="member-card-title">📰 Platform &amp; Keşif</h2></div><p class="member-card-desc">HydroGo platform yenilikleri, kurumsal duyurular ve geliştirici blog yazılarını inceleyin.</p><div class="member-action-list"><a class="member-action-item" href="/public/news/tr/html"><span>📰 Haberler ve Duyurular</span><span style="color:var(--theme-primary);">Oku →</span></a><a class="member-action-item" href="/public/pages/tr/about/html"><span>📖 Mimari Felsefe ve Hakkında</span><span style="color:var(--theme-primary);">Detaylar →</span></a></div></div></div><div class="member-right-col"><div class="member-card"><div class="member-card-header"><h2 class="member-card-title">🔒 Aktif Oturumlar &amp; Cihazlar</h2></div><p class="member-card-desc">Hesabınıza erişim sağlayan yetkilendirilmiş oturumlar:</p><table class="member-sessions-table"><thead><tr><th>Cihaz Bilgisi</th><th>Son Görülme</th></tr></thead><tbody>%s</tbody></table></div><div class="member-card"><div class="member-card-header"><h2 class="member-card-title">💬 Müşteri Desteği</h2></div><p class="member-card-desc">Hesabınız veya siparişleriniz ile ilgili her türlü soru için teknik ekibimizle dilediğiniz an iletişime geçebilirsiniz.</p><div style="font-size:0.88rem;color:var(--theme-muted);">Kurumsal İletişim: <strong style="color:var(--theme-ink);">destek@hydrogo.local</strong></div></div></div></div></main><footer class="news-footer" style="margin-top:40px;"><div class="footer-container"><div class="footer-brand-col"><div class="footer-logo"><div class="logo-mark"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg></div><strong>HYDROGO <span>PLATFORM</span></strong></div><p>Yüksek performanslı Go altyapısı ile kurumsal içerik ve üyelik platformu.</p></div></div></footer><script nonce="%s">document.querySelectorAll('[data-member-logout]').forEach(function(el){el.addEventListener('click',function(){fetch('/member/logout',{method:'POST',headers:{'X-CSRF-Token':'%s'}}).then(function(){try{localStorage.removeItem('hydro_member_state');if(typeof BroadcastChannel!=='undefined'){new BroadcastChannel('hydro_member_auth').postMessage({type:'LOGOUT'});}}catch(e){}window.location.assign('/member/login');});});});</script></body></html>`, template.HTMLEscapeString(member.Email), initial, template.HTMLEscapeString(member.Email), verifiedBadge, len(sessions), regDate, sessionRows.String(), nonce, csrf)
}

func (h Handler) register(w http.ResponseWriter, r *http.Request) {
	if h.Limiter != nil && !h.Limiter.Allow("member-register:"+requestmeta.ClientIP(r)) {
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
		if errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, "invalid credentials", 400)
			return
		}
		// Do not expose repository/provider details in a public response.
		http.Error(w, "registration unavailable", http.StatusBadRequest)
		return
	}
	h.issueSession(w, r, m)
}

func (h Handler) login(w http.ResponseWriter, r *http.Request) {
	if h.Limiter != nil && !h.Limiter.Allow("member-login:"+requestmeta.ClientIP(r)+":"+auth.NormalizeEmail(r.FormValue("email"))) {
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
	token, s, err := h.Service.CreateSession(r.Context(), m.ID, r.UserAgent(), requestmeta.ClientIP(r), 7*24*time.Hour)
	if err != nil {
		http.Error(w, "authentication unavailable", 503)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: members.SessionCookieName, Value: token, Path: "/", Expires: s.ExpiresAt, HttpOnly: true, Secure: h.Secure, SameSite: http.SameSiteLaxMode})
	w.Header().Set("Cache-Control", "no-store")

	returnTo := r.FormValue("return_to")
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
	if !active && h.Mobile != nil {
		if err := h.Mobile.RevokeAll(r.Context(), id); err != nil {
			http.Error(w, "member session update failed", http.StatusServiceUnavailable)
			return
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
