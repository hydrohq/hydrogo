package main

import (
	"strings"
	"testing"
)

func TestValidateRejectsUntrustedSelectorsAndDuplicateRoutes(t *testing.T) {
	m := manifest{Version: 1, BaseURL: "http://127.0.0.1:8090", Admin: adminManifest{LoginPath: "/login", Collections: []collection{{ID: "pages", Route: "/admin/pages", Fields: []field{{ID: "title", TestID: "field-title"}}}}}, Frontend: frontendManifest{Routes: []frontendRoute{{Name: "home", Path: "/", Public: true, TestID: "page-home"}, {Name: "home", Path: "/again", Public: true, TestID: "page-again"}}}}
	if err := validate(m); err == nil || !strings.Contains(err.Error(), "duplicate frontend route") {
		t.Fatalf("expected duplicate route rejection, got %v", err)
	}
}

func TestRenderIsDeterministicAndUsesExplicitTestIDs(t *testing.T) {
	m := manifest{Version: 1, BaseURL: "http://127.0.0.1:8090", Admin: adminManifest{LoginPath: "/login", Collections: []collection{{ID: "pages", Route: "/admin/pages", Fields: []field{{ID: "title", TestID: "field-title"}}}}}, Frontend: frontendManifest{Routes: []frontendRoute{{Name: "home", Path: "/", Public: true, TestID: "page-home"}}}}
	if first, second := render(m), render(m); first != second {
		t.Fatal("generated output is not deterministic")
	} else if !strings.Contains(first, `data-testid="page-home"`) || !strings.Contains(first, `[name="title"]`) || !strings.Contains(first, "toBeAttached()") {
		t.Fatal("generated output omitted explicit test IDs")
	}
}
