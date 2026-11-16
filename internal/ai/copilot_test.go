package ai

import (
	"testing"

	"github.com/hydrohq/hydrogo/internal/contenttypes"
)

func TestParseCopilotPlan(t *testing.T) {
	raw := "```json\n{\"reply\":\"Updated color\",\"operations\":[{\"op\":\"update\",\"collection_id\":\"site_settings\",\"record_id\":\"rec-1\",\"data\":{\"primary_color\":\"#123456\"}}]}\n```"
	plan, err := ParseCopilotPlan(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing plan: %v", err)
	}
	if plan.Reply != "Updated color" {
		t.Errorf("got reply %q, want %q", plan.Reply, "Updated color")
	}
	if len(plan.Operations) != 1 {
		t.Fatalf("got %d operations, want 1", len(plan.Operations))
	}
	if plan.Operations[0].CollectionID != "site_settings" {
		t.Errorf("got collection_id %q, want site_settings", plan.Operations[0].CollectionID)
	}
}

func TestValidateCopilotOperations(t *testing.T) {
	defs := []contenttypes.Definition{
		{
			ID: "site_settings",
			Fields: []contenttypes.Field{
				{ID: "primary_color", Kind: "text"},
				{ID: "bg_tone", Kind: "text"},
			},
		},
	}

	validOps := []CopilotOperation{
		{
			Op:           "update",
			CollectionID: "site_settings",
			Data:         map[string]any{"primary_color": "#ffffff"},
		},
	}
	if err := ValidateCopilotOperations(validOps, defs); err != nil {
		t.Errorf("expected valid ops to pass, got: %v", err)
	}

	invalidColOps := []CopilotOperation{
		{
			Op:           "update",
			CollectionID: "unknown_collection",
			Data:         map[string]any{"foo": "bar"},
		},
	}
	if err := ValidateCopilotOperations(invalidColOps, defs); err == nil {
		t.Error("expected error for unknown collection, got nil")
	}

	invalidFieldOps := []CopilotOperation{
		{
			Op:           "update",
			CollectionID: "site_settings",
			Data:         map[string]any{"malicious_field": "injected"},
		},
	}
	if err := ValidateCopilotOperations(invalidFieldOps, defs); err == nil {
		t.Error("expected error for unknown field on collection, got nil")
	}
}

func TestCloneMap(t *testing.T) {
	if CloneMap(nil) != nil {
		t.Error("CloneMap(nil) should return nil")
	}
	orig := map[string]any{"a": 1, "b": "str"}
	cloned := CloneMap(orig)
	if len(cloned) != 2 || cloned["a"] != 1 || cloned["b"] != "str" {
		t.Errorf("cloned map mismatch: %v", cloned)
	}
	cloned["c"] = 3
	if _, ok := orig["c"]; ok {
		t.Error("mutating clone modified original")
	}
}
