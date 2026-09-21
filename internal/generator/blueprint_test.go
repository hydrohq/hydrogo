package generator

import (
	"os"
	"path/filepath"
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
