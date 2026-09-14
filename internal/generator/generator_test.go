package generator

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
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

func TestGenerateRejectsSymlinkRoot(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "target")
	if err := os.Mkdir(target, 0o750); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(parent, "root-link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	spec := Spec{Name: "pages", Table: "pages_generated", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"pages.read"}, Dialects: []string{"sqlite"}}
	if _, err := Generate(link, spec); err == nil {
		t.Fatal("generator accepted a symlink root")
	}
}

func TestGenerateRejectsSymlinkOutputPaths(t *testing.T) {
	spec := Spec{Name: "pages", Table: "pages_generated", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"pages.read"}, Dialects: []string{"sqlite"}}

	root := t.TempDir()
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	migrationPath := filepath.Join(root, "generated", "pages", "migration.sql")
	outsideFile := filepath.Join(t.TempDir(), "migration.sql")
	if err := os.WriteFile(outsideFile, []byte("outside"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(migrationPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideFile, migrationPath); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, err := Generate(root, spec); err == nil {
		t.Fatal("generator accepted a symlink output file")
	}

	parentRoot := t.TempDir()
	outsideDir := filepath.Join(t.TempDir(), "generated")
	if err := os.Mkdir(outsideDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(parentRoot, "generated")); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, err := Generate(parentRoot, spec); err == nil {
		t.Fatal("generator accepted a symlink output parent")
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
		for _, file := range []string{"service.go", "policy.go", "fixture_test.go", "module.yaml"} {
			if _, err := os.Stat(filepath.Join(root, "generated", spec.Name, file)); err != nil {
				t.Fatalf("%s scaffold %s missing: %v", spec.Name, file, err)
			}
		}
		service, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "service.go"))
		if err != nil || !bytes.Contains(service, []byte("type "+exportName(spec.Name)+"Service struct")) {
			t.Fatalf("%s service scaffold missing: %v", spec.Name, err)
		}
		for _, method := range []string{"Create", "Update", "Delete"} {
			if !bytes.Contains(service, []byte("func (s "+exportName(spec.Name)+"Service) "+method)) {
				t.Fatalf("%s service scaffold missing %s", spec.Name, method)
			}
		}
		policy, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "policy.go"))
		if err != nil || !bytes.Contains(policy, []byte("CanCreate")) {
			t.Fatalf("%s policy scaffold missing: %v", spec.Name, err)
		}
		migration, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "migration.sql"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(migration, []byte("CREATE TABLE "+spec.Table)) || !bytes.Contains(migration, []byte(spec.Fields[0].Name)) {
			t.Fatalf("%s migration does not contain generated fields", spec.Name)
		}
		transport, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "transport.go"))
		if err != nil {
			t.Fatalf("%s transport: %v", spec.Name, err)
		}
		if !bytes.Contains(transport, []byte("/api/v1/"+spec.Name)) {
			t.Fatalf("%s transport does not use versioned API path: %s", spec.Name, transport)
		}
		manifest, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "admin_manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		var collection adminstudio.Collection
		if err := json.Unmarshal(manifest, &collection); err != nil {
			t.Fatalf("%s admin manifest is not JSON: %v", spec.Name, err)
		}
		if collection.ID != spec.Name || len(collection.Fields) != len(spec.Fields) || len(collection.Actions) == 0 {
			t.Fatalf("%s admin manifest = %#v", spec.Name, collection)
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

func TestAdminCollectionUsesExplicitPresentationMetadata(t *testing.T) {
	spec := Spec{
		Name:        "product",
		Table:       "products",
		Permissions: []string{"product.read"},
		Dialects:    []string{"sqlite", "postgres"},
		Fields: []Field{
			{Name: "title", Type: "string", Admin: FieldAdmin{Label: "Product title", VisibleInList: true}},
			{Name: "price_cents", Type: "integer", Admin: FieldAdmin{Label: "Price", Widget: "currency", VisibleInList: true}},
		},
		Admin: Admin{Label: "Catalog", Description: "Sellable products", Actions: []adminstudio.Action{{Label: "Products", Href: "/admin/commerce/products"}}},
	}
	collection, err := AdminCollection(spec)
	if err != nil {
		t.Fatal(err)
	}
	if collection.Label != "Catalog" || collection.Fields[1].Kind != adminstudio.FieldNumber || !collection.Fields[0].VisibleInList || collection.Actions[0].Href != "/admin/commerce/products" {
		t.Fatalf("collection = %#v", collection)
	}
}

func TestLoadSpecUsesVersionedStrictYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testimonial.yaml")
	content := `version: v0
collection:
  name: testimonial
  table: testimonials
  permissions: [testimonial.read]
  dialects: [sqlite, postgres]
  fields:
    - name: quote
      type: string
      admin:
        label: Customer quote
        visible_in_list: true
  admin:
    label: Testimonials
    actions:
      - label: Open testimonials
        href: /admin/testimonials
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	spec, err := LoadSpec(path)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Name != "testimonial" || !spec.Fields[0].Admin.VisibleInList || spec.Admin.Actions[0].Href != "/admin/testimonials" {
		t.Fatalf("loaded spec = %#v", spec)
	}
	if err := os.WriteFile(path, []byte(content+"  unknown: value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSpec(path); err == nil {
		t.Fatal("unknown YAML field was accepted")
	}
}
