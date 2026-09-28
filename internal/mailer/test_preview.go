package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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
		_, pingEnabled := h.Sender.(interface {
			Ping(context.Context) (PingResult, error)
		})
		if err := ui.EmailTestPreviewForm(security.CSRFTokenFromContext(r.Context()), user.EmailVerifiedAt != nil, r.URL.Query().Get("verification_sent") == "1", pingEnabled).Render(r.Context(), w); err != nil {
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
	if user == nil || !user.Active || user.Role != auth.RoleAdmin {
		http.Error(w, "verified admin email required", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if r.FormValue("_action") == "ping" {
		pingSender, ok := h.Sender.(interface {
			Ping(context.Context) (PingResult, error)
		})
		if !ok {
			http.Error(w, "SMTP diagnostics unavailable", http.StatusServiceUnavailable)
			return
		}
		result, err := pingSender.Ping(r.Context())
		message := "SMTP connection verified."
		if err != nil {
			message = "SMTP connection failed. Check the configured host and port."
			if result.Stage == "tls" {
				message = "TLS negotiation failed. Check the server TLS settings."
			} else if result.Stage == "authentication" {
				message = "SMTP authentication failed. Check the configured credentials."
			}
		} else if result.Authenticated {
			message = "SMTP connection and authentication verified."
		}
		if result.TLSVersion != "" {
			message += " " + result.TLSVersion + " is active."
		}
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			_ = ui.EmailTestPreviewResult(message).Render(r.Context(), w)
			return
		}
		status := http.StatusOK
		if err != nil {
			status = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, message)
		return
	}
	if user.EmailVerifiedAt == nil || !auth.ValidateEmail(user.Email) {
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
