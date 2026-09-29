package mailer

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

type DNSDiagnosticsHandler struct {
	From, Selector string
	Records        DNSRecords
}

func (h DNSDiagnosticsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	checks, err := DiagnoseDNS(r.Context(), h.From, h.Selector, h.Records)
	if err != nil {
		http.Error(w, "Email domain settings are unavailable.", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Email domain diagnostics", "Mailer", security.NonceFromContext(r.Context()), "settings"); err != nil {
		return
	}
	view := make([]designsystem.EmailDNSCheck, len(checks))
	for i, check := range checks {
		view[i] = designsystem.EmailDNSCheck{Name: check.Name, Detail: check.Detail, Guidance: check.Guidance, OK: check.OK}
	}
	if err := designsystem.EmailDNSChecks(view).Render(r.Context(), w); err != nil {
		return
	}
	_ = ui.AdminEnd(w)
}
