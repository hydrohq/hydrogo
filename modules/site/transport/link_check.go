package transport

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

func (h Handler) linkCheckPage(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) {
		http.Error(w, "Bu sayfaya erişim yetkiniz yok.", http.StatusForbidden)
		return
	}
	jobID := r.URL.Query().Get("job")
	var findings []designsystem.LinkCheckFindingView
	running, scanned, message := false, false, ""
	if jobID != "" {
		job, ok := h.LinkScanManager.Get(linkScanTenantKey(r), jobID)
		if !ok {
			message = "Denetim bulunamadı veya süresi doldu."
		} else if job.Failed {
			scanned = true
			message = "Denetim tamamlanamadı."
		} else {
			running, scanned = job.Running, !job.Running
			for _, finding := range job.Findings {
				status := strconv.Itoa(finding.Status)
				if finding.Status == 0 {
					status = "Unavailable"
				}
				findings = append(findings, designsystem.LinkCheckFindingView{SourcePage: finding.SourcePage, SourceTitle: finding.SourceTitle, URL: finding.URL, Status: status, AnchorText: finding.AnchorText, Suggestion: finding.Suggestion})
			}
		}
	}
	if ui.AdminStartForRequest(w, r, "Bağlantı denetimi", "Yayınlanmış sayfa, bağlantı ve medya adreslerini kontrol edin", "developer") != nil {
		return
	}
	_, _ = io.WriteString(w, ui.RenderDeveloperTabs("link-check"))
	_ = designsystem.LinkCheckAudit(security.CSRFTokenFromContext(r.Context()), jobID, findings, running, scanned, message).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) runLinkCheck(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !h.admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.LinkScanManager == nil || strings.TrimSpace(h.SitemapOrigin) == "" {
		http.Error(w, "Bu denetim şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	if user != nil && h.LinkScanLimiter != nil && !security.AllowForRequest(h.LinkScanLimiter, r.Context(), user.ID) {
		security.WriteRateLimitedResponse(w, "Bağlantı denetimi kısa süre önce çalıştırıldı.", time.Minute)
		return
	}
	job, err := h.LinkScanManager.Start(linkScanTenantKey(r), h.SitemapOrigin)
	if err != nil {
		http.Error(w, "Bağlantı denetimi başlatılamadı. Devam eden denetimin bitmesini bekleyin.", http.StatusConflict)
		return
	}
	http.Redirect(w, r, "/admin/site/link-check?job="+url.QueryEscape(job.ID), http.StatusSeeOther)
}

func linkScanTenantKey(r *http.Request) string {
	scope, err := tenant.FromContext(r.Context())
	if err == nil {
		return scope.TenantID
	}
	return "default"
}
