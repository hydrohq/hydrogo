package transport

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"

	"github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type pageSuggestionKey struct{}

type pageSuggestion struct {
	BeforeTitle string
	BeforeBody  string
	Result      ai.Suggestion
}

func (h Handler) adminAISuggest(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) || !h.assistantEnabled() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	operation := ai.Operation(strings.TrimSpace(r.FormValue("operation")))
	targetLocale := strings.TrimSpace(r.FormValue("target_locale"))
	if r.FormValue("consent") != "yes" || (operation != ai.Refine && operation != ai.SEO && operation != ai.Summary && operation != ai.Translate) || (operation == ai.Translate && !hasLocale(h.SupportedLocales, targetLocale)) {
		http.Error(w, "Öneri için bir işlem seçin ve içerik paylaşımını onaylayın.", http.StatusBadRequest)
		return
	}
	page, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		http.Error(w, "page unavailable", pageReadStatus(err))
		return
	}
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.Content.Transactions == nil {
		http.Error(w, "Öneri hizmeti şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	auditID := "ai-request-" + fmt.Sprintf("%x", sha256.Sum256([]byte(page.ID+"\x00"+user.ID+"\x00"+time.Now().UTC().Format(time.RFC3339Nano))))[:32]
	if err := h.Content.Transactions.Within(r.Context(), func(tx database.Executor) error {
		return h.AIAudit.Record(r.Context(), tx, auditID, "ai.suggestion.requested", page.ID)
	}); err != nil {
		http.Error(w, "Öneri hizmeti şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	result, err := h.Assistant.Suggest(r.Context(), ai.Request{
		Operation:    operation,
		Locale:       page.Locale,
		TargetLocale: targetLocale,
		Tone:         r.FormValue("tone"),
		Title:        page.Title,
		Body:         page.Body,
		Consent:      r.FormValue("consent") == "yes",
	})
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, ai.ErrConsent) || errors.Is(err, ai.ErrInvalidRequest) {
			status = http.StatusBadRequest
		}
		http.Error(w, "Öneri üretilemedi. İçeriğiniz değiştirilmedi.", status)
		return
	}
	suggestionContext := context.WithValue(r.Context(), pageSuggestionKey{}, pageSuggestion{BeforeTitle: page.Title, BeforeBody: page.Body, Result: result})
	h.adminPage(w, r.WithContext(suggestionContext))
}

func hasLocale(locales []string, target string) bool {
	for _, locale := range locales {
		if strings.EqualFold(strings.ReplaceAll(strings.TrimSpace(locale), "_", "-"), strings.ReplaceAll(strings.TrimSpace(target), "_", "-")) {
			return true
		}
	}
	return false
}

func (h Handler) adminAIApplied(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !editorAllowed(r) || !h.assistantEnabled() || user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	pageID := r.PathValue("id")
	if _, err := h.Content.Get(r.Context(), h.DB, pageID); err != nil {
		http.Error(w, "page unavailable", pageReadStatus(err))
		return
	}
	if h.Content.Transactions == nil {
		http.Error(w, "Öneri hizmeti şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	auditID := "ai-applied-" + fmt.Sprintf("%x", sha256.Sum256([]byte(pageID+"\x00"+user.ID+"\x00"+time.Now().UTC().Format(time.RFC3339Nano))))[:32]
	if err := h.Content.Transactions.Within(r.Context(), func(tx database.Executor) error {
		return h.AIAudit.Record(r.Context(), tx, auditID, "ai.suggestion.applied", pageID)
	}); err != nil {
		http.Error(w, "Öneri denetim kaydı tamamlanamadı.", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) editPresenceHeartbeat(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if h.Presence == nil || !editorAllowed(r) || user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if len(r.PathValue("id")) > 128 || strings.ContainsAny(r.PathValue("id"), "\x00\r\n") {
		http.Error(w, "invalid resource", http.StatusBadRequest)
		return
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	} else if h.RequireTenantScope {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	name := strings.TrimSpace(strings.SplitN(user.Email, "@", 2)[0])
	if name == "" {
		name = "Editor"
	}
	err := h.Presence.Heartbeat(realtime.PresenceSession{TenantID: tenantID, Resource: "page:" + r.PathValue("id"), UserID: user.ID, Name: name})
	if err != nil {
		http.Error(w, "presence unavailable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) editPresenceList(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if h.Presence == nil || !editorAllowed(r) || user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	} else if h.RequireTenantScope {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteJSON(w, http.StatusOK, h.Presence.Active(tenantID, "page:"+r.PathValue("id"), user.ID))
}
