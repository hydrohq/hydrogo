package designsystem

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/ai"
)

func TestAIAssistantComponentsExplainConsentAndKeepApplyInDraft(t *testing.T) {
	var request bytes.Buffer
	if err := AIAssistantRequest("/admin/pages/page1/ai-suggest", "csrf-token", []string{"en", "tr"}).Render(context.Background(), &request); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"name=\"consent\"", "name=\"_csrf\"", "name=\"target_locale\"", "value=\"tr\"", "Başka dile çevir", "kaydedilmez veya yayımlanmaz", "required"} {
		if !strings.Contains(request.String(), want) {
			t.Fatalf("request component lacks %q", want)
		}
	}
	var suggestion bytes.Buffer
	if err := AIAssistantSuggestion("Old title", "Old body", ai.Suggestion{Title: "New title", Body: "New body"}, "/admin/pages/page1/ai-applied", "csrf-token", "nonce-safe").Render(context.Background(), &suggestion); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Old body", "New body", `nonce="nonce-safe"`, "setText(nextBody)", "Taslak düzenleyici güncellendi", "X-CSRF-Token"} {
		if !strings.Contains(suggestion.String(), want) {
			t.Fatalf("suggestion component lacks %q", want)
		}
	}
	if strings.Contains(suggestion.String(), `type=\"submit\"`) {
		t.Fatal("applying a suggestion should not submit or save the content")
	}
}

func TestAIInboxSummaryRequiresExplicitConsentAndEscapesSummary(t *testing.T) {
	var output bytes.Buffer
	component := AIInboxSummary([]AIInboxOption{{ID: "private-id", Label: "Talep · 2026-09-29 12:00 UTC"}}, "/admin/site/inbox/contact/ai-summarize", "csrf-safe", `<script>alert(1)</script>`)
	if err := component.Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`name="submission_id"`, `name="consent"`, `name="_csrf"`, "required", "Talep · 2026-09-29 12:00 UTC", "&lt;script&gt;alert(1)&lt;/script&gt;"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("inbox AI component missing %q", want)
		}
	}
	if strings.Contains(output.String(), `<script>alert(1)</script>`) || strings.Contains(output.String(), `>private-id<`) {
		t.Fatal("summary markup exposed unsafe HTML or displayed the internal record id")
	}
}
