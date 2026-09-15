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
