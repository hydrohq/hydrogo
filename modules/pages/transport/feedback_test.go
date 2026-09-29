package transport

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

type feedbackCountsStub struct{ ids []string }

func (s *feedbackCountsStub) FeedbackTotals(context.Context, string, string, string) (analytics.FeedbackTotals, error) {
	return analytics.FeedbackTotals{}, nil
}
func (s *feedbackCountsStub) FeedbackCounts(_ context.Context, _ string, _ string, ids []string) (map[string]analytics.FeedbackTotals, error) {
	s.ids = append([]string(nil), ids...)
	return map[string]analytics.FeedbackTotals{ids[0]: {Up: 5, Down: 2}}, nil
}

func TestAdminFAQFeedbackMetricsShowsTenantScopedCounts(t *testing.T) {
	store := &feedbackCountsStub{}
	h := Handler{AnalyticsEnabled: true, Feedback: store}
	blocks := []pages.Block{{Type: "faq", Payload: []byte(`{"items":[{"question":"How?","answer":"This way"}]}`)}}
	got := h.faqFeedbackMetrics(httptest.NewRequest("GET", "/admin/pages/x", nil), blocks)
	if len(got) != 1 || got[0].Question != "How?" || got[0].Up != 5 || got[0].Down != 2 || len(store.ids) != 1 {
		t.Fatalf("metrics=%+v ids=%v", got, store.ids)
	}
}
