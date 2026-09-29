package designsystem

import (
	"strings"
	"testing"
)

func TestHelpfulFeedbackUsesTypedHTMXContractAndCSRF(t *testing.T) {
	got := render(t, HelpfulFeedback(HelpfulFeedbackProps{ContentType: "news", ContentID: "article-a", ElementID: "article-feedback", CSRF: "csrf", Nonce: "nonce", Up: 4, Down: 1, LoadHTMX: true}))
	for _, want := range []string{`hx-post="/api/v1/feedback"`, `hx-target="#article-feedback"`, `hx-swap="outerHTML"`, `name="_csrf" value="csrf"`, `data-feedback-type="news"`, `4 kişi faydalı buldu · 1 kişi faydalı bulmadı`, `/vendor/htmx/htmx.min.js`} {
		if !strings.Contains(got, want) {
			t.Errorf("feedback widget missing %q", want)
		}
	}
}

func TestFAQFeedbackIsEnabledOnlyByRenderOptionAndTenantSafeID(t *testing.T) {
	faq := FAQItem{Question: "When?", Answer: "Tomorrow"}
	id := FAQFeedbackID(faq, 0)
	props := FAQProps{ID: "help", Items: []FAQItem{faq}, FeedbackEnabled: true, CSRF: "csrf", Nonce: "nonce", FeedbackCounts: map[string][2]int64{id: {3, 2}}}
	got := render(t, FAQList(props))
	for _, want := range []string{`data-feedback-type="faq"`, `name="content_id" value="` + id + `"`, `id="faq-feedback-` + id + `"`, `3 kişi faydalı buldu · 2 kişi faydalı bulmadı`} {
		if !strings.Contains(got, want) {
			t.Errorf("FAQ feedback missing %q", want)
		}
	}
	without := render(t, FAQList(FAQProps{Items: []FAQItem{faq}}))
	if strings.Contains(without, `data-feedback-type="faq"`) {
		t.Fatal("feedback rendered when disabled")
	}
}
