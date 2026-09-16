package v1

import "testing"

func TestEditorialWorkflowContract(t *testing.T) {
	w := EditorialWorkflow{States: []EditorialState{EditorialDraft, EditorialPublished}, Transitions: map[EditorialState][]EditorialState{EditorialDraft: {EditorialPublished}}}
	if err := ValidateEditorialTransition(w, EditorialDraft, EditorialPublished); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEditorialTransition(w, EditorialPublished, EditorialDraft); err == nil {
		t.Fatal("expected invalid transition")
	}
}

func TestPreviewPathRejectsExternalTargets(t *testing.T) {
	if _, err := PreviewPath("//evil.example", "token"); err == nil {
		t.Fatal("expected external target rejection")
	}
	got, err := PreviewPath("/news/article?locale=tr", "a b")
	if err != nil || got != "/news/article?locale=tr&preview=a+b" {
		t.Fatalf("unexpected preview path: %q %v", got, err)
	}
}
