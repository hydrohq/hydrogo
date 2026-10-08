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
	_ "github.com/jackc/pgx/v5/stdlib"
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

func TestGenerateBuildsAggregateAdminRegistryForAllResources(t *testing.T) {
	root := t.TempDir()
	for _, spec := range []Spec{
		{Name: "project", Table: "projects", Fields: []Field{{Name: "title", Type: "string", Admin: FieldAdmin{VisibleInList: true}}}, Permissions: []string{"project.read", "project.write"}, Dialects: []string{"sqlite", "postgres"}},
		{Name: "article", Table: "articles", Fields: []Field{{Name: "headline", Type: "string", Admin: FieldAdmin{VisibleInList: true}}}, Permissions: []string{"article.read", "article.write"}, Dialects: []string{"sqlite", "postgres"}},
	} {
		if _, err := Generate(root, spec); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, "generated", "admin_registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	registry, err := adminstudio.ParseManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	resources := registry.Collections()
	if len(resources) != 2 || resources[0].ID != "article" || resources[1].ID != "project" {
		t.Fatalf("unexpected generated Admin resources: %#v", resources)
	}
	if _, err := CheckGeneratedCompatibility(root); err != nil {
		t.Fatalf("generated resources or Admin registry are incompatible: %v", err)
	}
	lockBytes, err := os.ReadFile(filepath.Join(root, "generated", "generator.lock"))
	if err != nil {
		t.Fatal(err)
	}
	var lock Lock
	if err := json.Unmarshal(lockBytes, &lock); err != nil {
		t.Fatal(err)
	}
	if len(lock.Resources) != 2 {
		t.Fatalf("generator lock lost resource entries: %#v", lock.Resources)
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
	defaultAttempts, defaultFeatured, defaultSummary := "7", "true", "fallback"
	for _, spec := range []Spec{
		{Name: "pages", Table: "pages", Fields: []Field{{Name: "title", Type: "string", Validation: Validation{MinLength: &titleMin, MaxLength: &titleMax}, Admin: FieldAdmin{VisibleInList: true, Searchable: true}}, {Name: "status", Type: "string", Admin: FieldAdmin{Filterable: true, Options: []string{"draft", "published"}}}, {Name: "email", Type: "string", Validation: Validation{Format: "email"}}, {Name: "published_on", Type: "string", Validation: Validation{Format: "date"}}, {Name: "updated_at", Type: "string", Validation: Validation{Format: "datetime"}}, {Name: "canonical_url", Type: "string", Validation: Validation{Format: "url"}}, {Name: "attempts", Type: "integer", Validation: Validation{Default: &defaultAttempts}}, {Name: "featured", Type: "bool", Validation: Validation{Default: &defaultFeatured}}, {Name: "summary", Type: "text", Nullable: true, Validation: Validation{Default: &defaultSummary}}}, Permissions: []string{"pages.read", "pages.write"}, Dialects: []string{"sqlite", "postgres"}},
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
		if err != nil || !bytes.Contains(openapi, []byte("HydroAPIError")) || !bytes.Contains(openapi, []byte("page_size")) || !bytes.Contains(openapi, []byte("429")) || !bytes.Contains(openapi, []byte("422")) || !bytes.Contains(openapi, []byte("/{id}")) || !bytes.Contains(openapi, []byte("\"post\"")) || spec.Name == "pages" && (!bytes.Contains(openapi, []byte("\"default\": 7")) || !bytes.Contains(openapi, []byte("\"default\": true"))) {
			t.Fatalf("%s OpenAPI missing query/error contract: %v", spec.Name, err)
		}
		if spec.Name == "pages" {
			formSchema, readErr := os.ReadFile(filepath.Join(root, "generated", spec.Name, "form.schema.json"))
			if readErr != nil || !bytes.Contains(formSchema, []byte(`"default": "fallback"`)) {
				t.Fatalf("form schema missing defaults including explicit empty string: err=%v schema=%s", readErr, formSchema)
			}
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
import (
    "context"
    "errors"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)
type allowPagesPolicy struct{}
func (allowPagesPolicy) CanRead(context.Context, string) bool { return true }
func (allowPagesPolicy) CanCreate(context.Context) bool { return true }
func (allowPagesPolicy) CanUpdate(context.Context, string) bool { return true }
func (allowPagesPolicy) CanDelete(context.Context, string) bool { return true }
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
func TestGeneratedPresenceAwareDefaults(t *testing.T) {
    omitted := httptest.NewRequest(http.MethodPost, PagesRoute, strings.NewReader("{}")); omittedResult := httptest.NewRecorder(); var omittedItem Pages
    omittedFields, ok := decodePagesCreateJSON(omittedResult, omitted, &omittedItem); if !ok { t.Fatalf("omitted decode failed: %s", omittedResult.Body.String()) }; omittedItem.ApplyCreateDefaults(omittedFields)
    if omittedItem.Attempts != 7 || !omittedItem.Featured || omittedItem.Summary != "fallback" { t.Fatalf("omitted defaults not applied: %+v", omittedItem) }
    explicit := httptest.NewRequest(http.MethodPost, PagesRoute, strings.NewReader("{\"attempts\":0,\"featured\":false,\"summary\":\"\"}")); explicitResult := httptest.NewRecorder(); var explicitItem Pages
    explicitFields, ok := decodePagesCreateJSON(explicitResult, explicit, &explicitItem); if !ok { t.Fatalf("explicit decode failed: %s", explicitResult.Body.String()) }; explicitItem.ApplyCreateDefaults(explicitFields)
    if explicitItem.Attempts != 0 || explicitItem.Featured || explicitItem.Summary != "" { t.Fatalf("explicit zero values were overwritten: %+v", explicitItem) }
}

func TestGeneratedTransportReturnsSafeFieldValidationError(t *testing.T) {
    handler := NewPagesHTTPHandler(NewPagesService(PagesRepository{}), allowPagesPolicy{})
    body := strings.NewReader("{\"id\":\"p2\",\"title\":\"漢字\",\"status\":\"draft\",\"email\":\"not-an-email\",\"published_on\":\"2026-09-23\",\"updated_at\":\"2026-09-23T12:00:00Z\",\"canonical_url\":\"https://example.com\"}")
    req := httptest.NewRequest(http.MethodPost, PagesRoute, body)
    req.Header.Set("X-Request-ID", "req_123-test")
    response := httptest.NewRecorder()
    handler.ServeHTTP(response, req)
    if response.Code != http.StatusUnprocessableEntity { t.Fatalf("status = %d body=%s", response.Code, response.Body.String()) }
    for _, marker := range []string{"\"code\":\"validation_failed\"", "\"fields\":{\"email\":\"invalid_format\"}", "\"request_id\":\"req_123-test\""} { if !strings.Contains(response.Body.String(), marker) { t.Fatalf("missing %s: %s", marker, response.Body.String()) } }
    if strings.Contains(response.Body.String(), "not-an-email") || strings.Contains(response.Body.String(), "mail.ParseAddress") { t.Fatalf("sensitive input or internals leaked: %s", response.Body.String()) }
}
func TestGeneratedTransportDoesNotMisclassifyRepositoryErrors(t *testing.T) {
    for _, tc := range []struct { name string; err error; want int }{{"internal", errors.New("database unavailable: secret"), http.StatusInternalServerError}, {"conflict", ErrResourceConflict, http.StatusConflict}} {
        t.Run(tc.name, func(t *testing.T) { req := httptest.NewRequest(http.MethodPost, PagesRoute, nil); response := httptest.NewRecorder(); writeGeneratedMutationError(response, req, tc.err); if response.Code != tc.want { t.Fatalf("status = %d, want %d", response.Code, tc.want) }; if strings.Contains(response.Body.String(), "database unavailable") || strings.Contains(response.Body.String(), "secret") { t.Fatalf("internal error leaked: %s", response.Body.String()) } })
    }
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

func TestGeneratedImmutableFieldsUseAtomicUpdateGuards(t *testing.T) {
	root := t.TempDir()
	specs := []Spec{
		{Name: "immutable_doc", Table: "immutable_docs", Versioned: true, Fields: []Field{{Name: "title", Type: "string", Validation: Validation{Immutable: true}}, {Name: "body", Type: "string"}}, Permissions: []string{"immutable_doc.read", "immutable_doc.write"}, Dialects: []string{"sqlite", "postgres"}},
		{Name: "tenant_immutable_doc", Table: "tenant_immutable_docs", TenantScoped: true, Versioned: true, Fields: []Field{{Name: "title", Type: "string", Validation: Validation{Immutable: true}}, {Name: "body", Type: "string"}}, Permissions: []string{"tenant_immutable_doc.read", "tenant_immutable_doc.write"}, Dialects: []string{"sqlite", "postgres"}},
	}
	for _, spec := range specs {
		if _, err := Generate(root, spec); err != nil {
			t.Fatal(err)
		}
		name := exportName(spec.Name)
		var source string
		if spec.TenantScoped {
			source = fmt.Sprintf(`package generated
import ("context"; "database/sql"; "errors"; "testing"; _ "modernc.org/sqlite")
func TestImmutableMutationIsAtomic(t *testing.T) {
 db, err := sql.Open("sqlite", ":memory:"); if err != nil { t.Fatal(err) }; defer db.Close()
 if _, err := db.Exec("CREATE TABLE %s (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, title TEXT NOT NULL, body TEXT NOT NULL, version BIGINT NOT NULL DEFAULT 1)"); err != nil { t.Fatal(err) }
 repo := %sRepository{DB: db, Dialect: "sqlite"}; ctx := context.Background()
 item := %s{ID:"row-1", TenantID:"tenant-1", Version:1, Title:"fixed", Body:"before"}
 if err := repo.Create(ctx, "tenant-1", item); err != nil { t.Fatal(err) }
 changed := item; changed.Title = "tampered"; changed.Body = "after"
 if err := repo.Update(ctx, "tenant-1", changed); !errors.Is(err, ErrResourceConflict) { t.Fatalf("immutable update error = %%v", err) }
 got, err := repo.Get(ctx, "tenant-1", "row-1"); if err != nil { t.Fatal(err) }; if got.Title != "fixed" || got.Body != "before" { t.Fatalf("rejected update changed row: %%+v", got) }
 item.Body = "after"; if err := repo.Update(ctx, "tenant-1", item); err != nil { t.Fatalf("mutable field update failed: %%v", err) }; fresh, err := repo.Get(ctx, "tenant-1", "row-1"); if err != nil || fresh.Version != 2 { t.Fatalf("version did not advance: item=%%+v err=%%v", fresh, err) }; if err := repo.Update(ctx, "tenant-1", item); !errors.Is(err, ErrResourceConflict) { t.Fatalf("stale version update error = %%v", err) }
}

`, spec.Table, name, name)
		} else {
			source = fmt.Sprintf(`package generated
import ("context"; "database/sql"; "errors"; "testing"; _ "modernc.org/sqlite")
func TestImmutableMutationIsAtomic(t *testing.T) {
 db, err := sql.Open("sqlite", ":memory:"); if err != nil { t.Fatal(err) }; defer db.Close()
 if _, err := db.Exec("CREATE TABLE %s (id TEXT PRIMARY KEY, title TEXT NOT NULL, body TEXT NOT NULL, version BIGINT NOT NULL DEFAULT 1)"); err != nil { t.Fatal(err) }
 repo := %sRepository{DB: db, Dialect: "sqlite"}; ctx := context.Background()
 item := %s{ID:"row-1", Version:1, Title:"fixed", Body:"before"}
 if err := repo.Create(ctx, item); err != nil { t.Fatal(err) }
 changed := item; changed.Title = "tampered"; changed.Body = "after"
 if err := repo.Update(ctx, changed); !errors.Is(err, ErrResourceConflict) { t.Fatalf("immutable update error = %%v", err) }
 got, err := repo.Get(ctx, "row-1"); if err != nil { t.Fatal(err) }; if got.Title != "fixed" || got.Body != "before" { t.Fatalf("rejected update changed row: %%+v", got) }
 item.Body = "after"; if err := repo.Update(ctx, item); err != nil { t.Fatalf("mutable field update failed: %%v", err) }; fresh, err := repo.Get(ctx, "row-1"); if err != nil || fresh.Version != 2 { t.Fatalf("version did not advance: item=%%+v err=%%v", fresh, err) }; if err := repo.Update(ctx, item); !errors.Is(err, ErrResourceConflict) { t.Fatalf("stale version update error = %%v", err) }
}
`, spec.Table, name, name)
		}
		if err := os.WriteFile(filepath.Join(root, "generated", spec.Name, "immutable_runtime_test.go"), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	moduleFile := fmt.Sprintf("module generatedimmutablefixture\n\ngo 1.27\n\nrequire (\n github.com/ardaozyildirim/hydrogo v0.0.0\n modernc.org/sqlite v1.59.0\n)\nreplace github.com/ardaozyildirim/hydrogo => %s\n", filepath.ToSlash(repoRoot))
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(moduleFile), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "-mod=mod", "./generated/...")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated immutable fixtures failed: %v\n%s", err, output)
	}
}

func TestVersionMigrationAddsOptimisticVersionWithoutChangingExistingRows(t *testing.T) {
	spec := Spec{Name: "article", Table: "articles", Versioned: true, Fields: []Field{{Name: "title", Type: "string"}}, Dialects: []string{"sqlite", "postgres"}}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(renderMigration(spec)); err != nil {
		t.Fatalf("SQLite rejected base migration: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO articles (id, title) VALUES ('a1', 'Existing')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(renderVersionMigration(spec)); err != nil {
		t.Fatalf("SQLite rejected additive version migration: %v", err)
	}
	var id, title string
	var version int64
	if err := db.QueryRow(`SELECT id, title, version FROM articles WHERE id = 'a1'`).Scan(&id, &title, &version); err != nil {
		t.Fatal(err)
	}
	if id != "a1" || title != "Existing" || version != 1 {
		t.Fatalf("existing row did not preserve its data/default version: id=%q title=%q version=%d", id, title, version)
	}
	metadata := renderVersionMigrationMetadata(spec)
	for _, marker := range []string{"sqlite: expand", "postgres: expand", "checksum_sha256:", "readable_epoch: 1", "writable_epoch: 2", "rollback:"} {
		if !strings.Contains(metadata, marker) {
			t.Fatalf("version migration metadata missing %q: %s", marker, metadata)
		}
	}
	if !strings.Contains(renderVersionMigration(spec), "ADD COLUMN version BIGINT NOT NULL DEFAULT 1") {
		t.Fatal("portable PostgreSQL/SQLite version migration was not generated")
	}
}

func TestRevisionMigrationPreservesTenantScopedAppendOnlyHistory(t *testing.T) {
	spec := Spec{Name: "article", Table: "articles", TenantScoped: true, Versioned: true, Revisions: true, Fields: []Field{{Name: "title", Type: "string"}}, Dialects: []string{"sqlite", "postgres"}}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(renderRevisionsMigration(spec)); err != nil {
		t.Fatalf("SQLite rejected revision migration: %v", err)
	}
	table := revisionTableName(spec.Table)
	insert := "INSERT INTO " + table + " (tenant_id, resource_id, version, schema_version, actor_id, created_at, state, snapshot) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	if _, err := db.Exec(insert, "t1", "a1", 1, 1, "editor", "2026-10-07T10:00:00Z", "draft", `{"title":"One"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(insert, "t1", "a1", 2, 1, "editor", "2026-10-07T10:01:00Z", "draft", `{"title":"Two"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(insert, "t2", "a1", 1, 1, "editor", "2026-10-07T10:02:00Z", "draft", `{"title":"Other tenant"}`); err != nil {
		t.Fatalf("revision identities must be tenant scoped: %v", err)
	}
	if _, err := db.Exec(insert, "t1", "a1", 2, 1, "editor", "2026-10-07T10:03:00Z", "draft", `{}`); err == nil {
		t.Fatal("duplicate resource version was accepted")
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM "+table+" WHERE tenant_id = ? AND resource_id = ?", "t1", "a1").Scan(&count); err != nil || count != 2 {
		t.Fatalf("tenant history count=%d err=%v", count, err)
	}
	metadata := renderRevisionsMigrationMetadata(spec)
	for _, marker := range []string{"sqlite: expand", "postgres: expand", "checksum_sha256:", "rollback:", "bounded append-only"} {
		if !strings.Contains(metadata, marker) {
			t.Fatalf("revision migration metadata missing %q: %s", marker, metadata)
		}
	}
}

func TestRevisionMigrationOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	spec := Spec{Name: "revision_contract", Table: "hydro_generated_revision_contract", TenantScoped: true, Versioned: true, Revisions: true, Fields: []Field{{Name: "title", Type: "string"}}, Dialects: []string{"sqlite", "postgres"}}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	table := revisionTableName(spec.Table)
	if _, err := db.Exec("DROP TABLE IF EXISTS " + table); err != nil {
		t.Fatal(err)
	}
	defer db.Exec("DROP TABLE IF EXISTS " + table)
	if _, err := db.Exec(renderRevisionsMigration(spec)); err != nil {
		t.Fatalf("PostgreSQL rejected revision migration: %v", err)
	}
	if _, err := db.Exec("INSERT INTO "+table+" (tenant_id, resource_id, version, schema_version, actor_id, created_at, state, snapshot) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", "t1", "a1", 1, 1, "editor", "2026-10-07T10:00:00Z", "draft", `{"title":"One"}`); err != nil {
		t.Fatalf("PostgreSQL rejected revision row: %v", err)
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM "+table+" WHERE tenant_id = $1 AND resource_id = $2", "t1", "a1").Scan(&count); err != nil || count != 1 {
		t.Fatalf("PostgreSQL revision count=%d err=%v", count, err)
	}
}

func TestDraftAndLocalizationMigrationsPreservePublicationAndDefaultLocale(t *testing.T) {
	spec := Spec{
		Name: "article", Table: "articles", Versioned: true, Revisions: true, Drafts: true, Localization: true,
		Fields:   []Field{{Name: "title", Type: "string", Admin: FieldAdmin{Localized: true}}, {Name: "status", Type: "string"}},
		Dialects: []string{"sqlite", "postgres"},
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(renderMigration(spec)); err != nil {
		t.Fatalf("SQLite base migration: %v", err)
	}
	if _, err := db.Exec("INSERT INTO articles (id, title, status) VALUES ('a1', 'Published title', 'published')"); err != nil {
		t.Fatal(err)
	}
	draftTable := draftTableName(spec.Table)
	if _, err := db.Exec(renderDraftsMigration(spec)); err != nil {
		t.Fatalf("SQLite draft migration: %v", err)
	}
	revisionTable := revisionTableName(spec.Table)
	if _, err := db.Exec(renderRevisionsMigration(spec)); err != nil {
		t.Fatalf("SQLite localized revision migration: %v", err)
	}
	for _, locale := range []string{"en", "fr"} {
		if _, err := db.Exec("INSERT INTO "+revisionTable+" (resource_id, locale, version, schema_version, actor_id, created_at, state, snapshot) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", "a1", locale, 1, 1, "editor", "2026-10-07T10:00:00Z", "draft", `{"title":"Version"}`); err != nil {
			t.Fatalf("locale revision identity was not independent for %s: %v", locale, err)
		}
	}
	if _, err := db.Exec("INSERT INTO "+draftTable+" (resource_id, locale, version, published_version, schema_version, actor_id, updated_at, state, snapshot) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", "a1", "en", 1, 4, 1, "editor", "2026-10-07T10:00:00Z", "draft", `{"title":"Draft title"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO "+draftTable+" (resource_id, locale, version, published_version, schema_version, actor_id, updated_at, state, snapshot) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", "a1", "fr", 1, 4, 1, "editor", "2026-10-07T10:01:00Z", "draft", `{"title":"Titre brouillon"}`); err != nil {
		t.Fatalf("draft versions must be independent by locale: %v", err)
	}
	var publishedVersion int64
	if err := db.QueryRow("SELECT published_version FROM "+draftTable+" WHERE resource_id = ? AND locale = ?", "a1", "fr").Scan(&publishedVersion); err != nil || publishedVersion != 4 {
		t.Fatalf("draft must retain its published compare version: version=%d err=%v", publishedVersion, err)
	}
	var published string
	if err := db.QueryRow("SELECT title FROM articles WHERE id = ?", "a1").Scan(&published); err != nil || published != "Published title" {
		t.Fatalf("draft changed published representation: title=%q err=%v", published, err)
	}
	localizationTable := localizationTableName(spec.Table)
	localizationSQL := strings.ReplaceAll(renderLocalizationMigration(spec, "sqlite"), "__HYDRO_DEFAULT_LOCALE__", "en")
	if _, err := db.Exec(localizationSQL); err != nil {
		t.Fatalf("SQLite localization migration: %v\n%s", err, localizationSQL)
	}
	var locale, value string
	if err := db.QueryRow("SELECT locale, json_extract(values_json, '$.title') FROM "+localizationTable+" WHERE resource_id = ?", "a1").Scan(&locale, &value); err != nil || locale != "en" || value != "Published title" {
		t.Fatalf("default locale backfill lost existing content: locale=%q value=%q err=%v", locale, value, err)
	}
	metadata := renderLocalizationMigrationMetadata(spec)
	for _, marker := range []string{"sqlite: expand", "postgres: expand", "checksum_sha256:", "rollback:", "default locale"} {
		if !strings.Contains(metadata, marker) {
			t.Fatalf("localization migration metadata missing %q: %s", marker, metadata)
		}
	}
}

func TestDraftLocalizationMigrationsOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	spec := Spec{Name: "editorial_contract", Table: "hydro_generated_editorial_contract", Versioned: true, Revisions: true, Drafts: true, Localization: true, Fields: []Field{{Name: "title", Type: "string", Admin: FieldAdmin{Localized: true}}, {Name: "status", Type: "string"}}, Dialects: []string{"sqlite", "postgres"}}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	draftTable, localeTable := draftTableName(spec.Table), localizationTableName(spec.Table)
	revisionTable := revisionTableName(spec.Table)
	for _, table := range []string{localeTable, draftTable, revisionTable, spec.Table} {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + table); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		for _, table := range []string{localeTable, draftTable, revisionTable, spec.Table} {
			_, _ = db.Exec("DROP TABLE IF EXISTS " + table)
		}
	}()
	if _, err := db.Exec(renderMigration(spec)); err != nil {
		t.Fatalf("PostgreSQL base migration: %v", err)
	}
	if _, err := db.Exec("INSERT INTO "+spec.Table+" (id, title, status) VALUES ($1, $2, $3)", "a1", "Published title", "published"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(renderDraftsMigration(spec)); err != nil {
		t.Fatalf("PostgreSQL draft migration: %v", err)
	}
	if _, err := db.Exec(renderRevisionsMigration(spec)); err != nil {
		t.Fatalf("PostgreSQL localized revision migration: %v", err)
	}
	for _, locale := range []string{"en", "fr"} {
		if _, err := db.Exec("INSERT INTO "+revisionTable+" (resource_id, locale, version, schema_version, actor_id, created_at, state, snapshot) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", "a1", locale, 1, 1, "editor", "2026-10-07T10:00:00Z", "draft", `{"title":"Version"}`); err != nil {
			t.Fatalf("PostgreSQL locale revision identity was not independent for %s: %v", locale, err)
		}
	}
	if _, err := db.Exec("INSERT INTO "+draftTable+" (resource_id, locale, version, published_version, schema_version, actor_id, updated_at, state, snapshot) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)", "a1", "en", 1, 4, 1, "editor", "2026-10-07T10:00:00Z", "draft", `{"title":"Draft title"}`); err != nil {
		t.Fatal(err)
	}
	var draftPublishedVersion int64
	if err := db.QueryRow("SELECT published_version FROM "+draftTable+" WHERE resource_id = $1 AND locale = $2", "a1", "en").Scan(&draftPublishedVersion); err != nil || draftPublishedVersion != 4 {
		t.Fatalf("PostgreSQL draft must retain its published compare version: version=%d err=%v", draftPublishedVersion, err)
	}
	if _, err := db.Exec(strings.ReplaceAll(renderLocalizationMigration(spec, "postgres"), "__HYDRO_DEFAULT_LOCALE__", "en")); err != nil {
		t.Fatalf("PostgreSQL localization migration: %v", err)
	}
	var locale, value, published string
	if err := db.QueryRow("SELECT locale, values_json::json->>'title' FROM "+localeTable+" WHERE resource_id = $1", "a1").Scan(&locale, &value); err != nil || locale != "en" || value != "Published title" {
		t.Fatalf("PostgreSQL locale backfill: locale=%q value=%q err=%v", locale, value, err)
	}
	if err := db.QueryRow("SELECT title FROM "+spec.Table+" WHERE id = $1", "a1").Scan(&published); err != nil || published != "Published title" {
		t.Fatalf("draft changed PostgreSQL published representation: title=%q err=%v", published, err)
	}
}

func TestVersionMigrationAndCASOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	spec := Spec{Name: "version_contract", Table: "hydro_generated_version_contract", Versioned: true, Fields: []Field{{Name: "title", Type: "string"}}, Dialects: []string{"sqlite", "postgres"}}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`DROP TABLE IF EXISTS hydro_generated_version_contract`); err != nil {
		t.Fatal(err)
	}
	defer db.Exec(`DROP TABLE IF EXISTS hydro_generated_version_contract`)
	if _, err := db.Exec(renderMigration(spec)); err != nil {
		t.Fatalf("PostgreSQL rejected base migration: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO hydro_generated_version_contract (id, title) VALUES ('a1', 'Existing')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(renderVersionMigration(spec)); err != nil {
		t.Fatalf("PostgreSQL rejected additive version migration: %v", err)
	}
	var version int64
	if err := db.QueryRow(`SELECT version FROM hydro_generated_version_contract WHERE id = $1`, "a1").Scan(&version); err != nil || version != 1 {
		t.Fatalf("existing PostgreSQL row version=%d err=%v", version, err)
	}
	result, err := db.Exec(`UPDATE hydro_generated_version_contract SET title = $1, version = version + 1 WHERE id = $2 AND version = $3`, "Updated", "a1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		t.Fatalf("PostgreSQL versioned update affected=%d err=%v", affected, err)
	}
	result, err = db.Exec(`UPDATE hydro_generated_version_contract SET title = $1, version = version + 1 WHERE id = $2 AND version = $3`, "Stale", "a1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 0 {
		t.Fatalf("stale PostgreSQL update affected=%d err=%v", affected, err)
	}
}

func TestImmutableFieldRequiresRepresentableScalarValue(t *testing.T) {
	tests := []Spec{
		{Name: "nullable_immutable", Table: "nullable_immutables", Fields: []Field{{Name: "value", Type: "string", Nullable: true, Validation: Validation{Immutable: true}}}, Permissions: []string{"nullable_immutable.read"}, Dialects: []string{"sqlite"}},
		{Name: "many_immutable", Table: "many_immutables", Fields: []Field{{Name: "related_ids", Type: "relation", Nullable: false, Relation: &Relation{Collection: "related_item", Cardinality: "many", MaxItems: 10}, Validation: Validation{Immutable: true}}}, Permissions: []string{"many_immutable.read"}, Dialects: []string{"sqlite"}},
	}
	for _, spec := range tests {
		if err := Validate(spec); err == nil {
			t.Errorf("unsupported immutable contract accepted for %s", spec.Name)
		}
	}
}

func TestFormDefaultsAreEscapedAndTyped(t *testing.T) {
	unsafe := `"><script>alert(1)</script>`
	truth := "true"
	spec := Spec{Name: "form_defaults", Table: "form_defaults", Fields: []Field{{Name: "title", Type: "string", Nullable: true, Validation: Validation{Default: &unsafe}}, {Name: "enabled", Type: "bool", Validation: Validation{Default: &truth}}}, Permissions: []string{"form_defaults.read"}, Dialects: []string{"sqlite"}}
	form := renderForm(spec)
	if strings.Contains(form, `<script>alert(1)</script>`) || !strings.Contains(form, `value="&#34;&gt;&lt;script&gt;alert(1)&lt;/script&gt;"`) || !strings.Contains(form, `type="checkbox" required checked`) {
		t.Fatalf("defaults were not safely rendered: %s", form)
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
		Admin: Admin{Label: "Catalog", Description: "Sellable products", Group: "commerce", Icon: "box", UseAsTitle: "title", Actions: []adminstudio.Action{{Label: "Products", Href: "/admin/commerce/products"}}},
	}
	collection, err := AdminCollection(spec)
	if err != nil {
		t.Fatal(err)
	}
	if collection.Label != "Catalog" || collection.Description != "Sellable products" || collection.Group != "commerce" || collection.Icon != "box" || collection.TitleField != "title" || collection.Fields[1].Kind != adminstudio.FieldNumber || !collection.Fields[0].VisibleInList || !collection.Fields[0].Searchable || !collection.Fields[1].Filterable || collection.Actions[0].Href != "/admin/commerce/products" {
		t.Fatalf("collection = %#v", collection)
	}
}

func TestAdminCollectionHumanizesDefaultFieldLabels(t *testing.T) {
	collection, err := AdminCollection(Spec{
		Name: "article", Table: "articles", Permissions: []string{"article.read"}, Dialects: []string{"sqlite", "postgres"},
		Fields: []Field{{Name: "published_at", Type: "string"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := collection.Fields[0].Label; got != "Published at" {
		t.Fatalf("default field label = %q, want %q", got, "Published at")
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
	for _, marker := range []string{"ui.BlockEditor", "Nonce: view.Nonce"} {
		if !strings.Contains(text, marker) {
			t.Fatalf("generated form missing %q: %s", marker, text)
		}
	}
	if strings.Contains(text, "/api/preview") {
		t.Fatalf("generated form invented an Edition preview route: %s", text)
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

func TestGeneratedUniqueIndexesAreTenantScopedAndPortable(t *testing.T) {
	email := Field{Name: "email", Type: "string", Validation: Validation{Unique: true}}
	for _, tc := range []struct {
		name   string
		spec   Spec
		marker string
	}{
		{name: "global unique", spec: Spec{Name: "contact", Table: "contacts", Fields: []Field{email}, Dialects: []string{"sqlite", "postgres"}}, marker: "ON contacts (email)"},
		{name: "tenant unique", spec: Spec{Name: "member", Table: "members", TenantScoped: true, Fields: []Field{email}, Dialects: []string{"sqlite", "postgres"}}, marker: "ON members (tenant_id, email)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			migration := renderMigration(tc.spec)
			if !strings.Contains(migration, "CREATE UNIQUE INDEX") || !strings.Contains(migration, tc.marker) {
				t.Fatalf("unique migration missing scope %q: %s", tc.marker, migration)
			}
			indexName := uniqueIndexName(tc.spec.Table, "email")
			if len(indexName) > 63 || indexName != uniqueIndexName(tc.spec.Table, "email") {
				t.Fatalf("index name is not deterministic/bounded: %q", indexName)
			}
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := db.Exec(migration); err != nil {
				t.Fatalf("SQLite rejected generated migration: %v", err)
			}
			if tc.spec.TenantScoped {
				if _, err := db.Exec("INSERT INTO members(id,tenant_id,email) VALUES(?,?,?)", "1", "a", "same@example.com"); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec("INSERT INTO members(id,tenant_id,email) VALUES(?,?,?)", "2", "b", "same@example.com"); err != nil {
					t.Fatalf("uniqueness crossed tenants: %v", err)
				}
				if _, err := db.Exec("INSERT INTO members(id,tenant_id,email) VALUES(?,?,?)", "3", "a", "same@example.com"); err == nil {
					t.Fatal("duplicate within tenant accepted")
				}
			} else {
				if _, err := db.Exec("INSERT INTO contacts(id,email) VALUES(?,?)", "1", "same@example.com"); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec("INSERT INTO contacts(id,email) VALUES(?,?)", "2", "same@example.com"); err == nil {
					t.Fatal("global duplicate accepted")
				}
			}
		})
	}
	if err := Validate(Spec{Name: "tags", Table: "tags", Fields: []Field{{Name: "tag_ids", Type: "relation", Relation: &Relation{Collection: "tag", Cardinality: "many", MaxItems: 5}, Validation: Validation{Unique: true}}}, Dialects: []string{"sqlite", "postgres"}}); err == nil {
		t.Fatal("many-valued relation accepted scalar uniqueness")
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
