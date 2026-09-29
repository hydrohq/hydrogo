package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

type EmailVerificationNotifier interface {
	SendEmailVerification(context.Context, string, string) error
}
type EmailVerificationNotifierFunc func(context.Context, string, string) error

func (f EmailVerificationNotifierFunc) SendEmailVerification(ctx context.Context, email, link string) error {
	return f(ctx, email, link)
}

type EmailVerificationHandler struct {
	Service      EmailVerificationService
	Transactions database.Transactor
	Notifier     EmailVerificationNotifier
	PublicOrigin string
	Limiter      security.RateLimiter
	AdminUsers   interface {
		GetByID(context.Context, string) (*User, error)
	}
	Audit AuditRecorder
}

func (h EmailVerificationHandler) Routes() []router.Route {
	return []router.Route{
		{Pattern: "POST /auth/me/email/verification", Access: router.Protected, Handler: http.HandlerFunc(h.request)},
		{Pattern: "GET /auth/email/verify", Access: router.Public, Handler: http.HandlerFunc(h.verify)},
		{Pattern: "POST /auth/users/{id}/email/verification/resend", Access: router.Protected, Handler: http.HandlerFunc(h.adminResend)},
	}
}

func (h EmailVerificationHandler) adminResend(w http.ResponseWriter, r *http.Request) {
	admin := UserFromContext(r.Context())
	if admin == nil || !admin.Active || admin.Role != RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.AdminUsers == nil || h.Notifier == nil || guards.IsNil(h.Transactions) || strings.TrimRight(h.PublicOrigin, "/") == "" {
		http.Error(w, "verification unavailable", http.StatusServiceUnavailable)
		return
	}
	targetID := strings.TrimSpace(r.PathValue("id"))
	if targetID == "" || len(targetID) > 128 {
		http.Error(w, "verification unavailable", http.StatusNotFound)
		return
	}
	if h.Limiter != nil && !security.AllowForRequest(h.Limiter, r.Context(), "admin-resend:"+admin.ID+":"+targetID) {
		security.WriteRateLimitedResponse(w, "verification message temporarily unavailable", time.Hour)
		return
	}
	target, err := h.AdminUsers.GetByID(r.Context(), targetID)
	if err != nil || target == nil {
		http.Error(w, "verification unavailable", http.StatusNotFound)
		return
	}
	if target.EmailVerifiedAt != nil {
		http.Redirect(w, r, "/admin/users?email_verified=1", http.StatusSeeOther)
		return
	}
	var email, token string
	err = h.Transactions.Within(r.Context(), func(tx database.Executor) error {
		var issueErr error
		email, token, issueErr = h.Service.Issue(r.Context(), tx, target, 2*time.Hour)
		return issueErr
	})
	if err != nil || token == "" {
		http.Error(w, "verification message could not be sent", http.StatusServiceUnavailable)
		return
	}
	link := strings.TrimRight(h.PublicOrigin, "/") + "/auth/email/verify?token=" + url.QueryEscape(token)
	if err := h.Notifier.SendEmailVerification(r.Context(), email, link); err != nil {
		http.Error(w, "verification message could not be sent", http.StatusServiceUnavailable)
		return
	}
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), "auth.email_verification.resent", target.ID)
	}
	http.Redirect(w, r, "/admin/users?verification_sent=1", http.StatusSeeOther)
}
func (h EmailVerificationHandler) request(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if user.EmailVerifiedAt != nil {
		http.Redirect(w, r, "/admin/mailer/test-preview?verified=1", http.StatusSeeOther)
		return
	}
	if h.Limiter != nil && !security.AllowForRequest(h.Limiter, r.Context(), user.ID) {
		security.WriteRateLimitedResponse(w, "email verification is temporarily unavailable", time.Hour)
		return
	}
	if h.Notifier == nil || strings.TrimRight(h.PublicOrigin, "/") == "" {
		http.Error(w, "email verification unavailable", http.StatusServiceUnavailable)
		return
	}
	var email, token string
	err := h.Transactions.Within(r.Context(), func(tx database.Executor) error {
		var issueErr error
		email, token, issueErr = h.Service.Issue(r.Context(), tx, user, 2*time.Hour)
		return issueErr
	})
	if err != nil {
		http.Error(w, "verification message could not be sent", http.StatusServiceUnavailable)
		return
	}
	if token == "" {
		http.Redirect(w, r, "/admin/mailer/test-preview?verified=1", http.StatusSeeOther)
		return
	}
	link := strings.TrimRight(h.PublicOrigin, "/") + "/auth/email/verify?token=" + url.QueryEscape(token)
	if err := h.Notifier.SendEmailVerification(r.Context(), email, link); err != nil {
		http.Error(w, "verification message could not be sent", http.StatusServiceUnavailable)
		return
	}
	http.Redirect(w, r, "/admin/mailer/test-preview?verification_sent=1", http.StatusSeeOther)
}
func (h EmailVerificationHandler) verify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	token := r.URL.Query().Get("token")
	if len(token) < 40 || len(token) > 128 {
		http.Error(w, "verification link is invalid or expired", http.StatusBadRequest)
		return
	}
	err := h.Transactions.Within(r.Context(), func(tx database.Executor) error { return h.Service.Verify(r.Context(), tx, token) })
	if err != nil {
		http.Error(w, "verification link is invalid or expired", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/login?email_verified=1", http.StatusSeeOther)
}
