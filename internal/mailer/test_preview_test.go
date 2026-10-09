package mailer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

type previewSender struct{ message Message }

func (s *previewSender) Send(_ context.Context, m Message) error { s.message = m; return nil }

type pingPreviewSender struct {
	previewSender
	result PingResult
	err    error
}

func (s *pingPreviewSender) Ping(context.Context) (PingResult, error) { return s.result, s.err }

func TestTestPreviewSendsTextAndHTMLToAdmin(t *testing.T) {
	sender := &previewSender{}
	route := protectedMailerMutation(t, "POST /admin/api/mailer/test-preview", TestPreviewHandler{Sender: sender})
	ctx := auth.WithUser(context.Background(), &auth.User{ID: "a", Role: auth.RoleAdmin, Active: true, Email: "admin@example.test", EmailVerifiedAt: func() *time.Time { now := time.Now(); return &now }()})
	req := httptest.NewRequest(http.MethodPost, "/admin/api/mailer/test-preview", nil)
	req = withMailerCSRF(req, "valid-token").WithContext(ctx)
	rec := httptest.NewRecorder()
	route.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || sender.message.To != "admin@example.test" || sender.message.HTMLBody == "" || !strings.Contains(sender.message.Body, "123456") {
		t.Fatalf("response=%d message=%+v", rec.Code, sender.message)
	}
}

func TestTestPreviewRejectsEditor(t *testing.T) {
	sender := &previewSender{}
	route := protectedMailerMutation(t, "POST /admin/api/mailer/test-preview", TestPreviewHandler{Sender: sender})
	for _, tc := range []struct {
		name  string
		user  *auth.User
		token string
	}{
		{name: "admin with invalid csrf", user: &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}, token: "wrong-token"},
		{name: "editor with valid csrf", user: &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, token: "valid-token"},
		{name: "viewer with valid csrf", user: &auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}, token: "valid-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := auth.WithUser(context.Background(), tc.user)
			req := httptest.NewRequest(http.MethodPost, "/admin/api/mailer/test-preview", nil)
			req = withMailerCSRF(req, tc.token).WithContext(ctx)
			rec := httptest.NewRecorder()
			route.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden || sender.message.To != "" {
				t.Fatalf("status=%d sent=%+v", rec.Code, sender.message)
			}
		})
	}
}

func TestTestPreviewRendersAdminPage(t *testing.T) {
	handler := TestPreviewHandler{Sender: &pingPreviewSender{}}
	ctx := auth.WithUser(context.Background(), &auth.User{ID: "a", Role: auth.RoleAdmin, Active: true, Email: "admin@example.test", EmailVerifiedAt: func() *time.Time { now := time.Now(); return &now }()})
	req := httptest.NewRequest("GET", "/admin/mailer/test-preview", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Send test email to my inbox") || !strings.Contains(rec.Body.String(), "Check mail connection") || !strings.Contains(rec.Body.String(), `hx-post="/admin/api/mailer/test-preview"`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTestPreviewSMTPPingDoesNotRequireVerifiedEmail(t *testing.T) {
	sender := &pingPreviewSender{result: PingResult{Stage: "complete", TLSVersion: "TLS 1.3", Authenticated: true}}
	route := protectedMailerMutation(t, "POST /admin/api/mailer/test-preview", TestPreviewHandler{Sender: sender})
	ctx := auth.WithUser(context.Background(), &auth.User{ID: "a", Role: auth.RoleAdmin, Active: true, Email: "admin@example.test"})
	request := httptest.NewRequest(http.MethodPost, "/admin/api/mailer/test-preview", strings.NewReader("_action=ping"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	request = withMailerCSRF(request, "valid-token").WithContext(ctx)
	response := httptest.NewRecorder()
	route.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "SMTP connection and authentication verified") || !strings.Contains(response.Body.String(), "TLS 1.3") {
		t.Fatalf("ping response=%d body=%s", response.Code, response.Body.String())
	}
}

func TestTestPreviewRequiresVerifiedAdminAddress(t *testing.T) {
	sender := &previewSender{}
	route := protectedMailerMutation(t, "POST /admin/api/mailer/test-preview", TestPreviewHandler{Sender: sender})
	ctx := auth.WithUser(context.Background(), &auth.User{ID: "a", Role: auth.RoleAdmin, Active: true, Email: "admin@example.test"})
	req := httptest.NewRequest(http.MethodPost, "/admin/api/mailer/test-preview", nil)
	req = withMailerCSRF(req, "valid-token").WithContext(ctx)
	rec := httptest.NewRecorder()
	route.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || sender.message.To != "" {
		t.Fatalf("unverified preview status=%d message=%+v", rec.Code, sender.message)
	}
	page := httptest.NewRecorder()
	handler := TestPreviewHandler{Sender: sender}
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/admin/mailer/test-preview", nil).WithContext(ctx))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Send confirmation link") || strings.Contains(page.Body.String(), `hx-post="/admin/api/mailer/test-preview"`) {
		t.Fatalf("unverified preview page status=%d body=%s", page.Code, page.Body.String())
	}
}

func TestEmailVerificationNotifierIncludesOneUseLinkAsHTMLAndText(t *testing.T) {
	sender := &previewSender{}
	link := "https://hydro.example.test/auth/email/verify?token=opaque"
	if err := (NotificationSender{Sender: sender}).SendEmailVerification(context.Background(), "admin@example.test", link); err != nil {
		t.Fatal(err)
	}
	if sender.message.To != "admin@example.test" || !strings.Contains(sender.message.Body, link) || !strings.Contains(sender.message.HTMLBody, link) {
		t.Fatalf("verification email=%+v", sender.message)
	}
}
