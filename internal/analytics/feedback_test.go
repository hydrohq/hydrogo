package analytics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type feedbackMemory struct {
	reaction string
	tenant   string
	totals   FeedbackTotals
}

func (s *feedbackMemory) RecordFeedback(_ context.Context, tenantID, _, _, reaction string, _ time.Time) error {
	s.tenant, s.reaction = tenantID, reaction
	if reaction == "up" {
		s.totals.Up++
	} else {
		s.totals.Down++
	}
	return nil
}
func (s *feedbackMemory) FeedbackTotals(context.Context, string, string, string) (FeedbackTotals, error) {
	return s.totals, nil
}

func TestFeedbackHandlerRecordsAggregateAndRendersThanks(t *testing.T) {
	store := &feedbackMemory{}
	ctx := tenant.WithScope(context.Background(), tenant.Scope{TenantID: "tenant-a", Slug: "tenant-a"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader("content_type=news&content_id=article-a&element_id=article-feedback&reaction=up")).WithContext(ctx)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	FeedbackHandler{Store: store, Now: func() time.Time { return time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC) }}.ServeHTTP(w, r)
	if w.Code != http.StatusOK || store.tenant != "tenant-a" || store.reaction != "up" || !strings.Contains(w.Body.String(), "Geri bildiriminiz için teşekkürler") {
		t.Fatalf("status=%d store=%+v body=%q", w.Code, store, w.Body.String())
	}
}

func TestFeedbackHandlerRejectsInvalidContentAndWrongMethod(t *testing.T) {
	h := FeedbackHandler{Store: &feedbackMemory{}}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader("content_type=other&content_id=a&element_id=x&reaction=up"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid content status=%d", w.Code)
	}
	r = httptest.NewRequest(http.MethodGet, "/api/v1/feedback", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d", w.Code)
	}
}

func TestFeedbackMutationRequiresSharedCSRFAndHonorsRateLimit(t *testing.T) {
	store := &feedbackMemory{}
	handler := security.CSRFMiddleware(false, nil)(FeedbackHandler{Store: store})
	body := "_csrf=csrf&content_type=news&content_id=article-a&element_id=article-feedback&reaction=up"
	r := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r = r.WithContext(tenant.WithScope(r.Context(), tenant.Scope{TenantID: "tenant-a", Slug: "tenant-a"}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || store.reaction != "" {
		t.Fatalf("missing CSRF accepted: status=%d store=%+v", w.Code, store)
	}
	r = httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "csrf"})
	r = r.WithContext(tenant.WithScope(r.Context(), tenant.Scope{TenantID: "tenant-a", Slug: "tenant-a"}))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK || store.reaction != "up" {
		t.Fatalf("valid CSRF rejected: status=%d store=%+v", w.Code, store)
	}
	limited := FeedbackHandler{Store: store, Limiter: feedbackDenyLimiter{}}
	r = httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	limited.ServeHTTP(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limiter status=%d", w.Code)
	}
}

func TestFeedbackHandlerFailsClosedWhenTenantScopeIsRequired(t *testing.T) {
	store := &feedbackMemory{}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader("content_type=news&content_id=article-a&element_id=article-feedback&reaction=up"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	FeedbackHandler{Store: store, RequireTenantScope: true}.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || store.tenant != "" {
		t.Fatalf("missing tenant scope status=%d store=%+v", w.Code, store)
	}
}

type feedbackDenyLimiter struct{}

func (feedbackDenyLimiter) Allow(string) bool { return false }
