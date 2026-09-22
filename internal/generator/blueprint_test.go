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

func TestLoadBlueprintRejectsUnsupportedVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.yaml")
	data := []byte("version: v1\nname: project\nfields:\n  - name: title\n    type: string\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBlueprint(path); err == nil || !strings.Contains(err.Error(), "unsupported blueprint version") {
		t.Fatalf("expected fail-closed version error, got %v", err)
	}
}

func TestLoadBlueprintAcceptsExplicitV0(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v0.yaml")
	data := []byte("version: v0\nname: project\nfields:\n  - name: title\n    type: string\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBlueprint(path); err != nil {
		t.Fatalf("explicit v0 blueprint rejected: %v", err)
	}
}

func TestLoadBlueprintMapsAuthCompositionFlag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.yaml")
	data := []byte("name: article\nauth: true\nfields:\n  - name: title\n    type: string\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := LoadBlueprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if !spec.Auth {
		t.Fatal("auth flag was not mapped to the versioned Spec")
	}
	root := t.TempDir()
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	module, err := os.ReadFile(filepath.Join(root, "generated", "article", "module.yaml"))
	if err != nil || !strings.Contains(string(module), "auth: true\npermissions:") {
		t.Fatalf("generated module lost auth metadata: %v\n%s", err, module)
	}
	wiring, err := os.ReadFile(filepath.Join(root, "generated", "article", "RUNTIME_WIRING.md"))
	if err != nil || !strings.Contains(string(wiring), "Identity/Session/RBAC") || !strings.Contains(string(wiring), "auth.NewPermissionCatalog(generated.ArticlePermissions()...)") || !strings.Contains(string(wiring), "authSvc.SetPermissionCatalog") {
		t.Fatalf("generated wiring lacks auth review gate: %v", err)
	}
	permissions, err := os.ReadFile(filepath.Join(root, "generated", "article", "permissions.go"))
	if err != nil || !strings.Contains(string(permissions), `"article.read"`) || !strings.Contains(string(permissions), `"article.write"`) {
		t.Fatalf("generated permission catalog is incomplete: %v\n%s", err, permissions)
	}
	runtime, err := os.ReadFile(filepath.Join(root, "generated", "article", "runtime.go"))
	if err != nil || !strings.Contains(string(runtime), "authenticated func(*http.Request) bool") || !strings.Contains(string(runtime), "http.StatusUnauthorized") {
		t.Fatalf("generated auth route guard missing: %v", err)
	}
}

func TestLoadBlueprintMapsBoundedWorkflow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workflow.yaml")
	data := []byte("name: article\nworkflow:\n  states: [draft, review, published]\n  transitions:\n    draft: [review]\n    review: [published, draft]\nfields:\n  - name: title\n    type: string\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := LoadBlueprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Workflow == nil || len(spec.Workflow.States) != 3 || len(spec.Workflow.Transitions["review"]) != 2 {
		t.Fatalf("workflow metadata was not mapped: %#v", spec.Workflow)
	}
	root := t.TempDir()
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	workflow, err := os.ReadFile(filepath.Join(root, "generated", "article", "workflow.json"))
	if err != nil || !strings.Contains(string(workflow), `"enabled": true`) || !strings.Contains(string(workflow), `"published"`) {
		t.Fatalf("generated workflow contract missing: %v\n%s", err, workflow)
	}
}

func TestLoadBlueprintRejectsInvalidWorkflowTransition(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid-workflow.yaml")
	data := []byte("name: article\nworkflow:\n  states: [draft]\n  transitions:\n    draft: [missing]\nfields:\n  - name: title\n    type: string\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBlueprint(path); err == nil {
		t.Fatal("workflow transition to unknown state was accepted")
	}
}

func TestLoadBlueprintMapsRelationsIntoManifestAndAdminForm(t *testing.T) {
	path := filepath.Join(t.TempDir(), "relations.yaml")
	data := []byte("name: article\nfields:\n  - name: author_id\n    type: relation\n    required: true\n    relation:\n      collection: author\n      cardinality: one\n  - name: tag_ids\n    type: relation\n    required: false\n    relation:\n      collection: tag\n      cardinality: many\n      max_items: 12\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := LoadBlueprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Fields[0].Relation == nil || spec.Fields[0].Relation.Collection != "author" || spec.Fields[0].Admin.Widget != "relation" || spec.Fields[0].Nullable {
		t.Fatalf("single relation mapping is incomplete: %#v", spec.Fields[0])
	}
	root := t.TempDir()
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(root, "generated", "article", "relations.json"))
	if err != nil || !strings.Contains(string(manifest), `"max_items": 12`) || !strings.Contains(string(manifest), `"storage": "edition_managed_collection"`) {
		t.Fatalf("relation metadata missing: %v\n%s", err, manifest)
	}
	form, err := os.ReadFile(filepath.Join(root, "generated", "article", "admin", "form.templ"))
	if err != nil || !strings.Contains(string(form), `RelationOptions["author_id"]`) {
		t.Fatalf("relation selector metadata missing from admin form: %v", err)
	}
	wiring, err := os.ReadFile(filepath.Join(root, "generated", "article", "RUNTIME_WIRING.md"))
	if err != nil || !strings.Contains(string(wiring), "authorized Edition resolver") {
		t.Fatalf("relation resolver review gate missing: %v", err)
	}
}

func TestLoadBlueprintRejectsUnboundedRelation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid-relation.yaml")
	data := []byte("name: article\nfields:\n  - name: tag_ids\n    type: relation\n    relation:\n      collection: tag\n      cardinality: many\n      max_items: 101\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBlueprint(path); err == nil {
		t.Fatal("relation above the item budget was accepted")
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

func TestLoadBlueprintMapsValidationContract(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile.yaml")
	content := []byte("name: profile\nfields:\n  - name: email\n    type: string\n    required: true\n    validation:\n      min_length: 5\n      max_length: 120\n      format: email\n      unique: true\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	spec, err := LoadBlueprint(path)
	if err != nil {
		t.Fatal(err)
	}
	validation := spec.Fields[0].Validation
	if validation.MinLength == nil || *validation.MinLength != 5 || validation.MaxLength == nil || *validation.MaxLength != 120 || validation.Format != "email" || !validation.Unique {
		t.Fatalf("validation = %#v", validation)
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
