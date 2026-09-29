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
	if err := AIAssistantRequest("/admin/pages/page1/ai-suggest", "csrf-token").Render(context.Background(), &request); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"name=\"consent\"", "name=\"_csrf\"", "kaydedilmez veya yayımlanmaz", "required"} {
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
