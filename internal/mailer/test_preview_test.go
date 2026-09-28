package mailer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

type previewSender struct{ message Message }

func (s *previewSender) Send(_ context.Context, m Message) error { s.message = m; return nil }
func TestTestPreviewSendsTextAndHTMLToAdmin(t *testing.T) {
	sender := &previewSender{}
	handler := TestPreviewHandler{Sender: sender}
	ctx := auth.WithUser(context.Background(), &auth.User{ID: "a", Role: auth.RoleAdmin, Active: true, Email: "admin@example.test"})
	req := httptest.NewRequest("POST", "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || sender.message.To != "admin@example.test" || sender.message.HTMLBody == "" || !strings.Contains(sender.message.Body, "123456") {
		t.Fatalf("response=%d message=%+v", rec.Code, sender.message)
	}
}

func TestTestPreviewRejectsEditor(t *testing.T) {
	handler := TestPreviewHandler{Sender: &previewSender{}}
	ctx := auth.WithUser(context.Background(), &auth.User{ID: "e", Role: auth.RoleEditor, Active: true, Email: "editor@example.test"})
	req := httptest.NewRequest("POST", "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestTestPreviewRendersAdminPage(t *testing.T) {
	handler := TestPreviewHandler{Sender: &previewSender{}}
	ctx := auth.WithUser(context.Background(), &auth.User{ID: "a", Role: auth.RoleAdmin, Active: true, Email: "admin@example.test"})
	req := httptest.NewRequest("GET", "/admin/mailer/test-preview", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Send test email to my inbox") || !strings.Contains(rec.Body.String(), `hx-post="/admin/api/mailer/test-preview"`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
