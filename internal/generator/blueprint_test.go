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
