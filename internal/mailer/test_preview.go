package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

type TestPreviewHandler struct{ Sender Sender }

func (h TestPreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin || !auth.ValidateEmail(user.Email) {
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
		http.Error(w, "test email could not be sent", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}
