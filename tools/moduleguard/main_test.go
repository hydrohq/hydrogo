package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckModulesRequiresDeclaredDirectDependencies(t *testing.T) {
	root := fixtureModules(t, map[string]string{
		"a/a.go": "package a\nimport _ \"" + modulePrefix + "b\"\n",
		"b/b.go": "package b\n",
	})
	a, b := filepath.ToSlash(filepath.Join(root, "a")), filepath.ToSlash(filepath.Join(root, "b"))

	if violations, err := checkModules(root, nil, map[string]map[string]bool{a: {b: true}}); err != nil || len(violations) != 0 {
		t.Fatalf("declared dependency violations=%v err=%v", violations, err)
	}
	violations, err := checkModules(root, nil, map[string]map[string]bool{})
	if err != nil || len(violations) != 1 || !strings.Contains(violations[0], "undeclared module dependency") {
		t.Fatalf("undeclared dependency violations=%v err=%v", violations, err)
	}
}

func TestCheckModulesRejectsDependencyCycles(t *testing.T) {
	root := fixtureModules(t, map[string]string{
		"a/a.go": "package a\nimport _ \"" + modulePrefix + "b\"\n",
		"b/b.go": "package b\nimport _ \"" + modulePrefix + "a\"\n",
	})
	a, b := filepath.ToSlash(filepath.Join(root, "a")), filepath.ToSlash(filepath.Join(root, "b"))
	dependencies := map[string]map[string]bool{a: {b: true}, b: {a: true}}
	violations, err := checkModules(root, nil, dependencies)
	if err != nil || len(violations) != 1 || !strings.Contains(violations[0], "dependency cycle") {
		t.Fatalf("cycle violations=%v err=%v", violations, err)
	}
}

func fixtureModules(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestCheckNoModuleImportsFlagsAnUndeclaredModuleImportFromInternal(t *testing.T) {
	root := fixtureModules(t, map[string]string{
		"pkg/clean.go": "package pkg\nimport \"fmt\"\nvar _ = fmt.Sprint\n",
		"pkg/dirty.go": "package pkg\nimport _ \"" + modulePrefix + "site\"\n",
	})
	violations, err := checkNoModuleImports(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || !strings.Contains(violations[0], "dirty.go") {
		t.Fatalf("violations = %v", violations)
	}
}

func TestCheckNoModuleImportsHonoursExceptionsAndFlagsStaleOnes(t *testing.T) {
	root := fixtureModules(t, map[string]string{
		"pkg/dirty.go": "package pkg\nimport _ \"" + modulePrefix + "site\"\n",
	})
	dirtyPath := filepath.ToSlash(filepath.Join(root, "pkg", "dirty.go"))

	if violations, err := checkNoModuleImports(root, map[string]bool{dirtyPath: true}); err != nil || len(violations) != 0 {
		t.Fatalf("a listed exception must not be flagged: violations=%v err=%v", violations, err)
	}

	cleanRoot := fixtureModules(t, map[string]string{
		"pkg/clean.go": "package pkg\nimport \"fmt\"\nvar _ = fmt.Sprint\n",
	})
	stalePath := filepath.ToSlash(filepath.Join(cleanRoot, "pkg", "clean.go"))
	violations, err := checkNoModuleImports(cleanRoot, map[string]bool{stalePath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || !strings.Contains(violations[0], "no longer imports a module") {
		t.Fatalf("a stale exception (the import was fixed) must be flagged so it gets removed, got %v", violations)
	}
}
