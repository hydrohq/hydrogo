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
