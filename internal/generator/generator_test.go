package generator

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGenerateIsDeterministicAndPreservesCustom(t *testing.T) {
	root := t.TempDir()
	customPath := filepath.Join(root, "custom", "validation.go")
	if err := os.MkdirAll(filepath.Dir(customPath), 0o755); err != nil {
		t.Fatal(err)
	}
	custom := []byte("package custom\n\nvar Rule = true\n")
	if err := os.WriteFile(customPath, custom, 0o644); err != nil {
		t.Fatal(err)
	}
	spec := Spec{Name: "testimonial", Table: "testimonials", Fields: []Field{{Name: "title", Type: "string"}, {Name: "published", Type: "bool"}}, Permissions: []string{"testimonial.read", "testimonial.write"}, Dialects: []string{"postgres", "sqlite"}}
	first, err := Generate(root, spec)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, "generated", "testimonial", "model.go"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(root, "generated", "testimonial", "model.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || first.SpecSHA256 == "" {
		t.Fatal("generator output is not deterministic")
	}
	gotCustom, _ := os.ReadFile(customPath)
	if !bytes.Equal(custom, gotCustom) {
		t.Fatal("custom file was changed")
	}
}

func TestValidateDeniesSecretsAndDestructiveMigrations(t *testing.T) {
	base := Spec{Name: "profile", Table: "profiles", Fields: []Field{{Name: "email", Type: "string"}}, Permissions: []string{"profile.read"}, Dialects: []string{"sqlite"}}
	secret := base
	secret.Fields = []Field{{Name: "password_hash", Type: "string"}}
	if !errors.Is(Validate(secret), ErrInvalidSpec) {
		t.Fatal("secret field was accepted")
	}
	destructive := base
	destructive.Destructive = true
	if !errors.Is(Validate(destructive), ErrDestructiveSpec) {
		t.Fatal("destructive spec was accepted")
	}
}

func TestGeneratePreservesPublishedMigration(t *testing.T) {
	root := t.TempDir()
	spec := Spec{Name: "pages", Table: "pages_generated", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"pages.read"}, Dialects: []string{"sqlite", "postgres"}}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	migrationPath := filepath.Join(root, "generated", "pages", "migration.sql")
	original, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(root, spec); err != nil {
		t.Fatalf("identical generation rejected: %v", err)
	}
	if err := os.WriteFile(migrationPath, append(original, []byte("-- published\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(root, spec); !errors.Is(err, ErrPublishedMigration) {
		t.Fatalf("changed published migration error = %v", err)
	}
}

func TestCoreCollectionFixturesGenerateDeterministically(t *testing.T) {
	for _, spec := range []Spec{
		{Name: "pages", Table: "pages", Fields: []Field{{Name: "title", Type: "string"}, {Name: "status", Type: "string"}}, Permissions: []string{"pages.read", "pages.write"}, Dialects: []string{"sqlite", "postgres"}},
		{Name: "news_items", Table: "news_items", Fields: []Field{{Name: "headline", Type: "string"}, {Name: "scheduled", Type: "bool"}}, Permissions: []string{"news.read", "news.write"}, Dialects: []string{"sqlite", "postgres"}},
	} {
		root := t.TempDir()
		first, err := Generate(root, spec)
		if err != nil {
			t.Fatalf("%s: %v", spec.Name, err)
		}
		before, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "model.go"))
		if err != nil {
			t.Fatal(err)
		}
		second, err := Generate(root, spec)
		if err != nil {
			t.Fatalf("%s second generation: %v", spec.Name, err)
		}
		after, _ := os.ReadFile(filepath.Join(root, "generated", spec.Name, "model.go"))
		if first.SpecSHA256 != second.SpecSHA256 || !bytes.Equal(before, after) {
			t.Fatalf("%s output is not deterministic", spec.Name)
		}
		repository, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "repository.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, method := range []string{"func (r " + exportName(spec.Name) + "Repository) Get", "func (r " + exportName(spec.Name) + "Repository) Create", "func (r " + exportName(spec.Name) + "Repository) Update", "func (r " + exportName(spec.Name) + "Repository) Delete"} {
			if !bytes.Contains(repository, []byte(method)) {
				t.Fatalf("%s repository missing %q", spec.Name, method)
			}
		}
		migration, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "migration.sql"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(migration, []byte("CREATE TABLE "+spec.Table)) || !bytes.Contains(migration, []byte(spec.Fields[0].Name)) {
			t.Fatalf("%s migration does not contain generated fields", spec.Name)
		}
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module generatedfixture\n\ngo 1.27\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		command := exec.Command("go", "test", "./...")
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("%s generated fixture does not compile: %v\n%s", spec.Name, err, output)
		}
	}
}
