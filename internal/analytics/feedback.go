package analytics

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type FeedbackStore interface {
	RecordFeedback(ctx context.Context, tenantID, contentType, contentID, reaction string, day time.Time) error
	FeedbackTotals(ctx context.Context, tenantID, contentType, contentID string) (FeedbackTotals, error)
}

type FeedbackTotals struct{ Up, Down int64 }

type FeedbackHandler struct {
	Store              FeedbackStore
	Limiter            security.RateLimiter
	Now                func() time.Time
	RequireTenantScope bool
}

func (h FeedbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if parseErr := r.ParseForm(); parseErr != nil {
		http.Error(w, "Invalid feedback", http.StatusBadRequest)
		return
	}
	contentType, contentID, reaction := r.FormValue("content_type"), r.FormValue("content_id"), r.FormValue("reaction")
	elementID := r.FormValue("element_id")
	if h.Store == nil || !validFeedbackElementID(elementID) || (contentType != "news" && contentType != "faq") || strings.TrimSpace(contentID) == "" || len(contentID) > 128 || (reaction != "up" && reaction != "down") {
		http.Error(w, "Invalid feedback", http.StatusBadRequest)
		return
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	} else if h.RequireTenantScope {
		http.Error(w, "Feedback is unavailable", http.StatusForbidden)
		return
	}
	if h.Limiter != nil && !security.AllowForRequest(h.Limiter, r.Context(), tenantID+":"+requestmeta.ClientIP(r)) {
		http.Error(w, "Please wait before sending feedback again.", http.StatusTooManyRequests)
		return
	}
	now := time.Now
	if h.Now != nil {
		now = h.Now
	}
	if err := h.Store.RecordFeedback(r.Context(), tenantID, contentType, contentID, reaction, now().UTC()); err != nil {
		http.Error(w, "Feedback could not be saved", http.StatusServiceUnavailable)
		return
	}
	totals, err := h.Store.FeedbackTotals(r.Context(), tenantID, contentType, contentID)
	if err != nil {
		http.Error(w, "Feedback could not be loaded", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"element_id": elementID, "up": totals.Up, "down": totals.Down})
}

func validFeedbackElementID(value string) bool {
	if value == "" || len(value) > 80 {
		return false
	}
	for _, r := range value {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
