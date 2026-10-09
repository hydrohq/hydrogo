package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedRuntimeWiringListsSelectedKits(t *testing.T) {
	root := t.TempDir()
	spec := Spec{Name: "article", Table: "articles", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"read"}, Dialects: []string{"sqlite"}, Kits: []string{"seo", "search"}}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "generated", "article", "RUNTIME_WIRING.md"))
	if err != nil || !strings.Contains(string(b), "Selected kits: search, seo") {
		t.Fatalf("runtime wiring kits missing: %v\n%s", err, b)
	}
}

func TestGeneratedRuntimeWiringExplainsEditionCRUDCapabilities(t *testing.T) {
	root := t.TempDir()
	spec := Spec{Name: "article", Table: "articles", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"article.read"}, Dialects: []string{"sqlite", "postgres"}}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	wiring, err := os.ReadFile(filepath.Join(root, "generated", "article", "RUNTIME_WIRING.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		"Edition authorization callback",
		"ResourceAccess",
		"same user, tenant and record rules as the",
		"New Edition compositions should",
		"displays all basic CRUD actions as a legacy compatibility behavior",
		"authorization boundary for every caller",
	} {
		if !strings.Contains(string(wiring), marker) {
			t.Errorf("runtime wiring omitted CRUD security guidance %q: %s", marker, wiring)
		}
	}
}

func TestGeneratedAdminRegistrationTracksBlueprintResources(t *testing.T) {
	root := t.TempDir()
	resources := []Spec{
		{Name: "article", Table: "articles", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"read"}, Dialects: []string{"sqlite", "postgres"}},
		{Name: "project", Table: "projects", Fields: []Field{{Name: "name", Type: "string"}}, Permissions: []string{"read"}, Dialects: []string{"sqlite", "postgres"}},
	}
	for _, resource := range resources {
		if _, err := Generate(root, resource); err != nil {
			t.Fatal(err)
		}
	}
	registration, err := os.ReadFile(filepath.Join(root, "generated", "admin_registration.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`func RegisterGeneratedAdmin(mux *http.ServeMux, options adminstudio.GeneratedAdminOptions) error`,
		`[]string{"article", "project"}`,
		`adminstudio.RegisterGenerated(mux, AdminRegistry`,
	} {
		if !strings.Contains(string(registration), want) {
			t.Fatalf("generated Admin registration missing %q:\n%s", want, registration)
		}
	}
}

func TestGeneratedAdminRegistrationCompatibilityRejectsStaleResourceSelection(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"article", "project"} {
		if _, err := Generate(root, Spec{Name: name, Table: name + "s", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"read"}, Dialects: []string{"sqlite", "postgres"}}); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(root, "generated", "admin_registration.go")
	if err := os.WriteFile(path, []byte("// stale generated activation list\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckGeneratedCompatibility(root); err == nil || !strings.Contains(err.Error(), "Admin registration does not match") {
		t.Fatalf("stale generated resource selection was accepted: %v", err)
	}
}

func TestGeneratedResourceRouteRegistrationTracksResourcesAndTenantResolvers(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module edition.example/site\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resources := []Spec{
		{Name: "article", Table: "articles", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"article.read"}, Dialects: []string{"sqlite", "postgres"}},
		{Name: "workspace", Table: "workspaces", TenantScoped: true, Fields: []Field{{Name: "name", Type: "string"}}, Permissions: []string{"workspace.read"}, Dialects: []string{"sqlite", "postgres"}},
	}
	for _, resource := range resources {
		if _, err := Generate(root, resource); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(root, "generated", "route_registration.go")
	registration, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`edition.example/site/generated/article`,
		`edition.example/site/generated/workspace`,
		`RegisterArticleRoute(mux, options.DB, options.Dialect, policy0)`,
		`RegisterWorkspaceRoute(mux, options.DB, options.Dialect, policy1, resolver1)`,
		`options.ResolveTenant("workspace", r)`,
		`generated tenant resource %s has no tenant resolver`,
	} {
		if !strings.Contains(string(registration), want) {
			t.Fatalf("aggregate generated route registration missing %q:\n%s", want, registration)
		}
	}
	if _, err := CheckGeneratedCompatibility(root); err != nil {
		t.Fatalf("generated route helper rejected: %v", err)
	}
	if err := os.WriteFile(path, []byte("// stale helper\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckGeneratedCompatibility(root); err == nil || !strings.Contains(err.Error(), "resource route registration does not match") {
		t.Fatalf("stale aggregate route helper was accepted: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "go.mod")); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(root, resources[1]); err != nil {
		t.Fatalf("regenerate without a Go module: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("module-less output retained an aggregate import helper: %v", err)
	}
	if _, err := CheckGeneratedCompatibility(root); err != nil {
		t.Fatalf("module-less generated resource set rejected: %v", err)
	}
}

func TestGeneratedKitArtifactsIncludeDependencies(t *testing.T) {
	root := t.TempDir()
	spec := Spec{Name: "order", Table: "orders", Fields: []Field{{Name: "total", Type: "decimal"}}, Permissions: []string{"read"}, Dialects: []string{"sqlite"}, Kits: []string{"commerce"}}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(root, "generated", "order")
	for _, file := range []string{"RUNTIME_WIRING.md", "kits.ts"} {
		b, err := os.ReadFile(filepath.Join(base, file))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "forms") || !strings.Contains(string(b), "commerce") {
			t.Fatalf("%s does not include resolved kit dependencies:\n%s", file, b)
		}
	}
}

func TestGeneratedFrontendIndexUsesNamespaces(t *testing.T) {
	root := t.TempDir()
	spec := Spec{Name: "article", Table: "articles", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"read"}, Dialects: []string{"sqlite"}}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "generated", "article", "index.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"export * as api from './client';", "export * as react from './react';", "export * as swr from './swr';"} {
		if !strings.Contains(string(b), marker) {
			t.Fatalf("frontend index missing %q: %s", marker, b)
		}
	}
}

func TestGeneratedBulkAdapterBuildsNormalizedRequest(t *testing.T) {
	root := t.TempDir()
	spec := Spec{Name: "article", Table: "articles", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"read"}, Dialects: []string{"sqlite"}}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "generated", "article", "bulk.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"ArticleBulkRequest", "export function ArticleBulkRequest", "new Set(normalized)", "normalized.sort()"} {
		if !strings.Contains(string(b), marker) {
			t.Fatalf("bulk adapter missing %q: %s", marker, b)
		}
	}
}
