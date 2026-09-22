package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBlueprintMapsRequiredFieldsAndDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "project.yaml")
	data := []byte("name: project\nfields:\n  - name: title\n    type: string\n    required: true\n  - name: status\n    type: enum\n    required: false\n    values: [draft, active]\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := LoadBlueprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Name != "project" || spec.Table != "projects" || len(spec.Fields) != 2 {
		t.Fatalf("unexpected spec: %#v", spec)
	}
	if spec.Fields[0].Nullable || !spec.Fields[1].Nullable || len(spec.Permissions) != 2 {
		t.Fatalf("required/permissions mapping incorrect: %#v", spec)
	}
}

func TestLoadBlueprintMapsAdminAndUIHintsWithoutChangingPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "article.yaml")
	data := []byte("name: article\nadmin:\n  label: Articles\n  icon: newspaper\n  group: content\nfields:\n  - name: title\n    type: string\n    required: true\n    ui:\n      list: true\n      searchable: true\n      filterable: true\n      widget: text\n  - name: body\n    type: text\n    required: false\n    ui:\n      widget: block-editor\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := LoadBlueprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Admin.Icon != "newspaper" || spec.Admin.Group != "content" || spec.Admin.Label != "Articles" {
		t.Fatalf("admin metadata was not mapped: %#v", spec.Admin)
	}
	if !spec.Fields[0].Admin.VisibleInList || !spec.Fields[0].Admin.Searchable || !spec.Fields[0].Admin.Filterable || spec.Fields[0].Admin.Widget != "text" {
		t.Fatalf("first field UI metadata was not mapped: %#v", spec.Fields[0].Admin)
	}
	if spec.Fields[0].Type != "string" || spec.Fields[1].Type != "text" || spec.Fields[1].Admin.Widget != "block-editor" {
		t.Fatalf("UI metadata changed persistence mapping: %#v", spec.Fields)
	}
}

func TestLoadBlueprintValuesStillDefaultToSelectWidget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "project.yaml")
	data := []byte("name: project\nfields:\n  - name: status\n    type: enum\n    values: [draft, active]\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := LoadBlueprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Fields[0].Admin.Widget != "select" || spec.Fields[0].Type != "string" {
		t.Fatalf("enum values should retain select default: %#v", spec.Fields[0])
	}
}

func TestLoadBlueprintSupportsGlobalResourceType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.yaml")
	content := []byte("type: global\nname: settings\nfields:\n  - name: site_title\n    type: string\n    required: true\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	spec, err := LoadBlueprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Type != "global" {
		t.Fatalf("type = %q, want global", spec.Type)
	}
}

func TestGeneratedMigrationMetadataIsDialectAwareAndNonDestructive(t *testing.T) {
	root := t.TempDir()
	spec := Spec{Name: "article", Table: "articles", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"article.read"}, Dialects: []string{"sqlite", "postgres"}}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	metadata, err := os.ReadFile(filepath.Join(root, "generated", "article", "migration.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(metadata)
	for _, required := range []string{"sqlite: expand", "postgres: expand", "sqlite: true", "postgres: true", "rollback:", "checksum_sha256:"} {
		if !strings.Contains(text, required) {
			t.Fatalf("migration metadata missing %q: %s", required, text)
		}
	}
}
