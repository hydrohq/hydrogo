package generator

import (
	"bytes"
	"errors"
	"os"
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
