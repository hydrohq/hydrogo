package transport

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type aiSuggestInput struct {
	Operation    string `json:"operation"`
	TargetLocale string `json:"target_locale"`
	Tone         string `json:"tone"`
	Consent      bool   `json:"consent"`
}

// GET /api/v1/admin/pages/ai/status reports whether the AI assistant is configured and active.
func (h Handler) adminAIStatus(w http.ResponseWriter, r *http.Request) {
	api.WriteData(w, http.StatusOK, map[string]any{
		"enabled": h.assistantEnabled(),
	}, nil)
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
	if h.AILimiter != nil && !security.AllowForRequest(h.AILimiter, r.Context(), "ai-suggest:"+user.ID) {
		security.WriteRateLimitedResponse(w, "too many AI requests", time.Minute)
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
	tenantID, ok := h.presenceTenant(w, r)
	if !ok {
		return
	}
	if err := h.checkAIQuota(w, r, tenantID); err != nil {
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
	h.consumeAIQuota(r, tenantID, estimateAITokens(page.Title, page.Body, result.Title, result.Body))
	api.WriteData(w, http.StatusOK, map[string]any{"before": map[string]string{"title": page.Title, "body": page.Body}, "suggestion": result}, nil)
}

// checkAIQuota pre-checks the tenant AI token quota (consuming nothing) and
// writes the matching error response when the workspace is out of AI budget.
func (h Handler) checkAIQuota(w http.ResponseWriter, r *http.Request, tenantID string) error {
	if h.AIDB == nil || h.Dialect == "" {
		return nil
	}
	if _, err := tenant.ConsumeAITokens(r.Context(), h.AIDB, h.Dialect, tenantID, 0); err != nil {
		switch {
		case errors.Is(err, tenant.ErrAIQuotaExceeded):
			api.WriteError(w, r, http.StatusTooManyRequests, "monthly AI token quota exceeded for this workspace", nil)
		case errors.Is(err, tenant.ErrAIDisabled):
			api.WriteError(w, r, http.StatusForbidden, "AI assistance is disabled for this workspace", nil)
		default:
			api.WriteError(w, r, http.StatusInternalServerError, "failed to check the workspace AI quota", nil)
		}
		return err
	}
	return nil
}

// consumeAIQuota records estimated usage after a successful call. The reply is
// already produced, so a bookkeeping failure is logged instead of failing the
// request; the pre-check above is the hard gate.
func (h Handler) consumeAIQuota(r *http.Request, tenantID string, tokens int64) {
	if h.AIDB == nil || h.Dialect == "" || tokens <= 0 {
		return
	}
	if _, err := tenant.ConsumeAITokens(r.Context(), h.AIDB, h.Dialect, tenantID, tokens); err != nil {
		slog.Warn("recording AI token usage failed", "tenant", tenantID, "tokens", tokens, "error", err)
	}
}

func estimateAITokens(titles ...string) int64 {
	total := 0
	for _, s := range titles {
		total += len(s)
	}
	return int64(total/4) + 1
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
	if err := h.Presence.Heartbeat(realtime.PresenceSession{TenantID: tenantID, Resource: "page:" + r.PathValue("id"), UserID: user.ID, Email: user.Email, Name: name}); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "presence is unavailable", nil)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusOK, h.Presence.Active(tenantID, "page:"+r.PathValue("id"), user.ID), nil)
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

// DELETE /api/v1/admin/pages/{id}/presence
func (h Handler) editPresenceLeave(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if h.Presence == nil || !editorAllowed(r) || user == nil {
		writeForbidden(w, r)
		return
	}
	tenantID, ok := h.presenceTenant(w, r)
	if !ok {
		return
	}
	h.Presence.Leave(tenantID, "page:"+r.PathValue("id"), user.ID)
	w.WriteHeader(http.StatusNoContent)
}
