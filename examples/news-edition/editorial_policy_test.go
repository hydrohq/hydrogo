package main

import (
	"errors"
	"testing"
	"time"
)

func TestEditorialDraftRequiresApprovalBeforeScheduling(t *testing.T) {
	draft := EditorialDraft{Headline: "Budget", Desk: "business", Scheduled: time.Now()}
	if !errors.Is(ValidateEditorialDraft(draft), ErrMissingApproval) {
		t.Fatal("scheduled draft without approval was accepted")
	}
	draft.ApprovedBy = "editor@example.test"
	if err := ValidateEditorialDraft(draft); err != nil {
		t.Fatalf("approved scheduled draft rejected: %v", err)
	}
}
