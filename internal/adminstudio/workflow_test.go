package adminstudio

import "testing"

func TestPublicationTransitions(t *testing.T) {
	if !CanTransition(StateDraft, StatePublished) || !CanTransition(StatePublished, StateRetracted) || CanTransition(StatePublished, StateScheduled) {
		t.Fatal("unexpected transition rules")
	}
}

func TestPreviewURLIsSameOriginAndEscaped(t *testing.T) {
	got, err := PreviewURL("/articles/one?locale=tr", "a&b")
	if err != nil || got != "/articles/one?locale=tr&preview=a%26b" {
		t.Fatalf("preview URL = %q, err=%v", got, err)
	}
	if _, err := PreviewURL("https://evil.test", "token"); err == nil {
		t.Fatal("external preview target accepted")
	}
}
