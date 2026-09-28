package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
)

type TestPreviewHandler struct{ Sender Sender }

func (h TestPreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		user := auth.UserFromContext(r.Context())
		if user == nil || !user.Active || user.Role != auth.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := ui.AdminStartForRequest(w, r, "Test email preview", "Mailer", security.NonceFromContext(r.Context()), "settings"); err != nil {
			return
		}
		if err := ui.EmailTestPreviewForm(security.CSRFTokenFromContext(r.Context()), user.EmailVerifiedAt != nil, r.URL.Query().Get("verification_sent") == "1").Render(r.Context(), w); err != nil {
			http.Error(w, "email preview unavailable", http.StatusInternalServerError)
			return
		}
		_ = ui.AdminEnd(w)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin || user.EmailVerifiedAt == nil || !auth.ValidateEmail(user.Email) {
		http.Error(w, "verified admin email required", http.StatusForbidden)
		return
	}
	var html bytes.Buffer
	if err := AutoresponderEmail("HydroGo email preview", []string{"Hello Alex, this is a test of the active email layout.", "Sample reset link: https://example.invalid/reset?token=sample", "Sample verification code: 123456"}).Render(context.Background(), &html); err != nil {
		http.Error(w, "email preview unavailable", http.StatusInternalServerError)
		return
	}
	body := "Hello Alex, this is a test of the active email layout.\n\nSample reset link: https://example.invalid/reset?token=sample\n\nSample verification code: 123456"
	if h.Sender == nil {
		http.Error(w, "email sending unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := h.Sender.Send(r.Context(), Message{To: strings.TrimSpace(user.Email), Subject: "HydroGo email template preview", Body: body, HTMLBody: html.String()}); err != nil {
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = ui.EmailTestPreviewResult("Test email could not be sent. Check mail delivery settings.").Render(r.Context(), w)
			return
		}
		http.Error(w, "test email could not be sent", http.StatusServiceUnavailable)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = ui.EmailTestPreviewResult("Test email sent to your verified address.").Render(r.Context(), w)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}
