package transport

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type aiSuggestInput struct {
	Operation    string `json:"operation"`
	TargetLocale string `json:"target_locale"`
	Tone         string `json:"tone"`
	Consent      bool   `json:"consent"`
}

func currentUser(r *http.Request) *auth.User {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		return nil
	}
	return user
}

// POST /api/v1/admin/pages/{id}/ai-suggest asks the configured assistant for
// a suggestion; the draft is never modified here.
func (h Handler) adminAISuggest(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if !editorAllowed(r) || !h.assistantEnabled() || user == nil {
		writeForbidden(w, r)
		return
	}
	var input aiSuggestInput
	if !api.DecodeJSON(w, r, &input, 8<<10) {
		return
	}
	operation := ai.Operation(strings.TrimSpace(input.Operation))
	targetLocale := strings.TrimSpace(input.TargetLocale)
	if !input.Consent {
		api.WriteValidation(w, r, map[string]string{"consent": "required"})
		return
	}
	if operation != ai.Refine && operation != ai.SEO && operation != ai.Summary && operation != ai.Translate {
		api.WriteValidation(w, r, map[string]string{"operation": "invalid"})
		return
	}
	if operation == ai.Translate && !hasLocale(h.SupportedLocales, targetLocale) {
		api.WriteValidation(w, r, map[string]string{"target_locale": "invalid"})
		return
	}
	page, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		writePageReadError(w, r, err)
		return
	}
	if h.Content.Transactions == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "the assistant is unavailable", nil)
		return
	}
	auditID := "ai-request-" + fmt.Sprintf("%x", sha256.Sum256([]byte(page.ID+"\x00"+user.ID+"\x00"+time.Now().UTC().Format(time.RFC3339Nano))))[:32]
	if err := h.Content.Transactions.Within(r.Context(), func(tx database.Executor) error {
		return h.AIAudit.Record(r.Context(), tx, auditID, "ai.suggestion.requested", page.ID)
	}); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "the assistant is unavailable", nil)
		return
	}
	result, err := h.Assistant.Suggest(r.Context(), ai.Request{
		Operation:    operation,
		Locale:       page.Locale,
		TargetLocale: targetLocale,
		Tone:         input.Tone,
		Title:        page.Title,
		Body:         page.Body,
		Consent:      true,
	})
	if err != nil {
		if errors.Is(err, ai.ErrConsent) || errors.Is(err, ai.ErrInvalidRequest) {
			api.WriteError(w, r, http.StatusBadRequest, "the suggestion request is invalid", nil)
			return
		}
		api.WriteError(w, r, http.StatusServiceUnavailable, "no suggestion could be produced; the draft is unchanged", nil)
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{"before": map[string]string{"title": page.Title, "body": page.Body}, "suggestion": result}, nil)
}

func hasLocale(locales []string, target string) bool {
	for _, locale := range locales {
		if strings.EqualFold(strings.ReplaceAll(strings.TrimSpace(locale), "_", "-"), strings.ReplaceAll(strings.TrimSpace(target), "_", "-")) {
			return true
		}
	}
	return false
}

// POST /api/v1/admin/pages/{id}/ai-applied records that an editor accepted a
// suggestion (the content itself is saved through PUT).
func (h Handler) adminAIApplied(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if !editorAllowed(r) || !h.assistantEnabled() || user == nil {
		writeForbidden(w, r)
		return
	}
	pageID := r.PathValue("id")
	if _, err := h.Content.Get(r.Context(), h.DB, pageID); err != nil {
		writePageReadError(w, r, err)
		return
	}
	if h.Content.Transactions == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "the assistant is unavailable", nil)
		return
	}
	auditID := "ai-applied-" + fmt.Sprintf("%x", sha256.Sum256([]byte(pageID+"\x00"+user.ID+"\x00"+time.Now().UTC().Format(time.RFC3339Nano))))[:32]
	if err := h.Content.Transactions.Within(r.Context(), func(tx database.Executor) error {
		return h.AIAudit.Record(r.Context(), tx, auditID, "ai.suggestion.applied", pageID)
	}); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "the suggestion audit record could not be written", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) presenceTenant(w http.ResponseWriter, r *http.Request) (string, bool) {
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		return scope.TenantID, true
	}
	if h.RequireTenantScope {
		writeForbidden(w, r)
		return "", false
	}
	return "default", true
}

// POST /api/v1/admin/pages/{id}/presence
func (h Handler) editPresenceHeartbeat(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if h.Presence == nil || !editorAllowed(r) || user == nil {
		writeForbidden(w, r)
		return
	}
	if len(r.PathValue("id")) > 128 || strings.ContainsAny(r.PathValue("id"), "\x00\r\n") {
		api.WriteError(w, r, http.StatusBadRequest, "invalid resource", nil)
		return
	}
	tenantID, ok := h.presenceTenant(w, r)
	if !ok {
		return
	}
	name := strings.TrimSpace(strings.SplitN(user.Email, "@", 2)[0])
	if name == "" {
		name = "Editor"
	}
	if err := h.Presence.Heartbeat(realtime.PresenceSession{TenantID: tenantID, Resource: "page:" + r.PathValue("id"), UserID: user.ID, Name: name}); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "presence is unavailable", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/admin/pages/{id}/presence
func (h Handler) editPresenceList(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if h.Presence == nil || !editorAllowed(r) || user == nil {
		writeForbidden(w, r)
		return
	}
	tenantID, ok := h.presenceTenant(w, r)
	if !ok {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusOK, h.Presence.Active(tenantID, "page:"+r.PathValue("id"), user.ID), nil)
}
