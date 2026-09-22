package generator

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	_ "modernc.org/sqlite"
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

func TestGeneratedSpecIsCanonicalAcrossInputOrder(t *testing.T) {
	root := t.TempDir()
	first := Spec{Name: "article", Table: "articles", Fields: []Field{{Name: "title", Type: "string"}, {Name: "published", Type: "bool"}}, Permissions: []string{"article.write", "article.read"}, Dialects: []string{"postgres", "sqlite"}, Kits: []string{"seo", "localization"}}
	if _, err := Generate(root, first); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, "generated", "article", "spec.json"))
	if err != nil {
		t.Fatal(err)
	}
	second := Spec{Name: first.Name, Table: first.Table, Fields: []Field{{Name: "published", Type: "bool"}, {Name: "title", Type: "string"}}, Permissions: []string{"article.read", "article.write"}, Dialects: []string{"sqlite", "postgres"}, Kits: []string{"localization", "seo"}}
	if _, err := Generate(root, second); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(root, "generated", "article", "spec.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("spec.json changed for semantically equivalent input order")
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

func TestValidateDeniesUnsafeOrDuplicatePermissions(t *testing.T) {
	base := Spec{Name: "profile", Table: "profiles", Fields: []Field{{Name: "email", Type: "string"}}, Permissions: []string{"profile.read"}, Dialects: []string{"sqlite"}}
	for _, permissions := range [][]string{{"Profile.read"}, {"profile/../read"}, {"profile.read", "profile.read"}} {
		spec := base
		spec.Permissions = permissions
		if !errors.Is(Validate(spec), ErrInvalidSpec) {
			t.Fatalf("unsafe permissions accepted: %#v", permissions)
		}
	}
}

func TestValidateRequiresExplicitRelationTargetAndCardinality(t *testing.T) {
	base := Spec{Name: "article", Table: "articles", Fields: []Field{{Name: "author", Type: "relation"}}, Permissions: []string{"article.read"}, Dialects: []string{"sqlite", "postgres"}}
	if !errors.Is(Validate(base), ErrInvalidSpec) {
		t.Fatal("relation without metadata was accepted")
	}
	base.Fields[0].Relation = &Relation{Collection: "authors", Cardinality: "one"}
	if err := Validate(base); err != nil {
		t.Fatalf("valid relation rejected: %v", err)
	}
	base.Fields[0].Relation.Cardinality = "many_to_many"
	if !errors.Is(Validate(base), ErrInvalidSpec) {
		t.Fatal("unsupported relation cardinality was accepted")
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
	titleMin, titleMax := 2, 2
	for _, spec := range []Spec{
		{Name: "pages", Table: "pages", Fields: []Field{{Name: "title", Type: "string", Validation: Validation{MinLength: &titleMin, MaxLength: &titleMax}, Admin: FieldAdmin{VisibleInList: true, Searchable: true}}, {Name: "status", Type: "string", Admin: FieldAdmin{Filterable: true, Options: []string{"draft", "published"}}}, {Name: "email", Type: "string", Validation: Validation{Format: "email"}}, {Name: "published_on", Type: "string", Validation: Validation{Format: "date"}}, {Name: "updated_at", Type: "string", Validation: Validation{Format: "datetime"}}, {Name: "canonical_url", Type: "string", Validation: Validation{Format: "url"}}}, Permissions: []string{"pages.read", "pages.write"}, Dialects: []string{"sqlite", "postgres"}},
		{Name: "news_items", Table: "news_items", Fields: []Field{{Name: "headline", Type: "string", Admin: FieldAdmin{VisibleInList: true, Searchable: true}}, {Name: "scheduled", Type: "bool"}}, Permissions: []string{"news.read", "news.write"}, Dialects: []string{"sqlite", "postgres"}},
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
		for _, file := range []string{"spec.json", "service.go", "policy.go", "runtime.go", "workflow.json", "relations.json", "fixture_test.go", "module.yaml", "REVIEW.md", "RUNTIME_WIRING.md", "form.html", "form.schema.json", "table.html", "openapi.json", "client.ts", "query.ts", "bulk.ts", "import-export.ts", "upload.ts", "react.ts", "swr.ts", "htmx.ts", "vanilla.ts", "kits.ts", "index.ts", "admin/list.templ", "admin/form.templ", "admin/views.json"} {
			if _, err := os.Stat(filepath.Join(root, "generated", spec.Name, file)); err != nil {
				t.Fatalf("%s scaffold %s missing: %v", spec.Name, file, err)
			}
		}
		runtime, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "runtime.go"))
		if err != nil || !bytes.Contains(runtime, []byte("Register"+exportName(spec.Name)+"Route")) || !bytes.Contains(runtime, []byte("policy")) {
			t.Fatalf("%s runtime route registration missing: %v", spec.Name, err)
		}
		wiring, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "RUNTIME_WIRING.md"))
		if err != nil || !bytes.Contains(wiring, []byte("/api/v1/"+spec.Name)) || !bytes.Contains(wiring, []byte("Compact profile")) {
			t.Fatalf("%s runtime wiring checklist missing: %v", spec.Name, err)
		}
		for _, file := range []string{"form.html", "form.schema.json", "table.html", "openapi.json", "client.ts", "query.ts", "bulk.ts", "import-export.ts", "upload.ts", "react.ts", "swr.ts", "htmx.ts", "vanilla.ts", "index.ts"} {
			content, readErr := os.ReadFile(filepath.Join(root, "generated", spec.Name, file))
			if readErr != nil || len(content) == 0 {
				t.Fatalf("generated %s missing: %v", file, readErr)
			}
		}
		listTemplate, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "admin", "list.templ"))
		if err != nil || !bytes.Contains(listTemplate, []byte("ui.Table")) || !bytes.Contains(listTemplate, []byte("delay:300ms")) || bytes.Contains(listTemplate, []byte("<!--")) {
			t.Fatalf("%s generated list template missing typed HTMX contract: %v", spec.Name, err)
		}
		formTemplate, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "admin", "form.templ"))
		if err != nil || !bytes.Contains(formTemplate, []byte("ui.Form")) || !bytes.Contains(formTemplate, []byte("CSRFToken")) || bytes.Contains(formTemplate, []byte("<!--")) {
			t.Fatalf("%s generated form template missing typed CSRF contract: %v", spec.Name, err)
		}
		for file, marker := range map[string]string{"react.ts": "create" + exportName(spec.Name) + "Query", "swr.ts": "create" + exportName(spec.Name) + "SWR", "htmx.ts": "hx-get", "vanilla.ts": "create" + exportName(spec.Name) + "Client"} {
			content, readErr := os.ReadFile(filepath.Join(root, "generated", spec.Name, file))
			if readErr != nil || !bytes.Contains(content, []byte(marker)) {
				t.Fatalf("%s adapter %s missing marker %q: %v", spec.Name, file, marker, readErr)
			}
		}
		client, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "client.ts"))
		if err != nil || !bytes.Contains(client, []byte("class HydroAPIError")) || !bytes.Contains(client, []byte("parseHydroError")) || !bytes.Contains(client, []byte("getReader")) || !bytes.Contains(client, []byte("MAX_RESPONSE_BYTES")) {
			t.Fatalf("%s generated client missing bounded typed errors: %v", spec.Name, err)
		}
		openapi, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "openapi.json"))
		if err != nil || !bytes.Contains(openapi, []byte("HydroAPIError")) || !bytes.Contains(openapi, []byte("page_size")) || !bytes.Contains(openapi, []byte("429")) {
			t.Fatalf("%s OpenAPI missing query/error contract: %v", spec.Name, err)
		}
		react, err := os.ReadFile(filepath.Join(root, "generated", spec.Name, "react.ts"))
		if err != nil || !bytes.Contains(react, []byte("QueryOptions")) || !bytes.Contains(react, []byte("QueryKey")) {
			t.Fatalf("%s React adapter must expose typed query options and keys: %v", spec.Name, err)
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
		repository, err = os.ReadFile(filepath.Join(root, "generated", spec.Name, "repository.go"))
		if err != nil || !bytes.Contains(repository, []byte("func (r "+exportName(spec.Name)+"Repository) List")) || !bytes.Contains(repository, []byte("invalid pagination")) {
			t.Fatalf("%s repository pagination scaffold missing: %v", spec.Name, err)
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
		if !bytes.Contains(transport, []byte("ParseResourceQuery")) || !bytes.Contains(transport, []byte("invalid query")) {
			t.Fatalf("%s transport does not bind the public query contract", spec.Name)
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
		repoRoot, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		repoRoot, err = filepath.Abs(filepath.Join(repoRoot, "..", ".."))
		if err != nil {
			t.Fatal(err)
		}
		moduleFile := fmt.Sprintf("module generatedfixture\n\ngo 1.27\n\nrequire github.com/ardaozyildirim/hydrogo v0.0.0\nreplace github.com/ardaozyildirim/hydrogo => %s\n", filepath.ToSlash(repoRoot))
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(moduleFile), 0o644); err != nil {
			t.Fatal(err)
		}
		if spec.Name == "pages" {
			validationTest := `package generated
import "testing"
func TestGeneratedValidationRuntime(t *testing.T) {
    item := Pages{ID:"p1", Title:"漢字", Status:"draft", Email:"a@example.com", PublishedOn:"2026-09-23", UpdatedAt:"2026-09-23T12:00:00Z", CanonicalUrl:"https://example.com"}
    if err := item.Validate(); err != nil { t.Fatalf("valid Unicode/formats rejected: %v", err) }
    item.Title = "漢字語"
    if err := item.Validate(); err == nil { t.Fatal("Unicode max length was not enforced") }
    item.Title = "漢字"
    invalid := []struct{name string; change func(*Pages)}{
        {"email", func(p *Pages){ p.Email = "display <a@example.com>" }},
        {"date", func(p *Pages){ p.PublishedOn = "2026-02-30" }},
        {"datetime", func(p *Pages){ p.UpdatedAt = "2026-09-23 12:00:00" }},
        {"url", func(p *Pages){ p.CanonicalUrl = "ftp://example.com/file" }},
    }
    for _, tc := range invalid { t.Run(tc.name, func(t *testing.T){ candidate := item; tc.change(&candidate); if err := candidate.Validate(); err == nil { t.Fatal("invalid value accepted") } }) }
}
`
			if err := os.WriteFile(filepath.Join(root, "generated", spec.Name, "validation_runtime_test.go"), []byte(validationTest), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		command := exec.Command("go", "test", "-mod=mod", "./...")
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
			{Name: "title", Type: "string", Admin: FieldAdmin{Label: "Product title", VisibleInList: true, Searchable: true}},
			{Name: "price_cents", Type: "integer", Admin: FieldAdmin{Label: "Price", Widget: "currency", VisibleInList: true, Filterable: true}},
		},
		Admin: Admin{Label: "Catalog", Description: "Sellable products", Actions: []adminstudio.Action{{Label: "Products", Href: "/admin/commerce/products"}}},
	}
	collection, err := AdminCollection(spec)
	if err != nil {
		t.Fatal(err)
	}
	if collection.Label != "Catalog" || collection.Fields[1].Kind != adminstudio.FieldNumber || !collection.Fields[0].VisibleInList || !collection.Fields[0].Searchable || !collection.Fields[1].Filterable || collection.Actions[0].Href != "/admin/commerce/products" {
		t.Fatalf("collection = %#v", collection)
	}
}

func TestGeneratedAdminTemplatesUseBlockEditorContract(t *testing.T) {
	root := t.TempDir()
	spec := Spec{Name: "article", Table: "articles", Fields: []Field{{Name: "body", Type: "text", Admin: FieldAdmin{Widget: "block-editor"}}}, Permissions: []string{"article.read"}, Dialects: []string{"sqlite", "postgres"}}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "generated", "article", "admin", "form.templ"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, marker := range []string{"ui.BlockEditor", `PreviewURL: "/api/preview"`, "Nonce: view.Nonce"} {
		if !strings.Contains(text, marker) {
			t.Fatalf("generated form missing %q: %s", marker, text)
		}
	}
}

func TestGeneratedAdminTemplatesUseTypedRelationAndMediaContracts(t *testing.T) {
	root := t.TempDir()
	spec := Spec{
		Name: "task", Table: "tasks", Fields: []Field{
			{Name: "project_id", Type: "relation", Relation: &Relation{Collection: "project", Cardinality: "one"}, Admin: FieldAdmin{Widget: "relation"}},
			{Name: "attachment", Type: "string", Admin: FieldAdmin{Widget: "media"}},
		},
		Permissions: []string{"task.read"}, Dialects: []string{"sqlite", "postgres"},
	}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "generated", "task", "admin", "form.templ"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, marker := range []string{"RelationOptions map[string][]string", "ui.Select", "view.RelationOptions[\"project_id\"]", "ui.MediaPicker"} {
		if !strings.Contains(text, marker) {
			t.Fatalf("generated form missing %q: %s", marker, text)
		}
	}
}

func TestGeneratedManyRelationUsesNormalizedBoundedStorage(t *testing.T) {
	root := t.TempDir()
	spec := Spec{
		Name: "record", Table: "records", TenantScoped: true,
		Fields: []Field{
			{Name: "tag_ids", Type: "relation", Relation: &Relation{Collection: "tag", Cardinality: "many", MaxItems: 20}},
			{Name: "title", Type: "string"},
		},
		Permissions: []string{"record.read"}, Dialects: []string{"sqlite", "postgres"},
	}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile(filepath.Join(root, "generated", "record", "migration.sql"))
	if err != nil {
		t.Fatal(err)
	}
	relations, err := os.ReadFile(filepath.Join(root, "generated", "record", "relations_store.go"))
	if err != nil {
		t.Fatal(err)
	}
	repository, err := os.ReadFile(filepath.Join(root, "generated", "record", "repository.go"))
	if err != nil {
		t.Fatal(err)
	}
	model, err := os.ReadFile(filepath.Join(root, "generated", "record", "model.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		"CREATE TABLE records_tag_ids_rel (tenant_id TEXT NOT NULL",
		"PRIMARY KEY (tenant_id, resource_id, related_id)",
		"CREATE INDEX records_tag_ids_rel_related_idx",
	} {
		if !strings.Contains(string(migration), marker) {
			t.Fatalf("generated migration missing %q: %s", marker, migration)
		}
	}
	if strings.Contains(string(migration), "tag_ids TEXT") {
		t.Fatalf("many relation also emitted as scalar column: %s", migration)
	}
	for _, marker := range []string{"ReplaceTagIdsTx", "ListTagIds", "maxItems int", "tenant_id = "} {
		if !strings.Contains(string(relations), marker) {
			t.Fatalf("generated relation store missing %q: %s", marker, relations)
		}
	}
	if !strings.Contains(string(model), "TagIds") || !strings.Contains(string(model), "[]string") || strings.Contains(string(repository), "tag_ids") {
		t.Fatalf("many-valued field must remain typed but be excluded from scalar CRUD SQL: model=%s repository=%s", model, repository)
	}
}

func TestGeneratedServiceBindsValidationContract(t *testing.T) {
	root := t.TempDir()
	minLength := 3
	spec := Spec{Name: "profile", Table: "profiles", Fields: []Field{{Name: "name", Type: "string", Validation: Validation{MinLength: &minLength}}}, Permissions: []string{"profile.read"}, Dialects: []string{"sqlite", "postgres"}}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	validation, err := os.ReadFile(filepath.Join(root, "generated", "profile", "validation.go"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := os.ReadFile(filepath.Join(root, "generated", "profile", "service.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(validation, []byte("utf8.RuneCountInString(item.Name) < 3")) || !bytes.Contains(service, []byte("item.Validate()")) {
		t.Fatalf("generated validation was not bound: validation=%s service=%s", validation, service)
	}
}

func TestGeneratedValidationExecutesDeclaredFormats(t *testing.T) {
	spec := Spec{Name: "profile", Table: "profiles", Fields: []Field{
		{Name: "email", Type: "string", Validation: Validation{Format: "email"}},
		{Name: "birthday", Type: "string", Validation: Validation{Format: "date"}},
		{Name: "created_at", Type: "string", Validation: Validation{Format: "datetime"}},
		{Name: "website", Type: "text", Validation: Validation{Format: "url"}},
	}, Permissions: []string{"profile.read"}, Dialects: []string{"sqlite", "postgres"}}
	if err := Validate(spec); err != nil {
		t.Fatal(err)
	}
	validation := renderValidation(spec)
	for _, marker := range []string{"mail.ParseAddress(item.Email)", "time.Parse(\"2006-01-02\", item.Birthday)", "time.Parse(time.RFC3339, item.CreatedAt)", "url.ParseRequestURI(item.Website)"} {
		if !strings.Contains(validation, marker) {
			t.Fatalf("format validation missing %q: %s", marker, validation)
		}
	}
	if err := Validate(Spec{Name: "profile", Table: "profiles", Fields: []Field{{Name: "age", Type: "integer", Validation: Validation{Format: "date"}}}, Dialects: []string{"sqlite"}}); err == nil {
		t.Fatal("non-text field accepted a text format")
	}
}

func TestGeneratedAdminFormsExposeValidationHints(t *testing.T) {
	minLength, maxLength := 3, 80
	spec := Spec{Name: "profile", Fields: []Field{
		{Name: "email", Type: "string", Validation: Validation{Format: "email", MinLength: &minLength, MaxLength: &maxLength}},
		{Name: "bio", Type: "text", Admin: FieldAdmin{Widget: "textarea"}, Validation: Validation{MinLength: &minLength, MaxLength: &maxLength}},
	}}
	form := renderAdminFormTempl(spec)
	for _, marker := range []string{`Type: "email"`, `MinLength: 3`, `MaxLength: 80`, `HasMinLength: true`, `HasMaxLength: true`} {
		if !strings.Contains(form, marker) {
			t.Fatalf("generated form is missing %q: %s", marker, form)
		}
	}
}

func TestGeneratedGlobalServiceNormalizesSingletonID(t *testing.T) {
	root := t.TempDir()
	spec := Spec{Name: "settings", Type: "global", Table: "settings", Fields: []Field{{Name: "site_title", Type: "string"}}, Permissions: []string{"settings.read"}, Dialects: []string{"sqlite", "postgres"}}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	validation, err := os.ReadFile(filepath.Join(root, "generated", "settings", "validation.go"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := os.ReadFile(filepath.Join(root, "generated", "settings", "service.go"))
	if err != nil {
		t.Fatal(err)
	}
	transport, err := os.ReadFile(filepath.Join(root, "generated", "settings", "transport.go"))
	if err != nil {
		t.Fatal(err)
	}
	listTemplate, err := os.ReadFile(filepath.Join(root, "generated", "settings", "admin", "list.templ"))
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile(filepath.Join(root, "generated", "settings", "migration.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(validation, []byte(`const SettingsGlobalID = "global"`)) || !bytes.Contains(validation, []byte("item.ID = SettingsGlobalID")) || !bytes.Contains(service, []byte("item.normalize()")) || !bytes.Contains(transport, []byte(`id = SettingsGlobalID`)) {
		t.Fatalf("global singleton normalization missing: validation=%s service=%s", validation, service)
	}
	if !bytes.Contains(migration, []byte("CHECK (id = 'global')")) {
		t.Fatalf("global migration must enforce singleton id: %s", migration)
	}
	if !bytes.Contains(listTemplate, []byte("tekil bir ayardır")) || bytes.Contains(listTemplate, []byte("@ui.Table")) {
		t.Fatalf("global admin list must not render collection table: %s", listTemplate)
	}
}

func TestGeneratedGlobalMigrationEnforcesSQLiteSingleton(t *testing.T) {
	root := t.TempDir()
	spec := Spec{Name: "site_settings", Type: "global", Table: "site_settings", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"settings.read"}, Dialects: []string{"sqlite", "postgres"}}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile(filepath.Join(root, "generated", "site_settings", "migration.sql"))
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(string(migration)); err != nil {
		t.Fatalf("generated global migration must execute on sqlite: %v", err)
	}
	if _, err := db.Exec("INSERT INTO site_settings (id, title) VALUES ('global', 'ok')"); err != nil {
		t.Fatalf("stable global id insert failed: %v", err)
	}
	if _, err := db.Exec("INSERT INTO site_settings (id, title) VALUES ('other', 'bad')"); err == nil {
		t.Fatal("non-global singleton id unexpectedly accepted")
	}
}

func TestGeneratedTenantScopedResourceBindsTenantContract(t *testing.T) {
	root := t.TempDir()
	spec := Spec{Name: "invoice", Table: "invoices", TenantScoped: true, Fields: []Field{{Name: "total", Type: "decimal"}}, Permissions: []string{"invoice.read"}, Dialects: []string{"sqlite", "postgres"}}
	if _, err := Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	model, err := os.ReadFile(filepath.Join(root, "generated", "invoice", "model.go"))
	if err != nil {
		t.Fatal(err)
	}
	repository, err := os.ReadFile(filepath.Join(root, "generated", "invoice", "repository.go"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := os.ReadFile(filepath.Join(root, "generated", "invoice", "service.go"))
	if err != nil {
		t.Fatal(err)
	}
	transport, err := os.ReadFile(filepath.Join(root, "generated", "invoice", "transport.go"))
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile(filepath.Join(root, "generated", "invoice", "migration.sql"))
	if err != nil {
		t.Fatal(err)
	}
	wiring, err := os.ReadFile(filepath.Join(root, "generated", "invoice", "RUNTIME_WIRING.md"))
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name    string
		data    []byte
		markers []string
	}{
		{"model", model, []string{"TenantID string `json:\"tenant_id\"`"}},
		{"repository", repository, []string{"Get(ctx context.Context, tenantID, id string)", "tenant_id =", "List(ctx context.Context, tenantID string"}},
		{"service", service, []string{"Create(ctx context.Context, tenantID string", "Update(ctx context.Context, tenantID string"}},
		{"transport", transport, []string{"ResolveTenant", "h.ResolveTenant", "Service.List(r.Context(), tenantID"}},
		{"migration", migration, []string{"tenant_id TEXT NOT NULL", "CREATE INDEX invoices_tenant_id_idx ON invoices (tenant_id, id)"}},
		{"runtime wiring", wiring, []string{"tenant.VerifyProvisioningRLSForTables"}},
	}
	for _, check := range checks {
		for _, marker := range check.markers {
			if !bytes.Contains(check.data, []byte(marker)) {
				t.Fatalf("tenant scoped %s missing %q: %s", check.name, marker, check.data)
			}
		}
	}
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot, err = filepath.Abs(filepath.Join(repoRoot, "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	moduleFile := fmt.Sprintf("module tenantfixture\n\ngo 1.27\n\nrequire github.com/ardaozyildirim/hydrogo v0.0.0\nreplace github.com/ardaozyildirim/hydrogo => %s\n", filepath.ToSlash(repoRoot))
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(moduleFile), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("tenant generated fixture does not compile: %v\n%s", err, output)
	}
}

func TestTenantIndexNameStaysWithinDialectIdentifierLimit(t *testing.T) {
	table := strings.Repeat("a", 63)
	name := tenantIndexName(table)
	if len(name) != 63 || !strings.HasPrefix(name, table[:54]+"_") {
		t.Fatalf("tenant index name is not bounded: %q (%d)", name, len(name))
	}
}

func TestLoadSpecUsesVersionedStrictYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testimonial.yaml")
	content := `version: v0
collection:
  name: testimonial
  table: testimonials
  tenant_scoped: true
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
	if spec.Name != "testimonial" || !spec.TenantScoped || !spec.Fields[0].Admin.VisibleInList || spec.Admin.Actions[0].Href != "/admin/testimonials" {
		t.Fatalf("loaded spec = %#v", spec)
	}
	if err := os.WriteFile(path, []byte(content+"  unknown: value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSpec(path); err == nil {
		t.Fatal("unknown YAML field was accepted")
	}
}

func TestModuleStarterFixtureLoadsAndValidates(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "module-starter", "collection.yaml")
	spec, err := LoadSpec(path)
	if err != nil {
		t.Fatalf("starter fixture: %v", err)
	}
	if err := Validate(spec); err != nil {
		t.Fatalf("starter fixture validation: %v", err)
	}
	if spec.Name != "testimonial" || len(spec.Fields) != 2 {
		t.Fatalf("starter fixture spec = %#v", spec)
	}
}

func TestGeneratedPermissionCatalogIsDeterministic(t *testing.T) {
	generated := renderPermissions(Spec{Name: "article", Permissions: []string{"article.write", "article.read"}})
	readAt, writeAt := strings.Index(generated, `"article.read"`), strings.Index(generated, `"article.write"`)
	if !strings.Contains(generated, "func ArticlePermissions() []string") || readAt < 0 || writeAt < 0 || readAt > writeAt {
		t.Fatalf("generated permission catalog is missing or not sorted: %s", generated)
	}
}

func TestAuthEnabledRuntimeWiringRegistersGeneratedPermissionCatalog(t *testing.T) {
	spec := Spec{Name: "article", Auth: true, Table: "articles", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"article.read", "article.write"}, Dialects: []string{"sqlite"}}
	wiring := renderRuntimeWiring(spec)
	for _, marker := range []string{"auth.NewPermissionCatalog(generated.ArticlePermissions()...)", "authSvc.SetPermissionCatalog", "generated permissions do not receive role grants implicitly", "unknown names deny"} {
		if !strings.Contains(wiring, marker) {
			t.Errorf("auth runtime wiring missing %q", marker)
		}
	}
}

func TestResourcePermissionNamesMustBeCanonical(t *testing.T) {
	spec := Spec{Name: "article", Table: "articles", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{" article.read"}, Dialects: []string{"sqlite"}}
	if err := Validate(spec); err == nil {
		t.Fatal("whitespace-padded permission identifier was accepted")
	}
}

func TestResourcePermissionListHasBoundedSize(t *testing.T) {
	permissions := make([]string, maxResourcePermissions+1)
	for i := range permissions {
		permissions[i] = fmt.Sprintf("resource.permission:%d", i)
	}
	spec := Spec{Name: "article", Table: "articles", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: permissions, Dialects: []string{"sqlite"}}
	if err := Validate(spec); err == nil {
		t.Fatal("oversized resource permission list was accepted")
	}
}

func TestCheckCompatibility(t *testing.T) {
	root := t.TempDir()
	spec := Spec{Name: "testimonial", Table: "testimonials", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"testimonial.read"}, Dialects: []string{"sqlite"}}
	if _, err := Generate(root, spec); err != nil {
		t.Fatalf("generate: %v", err)
	}
	report, err := CheckCompatibility(root, spec)
	if err != nil {
		t.Fatalf("check compatibility: %v", err)
	}
	if report.Collection != spec.Name || report.ContractVersion != CollectionContractVersion || report.FilesChecked != len(requiredGeneratedFiles) {
		t.Fatalf("unexpected report: %+v", report)
	}
	lockData, err := os.ReadFile(filepath.Join(root, "generated", "generator.lock"))
	if err != nil {
		t.Fatal(err)
	}
	var lock Lock
	if err := json.Unmarshal(lockData, &lock); err != nil {
		t.Fatal(err)
	}
	if lock.ContractVersion != CollectionContractVersion {
		t.Fatalf("generated lock contract version = %q", lock.ContractVersion)
	}
	lock.ContractVersion = ""
	legacyLock, err := json.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "generated", "generator.lock"), legacyLock, 0o644); err != nil {
		t.Fatal(err)
	}
	legacyReport, err := CheckCompatibility(root, spec)
	if err != nil || legacyReport.ContractVersion != CollectionContractVersion {
		t.Fatalf("legacy lock compatibility report = %+v, err=%v", legacyReport, err)
	}
	lock.ContractVersion = "v999"
	unsupportedLock, err := json.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "generated", "generator.lock"), unsupportedLock, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckCompatibility(root, spec); err == nil {
		t.Fatal("expected unsupported generated contract version to fail")
	}
	spec.Fields[0].Name = "changed"
	if _, err := Generate(root, Spec{Name: "testimonial", Table: "testimonials", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"testimonial.read"}, Dialects: []string{"sqlite"}}); err != nil {
		t.Fatalf("restore generated lock: %v", err)
	}
	if _, err := CheckCompatibility(root, spec); err == nil {
		t.Fatal("expected spec hash mismatch")
	}
}

func TestCheckCompatibilityRejectsEmbeddedSpecDriftWithUpdatedArtifactDigest(t *testing.T) {
	root := t.TempDir()
	spec := Spec{Name: "testimonial", Table: "testimonials", Fields: []Field{{Name: "title", Type: "string"}}, Permissions: []string{"testimonial.read"}, Dialects: []string{"sqlite"}}
	if _, err := Generate(root, spec); err != nil {
		t.Fatalf("generate: %v", err)
	}
	generatedRoot := filepath.Join(root, "generated")
	specPath := filepath.Join(generatedRoot, spec.Name, "spec.json")
	specBytes, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	var document SpecDocument
	if err := json.Unmarshal(specBytes, &document); err != nil {
		t.Fatal(err)
	}
	document.Collection.Fields[0].Type = "integer"
	tampered, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	tampered = append(tampered, '\n')
	if err := os.WriteFile(specPath, tampered, 0o644); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(generatedRoot, "generator.lock")
	lockBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	var lock Lock
	if err := json.Unmarshal(lockBytes, &lock); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(tampered)
	lock.ArtifactsSHA256["spec.json"] = hex.EncodeToString(digest[:])
	lockBytes, err = json.MarshalIndent(lock, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, append(lockBytes, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckCompatibility(root, spec); err == nil || !strings.Contains(err.Error(), "embedded resource spec does not match requested contract") {
		t.Fatalf("expected embedded spec drift rejection despite updated artifact digest, got %v", err)
	}
}

func TestGeneratedQueryAdapterContainsBoundedInputGuards(t *testing.T) {
	query := renderQueryClient(Spec{Name: "products"})
	for _, marker := range []string{"invalid page size", "too many filters", "invalid filter", "invalid filter value", "< 1", "> 100"} {
		if !strings.Contains(query, marker) {
			t.Fatalf("generated query adapter missing guard %q: %s", marker, query)
		}
	}
}

func TestGeneratedClientUsesBoundedQueryAndCancellation(t *testing.T) {
	client := renderTypeScriptClient(Spec{Name: "products", Fields: []Field{{Name: "title", Type: "text"}}})
	for _, marker := range []string{"./query", "AbortSignal", "ProductsListQuery", "ProductsQueryParams", "endpoint.search = encoded"} {
		if !strings.Contains(client, marker) {
			t.Fatalf("generated client missing %q: %s", marker, client)
		}
	}
}
