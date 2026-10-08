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

func TestValidateRejectsCrossSurfaceTestIDCollision(t *testing.T) {
	m := manifest{Version: 1, BaseURL: "http://127.0.0.1:8090", Admin: adminManifest{
		LoginPath: "/login",
		Surfaces:  []adminSurface{{ID: "admin", Route: "/admin", TestID: "shared"}},
	}, Frontend: frontendManifest{Routes: []frontendRoute{{Name: "home", Path: "/", Public: true, TestID: "shared"}}}}
	if err := validate(m); err == nil || !strings.Contains(err.Error(), "duplicate test_id") {
		t.Fatalf("expected cross-surface collision rejection, got %v", err)
	}
}

func TestRenderIncludesDeclaredMediaUploadJourney(t *testing.T) {
	m := manifest{Version: 1, BaseURL: "http://127.0.0.1:8090", Admin: adminManifest{LoginPath: "/login"}, Media: mediaManifest{Route: "/admin/media", UploadEnabled: true}}
	output := render(m)
	if !strings.Contains(output, "admin: media upload") || !strings.Contains(output, "setInputFiles") {
		t.Fatal("declared media upload journey was not generated")
	}
}

func TestRenderIncludesNavigationLifecycleJourney(t *testing.T) {
	m := manifest{Version: 1, BaseURL: "http://127.0.0.1:8090", Admin: adminManifest{LoginPath: "/login"}, Frontend: frontendManifest{Routes: []frontendRoute{{Name: "home", Path: "/", Public: true, TestID: "page-home"}}}}
	output := render(m)
	if !strings.Contains(output, "frontend navigation lifecycle: home") || !strings.Contains(output, "page.reload()") || !strings.Contains(output, "page.goBack()") || !strings.Contains(output, "page.goForward()") {
		t.Fatal("navigation lifecycle journey was not generated")
	}
}

func TestRenderRefreshesExpiredAdminStorageState(t *testing.T) {
	m := manifest{Version: 1, BaseURL: "http://127.0.0.1:8090", Admin: adminManifest{LoginPath: "/login", Surfaces: []adminSurface{{ID: "admin", Route: "/admin", TestID: "admin-home"}}}}
	output := render(m)
	for _, expected := range []string{"test.beforeEach(async ({ page })", "HYDRO_E2E_ADMIN_EMAIL", "HYDRO_E2E_ADMIN_PASSWORD", "saved session has expired"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("generated Admin tests do not recover an expired saved session; missing %q", expected)
		}
	}
}
