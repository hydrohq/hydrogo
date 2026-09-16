// Package generator implements the deterministic, deny-by-default collection
// generator described by ADR-0010. It emits reviewable skeletons only.
package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/pathguard"
)

var (
	ErrInvalidSpec        = errors.New("invalid collection spec")
	ErrDestructiveSpec    = errors.New("destructive migration must be written by hand")
	ErrPublishedMigration = errors.New("published generated migration cannot be changed")
	identifierPattern     = regexp.MustCompile(`^[a-z][a-z0-9_]{1,62}$`)
	permissionPattern     = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{1,127}$`)
	forbiddenFieldPattern = regexp.MustCompile(`(?i)(password|secret|token|private_key)`)
)

type Field struct {
	Name     string     `json:"name" yaml:"name"`
	Type     string     `json:"type" yaml:"type"`
	Nullable bool       `json:"nullable" yaml:"nullable"`
	Admin    FieldAdmin `json:"admin" yaml:"admin"`
}

// FieldAdmin is presentation metadata. It does not change persistence or
// authorization rules, which remain explicit in the generated review surface.
type FieldAdmin struct {
	Label         string `json:"label" yaml:"label"`
	Widget        string `json:"widget" yaml:"widget"`
	ReadOnly      bool   `json:"read_only" yaml:"read_only"`
	VisibleInList bool   `json:"visible_in_list" yaml:"visible_in_list"`
}

type Admin struct {
	Label       string               `json:"label" yaml:"label"`
	Description string               `json:"description" yaml:"description"`
	Actions     []adminstudio.Action `json:"actions" yaml:"actions"`
}

type Spec struct {
	Name        string   `json:"name" yaml:"name"`
	Table       string   `json:"table" yaml:"table"`
	Fields      []Field  `json:"fields" yaml:"fields"`
	Permissions []string `json:"permissions" yaml:"permissions"`
	Dialects    []string `json:"dialects" yaml:"dialects"`
	Destructive bool     `json:"destructive" yaml:"destructive"`
	Admin       Admin    `json:"admin" yaml:"admin"`
	Kits        []string `json:"kits,omitempty" yaml:"kits,omitempty"`
}

type Lock struct {
	GeneratorVersion string `json:"generator_version"`
	SpecSHA256       string `json:"spec_sha256"`
}

func Validate(spec Spec) error {
	if !identifierPattern.MatchString(spec.Name) || !identifierPattern.MatchString(spec.Table) || len(spec.Fields) == 0 || len(spec.Permissions) == 0 {
		return ErrInvalidSpec
	}
	if spec.Destructive {
		return ErrDestructiveSpec
	}
	seen := map[string]struct{}{}
	seenPermissions := map[string]struct{}{}
	for _, permission := range spec.Permissions {
		permission = strings.TrimSpace(permission)
		if !permissionPattern.MatchString(permission) || strings.Contains(permission, "..") {
			return ErrInvalidSpec
		}
		if _, ok := seenPermissions[permission]; ok {
			return ErrInvalidSpec
		}
		seenPermissions[permission] = struct{}{}
	}
	for _, dialect := range spec.Dialects {
		if dialect != "sqlite" && dialect != "postgres" {
			return ErrInvalidSpec
		}
	}
	if len(spec.Dialects) == 0 {
		return ErrInvalidSpec
	}
	for _, field := range spec.Fields {
		if !identifierPattern.MatchString(field.Name) || field.Name == "id" || strings.TrimSpace(field.Type) == "" || forbiddenFieldPattern.MatchString(field.Name) {
			return ErrInvalidSpec
		}
		if _, ok := seen[field.Name]; ok {
			return ErrInvalidSpec
		}
		seen[field.Name] = struct{}{}
	}
	for _, action := range spec.Admin.Actions {
		if strings.TrimSpace(action.Label) == "" || !strings.HasPrefix(action.Href, "/") || strings.HasPrefix(action.Href, "//") {
			return ErrInvalidSpec
		}
	}
	knownKits := map[string]struct{}{"localization": {}, "media": {}, "seo": {}, "forms": {}, "redirects": {}, "search": {}, "commerce": {}}
	for _, kit := range spec.Kits {
		if _, ok := knownKits[strings.ToLower(strings.TrimSpace(kit))]; !ok {
			return ErrInvalidSpec
		}
	}
	return nil
}

// AdminCollection projects a reviewable spec into metadata that the Admin
// Studio can render without obtaining a direct database capability.
func AdminCollection(spec Spec) (adminstudio.Collection, error) {
	if err := Validate(spec); err != nil {
		return adminstudio.Collection{}, err
	}
	label := strings.TrimSpace(spec.Admin.Label)
	if label == "" {
		label = exportName(spec.Name)
	}
	fields := make([]adminstudio.Field, 0, len(spec.Fields))
	for _, field := range spec.Fields {
		fieldLabel := strings.TrimSpace(field.Admin.Label)
		if fieldLabel == "" {
			fieldLabel = exportName(field.Name)
		}
		fields = append(fields, adminstudio.Field{ID: field.Name, Label: fieldLabel, Kind: adminFieldKind(field.Type), Widget: field.Admin.Widget, Required: !field.Nullable, ReadOnly: field.Admin.ReadOnly, VisibleInList: field.Admin.VisibleInList})
	}
	actions := append([]adminstudio.Action(nil), spec.Admin.Actions...)
	if len(actions) == 0 {
		actions = []adminstudio.Action{{Label: "Open " + label, Href: "/admin/" + spec.Name}}
	}
	kits := make([]adminstudio.Kit, 0, len(spec.Kits))
	for _, kit := range spec.Kits {
		kits = append(kits, adminstudio.Kit(strings.ToLower(strings.TrimSpace(kit))))
	}
	resolvedKits, err := adminstudio.ResolveKits(kits)
	if err != nil {
		return adminstudio.Collection{}, err
	}
	registry, err := adminstudio.NewRegistry([]adminstudio.Collection{{ID: spec.Name, Label: label, Description: spec.Admin.Description, Fields: fields, Actions: actions, Kits: resolvedKits}})
	if err != nil {
		return adminstudio.Collection{}, err
	}
	return registry.Collections()[0], nil
}

func Generate(root string, spec Spec) (Lock, error) {
	if err := Validate(spec); err != nil {
		return Lock{}, err
	}
	if strings.TrimSpace(root) == "" {
		return Lock{}, ErrInvalidSpec
	}
	if info, err := os.Lstat(root); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return Lock{}, fmt.Errorf("generator root must be a real directory: %s", root)
		}
	} else if !os.IsNotExist(err) {
		return Lock{}, err
	}
	generatedRoot := filepath.Join(root, "generated")
	generatedCollection := filepath.Join(generatedRoot, spec.Name)
	if err := pathguard.RejectSymlinkComponents(generatedCollection); err != nil {
		return Lock{}, fmt.Errorf("generator output path: %w", err)
	}
	if err := os.MkdirAll(generatedCollection, 0o755); err != nil {
		return Lock{}, err
	}
	canonical, err := canonicalSpec(spec)
	if err != nil {
		return Lock{}, err
	}
	hash := sha256.Sum256(canonical)
	lock := Lock{GeneratorVersion: "0.1.0", SpecSHA256: hex.EncodeToString(hash[:])}
	lockBytes, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return Lock{}, err
	}
	lockBytes = append(lockBytes, '\n')
	if err := writeGenerated(filepath.Join(generatedRoot, "generator.lock"), lockBytes); err != nil {
		return Lock{}, err
	}
	files := map[string]string{
		"model.go":            renderModel(spec),
		"repository.go":       renderRepository(spec),
		"service.go":          renderService(spec),
		"policy.go":           renderPolicy(spec),
		"transport.go":        renderTransport(spec),
		"form.html":           renderForm(spec),
		"form.schema.json":    renderFormSchema(spec),
		"table.html":          renderTable(spec),
		"openapi.json":        renderOpenAPI(spec),
		"client.ts":           renderTypeScriptClient(spec),
		"query.ts":            renderQueryClient(spec),
		"bulk.ts":             renderBulkClient(spec),
		"import-export.ts":    renderImportExportClient(spec),
		"upload.ts":           renderUploadClient(spec),
		"react.ts":            renderReactAdapter(spec),
		"swr.ts":              renderSWRAdapter(spec),
		"htmx.ts":             renderHTMXAdapter(spec),
		"vanilla.ts":          renderVanillaAdapter(spec),
		"kits.ts":             renderKitManifest(spec),
		"migration.sql":       renderMigration(spec),
		"contract_test.go":    renderContractTest(spec),
		"fixture_test.go":     renderFixtureTest(spec),
		"admin_manifest.json": renderAdminManifest(spec),
		"module.yaml":         renderModuleManifest(spec),
		"REVIEW.md":           renderReviewChecklist(spec),
		"RUNTIME_WIRING.md":   renderRuntimeWiring(spec),
	}
	for name, content := range files {
		path := filepath.Join(generatedCollection, name)
		var err error
		if name == "migration.sql" {
			err = writeImmutableGenerated(path, []byte(content))
		} else {
			err = writeGenerated(path, []byte(content))
		}
		if err != nil {
			return Lock{}, err
		}
	}
	return lock, nil
}

func canonicalSpec(spec Spec) ([]byte, error) {
	copySpec := spec
	sort.Strings(copySpec.Permissions)
	sort.Strings(copySpec.Dialects)
	sort.Strings(copySpec.Kits)
	sort.Slice(copySpec.Fields, func(i, j int) bool { return copySpec.Fields[i].Name < copySpec.Fields[j].Name })
	return json.Marshal(copySpec)
}

func writeGenerated(path string, content []byte) error {
	if err := validateGeneratedPath(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := validateGeneratedPath(path); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func writeImmutableGenerated(path string, content []byte) error {
	if err := validateGeneratedPath(path); err != nil {
		return err
	}
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) != string(content) {
			return ErrPublishedMigration
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return writeGenerated(path, content)
}

func validateGeneratedPath(path string) error {
	if err := pathguard.RejectSymlinkComponents(filepath.Dir(path)); err != nil {
		return fmt.Errorf("generated parent path: %w", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("generated path cannot be a symlink: %s", path)
	}
	return nil
}

func renderModel(spec Spec) string {
	return fmt.Sprintf("// Code generated by hydrogo generator; DO NOT EDIT.\npackage generated\n\ntype %s struct {\n", exportName(spec.Name)) + renderFields(spec) + "}\n"
}

func renderFields(spec Spec) string {
	fields := append([]Field(nil), spec.Fields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	var b strings.Builder
	b.WriteString("\tID string `json:\"id\"`\n")
	for _, field := range fields {
		fmt.Fprintf(&b, "\t%s %s `json:\"%s\"`\n", goFieldName(field.Name), goType(field.Type), field.Name)
	}
	return b.String()
}

func renderRepository(spec Spec) string {
	name := exportName(spec.Name)
	fields := append([]Field(nil), spec.Fields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	columns := []string{"id"}
	for _, field := range fields {
		columns = append(columns, field.Name)
	}
	args := make([]string, 0, len(columns))
	for i := range columns {
		args = append(args, fmt.Sprintf("placeholder(dialect, %d)", i+1))
	}
	insertFields := strings.Join(columns, ", ")
	insertQuery := fmt.Sprintf("\"INSERT INTO %s (%s) VALUES (\" + %s + \")\"", spec.Table, insertFields, strings.Join(args, " + \", \" + "))
	var scan strings.Builder
	for i, column := range columns {
		if i > 0 {
			scan.WriteString(", ")
		}
		fmt.Fprintf(&scan, "&item.%s", goFieldName(column))
	}
	updateArgs := make([]string, 0, len(fields)+1)
	for _, field := range fields {
		updateArgs = append(updateArgs, fmt.Sprintf("item.%s", goFieldName(field.Name)))
	}
	updateArgs = append(updateArgs, "item.ID")
	updateParts := make([]string, 0, len(fields))
	for i, field := range fields {
		updateParts = append(updateParts, fmt.Sprintf("\"%s = \" + placeholder(dialect, %d)", field.Name, i+1))
	}
	updateQuery := fmt.Sprintf("\"UPDATE %s SET \" + %s + \" WHERE id = \" + placeholder(dialect, %d)", spec.Table, strings.Join(updateParts, " + \", \" + "), len(fields)+1)
	return fmt.Sprintf(`// Code generated by hydrogo generator; review SQL and policies before use.
package generated

import (
    "context"
    "database/sql"
    "fmt"
)

// %sRepository is a reviewable CRUD boundary; no runtime DDL is emitted.
type %sRepository struct {
    DB *sql.DB
    Dialect string
}

func (r %sRepository) Get(ctx context.Context, id string) (%s, error) {
    var item %s
    dialect := r.Dialect
    err := r.DB.QueryRowContext(ctx, "SELECT %s FROM %s WHERE id = "+placeholder(dialect, 1), id).Scan(%s)
    return item, err
}

func (r %sRepository) Create(ctx context.Context, item %s) error {
    dialect := r.Dialect
	_, err := r.DB.ExecContext(ctx, %s, %s)
    return err
}

func (r %sRepository) Update(ctx context.Context, item %s) error {
    dialect := r.Dialect
	_, err := r.DB.ExecContext(ctx, %s, %s)
    return err
}

func (r %sRepository) Delete(ctx context.Context, id string) error {
    dialect := r.Dialect
    _, err := r.DB.ExecContext(ctx, "DELETE FROM %s WHERE id = "+placeholder(dialect, 1), id)
    return err
}

func placeholder(dialect string, position int) string {
    if dialect == "postgres" { return fmt.Sprintf("$%%d", position) }
    return "?"
}
`, name, name, name, name, name, strings.Join(columns, ", "), spec.Table, scan.String(), name, name, insertQuery, strings.Join(append([]string{"item.ID"}, fieldArgs(fields)...), ", "), name, name, updateQuery, strings.Join(updateArgs, ", "), name, spec.Table)
}

func renderTransport(spec Spec) string {
	return fmt.Sprintf("// Code generated by hydrogo generator; authorization remains explicit.\npackage generated\n\nconst %sRoute = \"/api/v1/%s\"\n", exportName(spec.Name), spec.Name)
}

func renderReviewChecklist(spec Spec) string {
	return fmt.Sprintf(`# %s generated resource review

This directory is generated from the reviewed collection contract. Keep the
generated skeleton deterministic, then complete and review these boundaries
before enabling the resource in a Client Edition.

- [ ] Authorization and tenant/member scope are explicit for every read/write action.
- [ ] Input validation, CSRF, audit events, idempotency and error copy are covered.
- [ ] SQLite and PostgreSQL behavior, indexes, rollback and migration metadata are verified.
- [ ] List pagination, filter/sort bounds, cache TTL/invalidation and stampede behavior are reviewed.
- [ ] Form/table labels, empty/error/loading states, keyboard focus, RTL and long text are tested.
- [ ] Runtime transport wiring and OpenAPI error responses match the module contract.
- [ ] Fixtures and contract tests are extended with domain-specific invariants.
- [ ] Capacity, background work, data retention and deployment compatibility are documented.

Generated collection: %s
`, exportName(spec.Name), spec.Name)
}

func renderRuntimeWiring(spec Spec) string {
	kits := resolvedKitNames(spec.Kits)
	return fmt.Sprintf(`# %s runtime wiring

This generated contract deliberately separates deterministic scaffolding from
application composition. Complete each item in the owning module before
enabling the resource; the generator never guesses authorization or runtime
dependencies.

## Registration checklist

- [ ] Register the repository with the module composition root and pass the
  configured SQL dialect/connection pool.
- [ ] Register the service and transport under the versioned route
  /api/v1/%s, including authentication, authorization and request limits.
- [ ] Bind the generated form/table/admin manifest to the authenticated admin
  surface; add CSRF, validation, error and empty-state handling.
- [ ] Apply the migration through the module migration registry for every
  declared dialect: %s.
- [ ] Add cache policy, invalidation and background-job ownership only when the
  module contract requires them; keep Compact profile resources bounded.
- [ ] Extend contract/fixture tests with domain invariants and include this
  resource in the module health/readiness checks.

## Optional capability kits

Selected kits: %s

For each selected kit, bind its module adapter and permission surface explicitly;
an omitted kit must not add routes, migrations, workers or assets.

## Required evidence

Record the owning ADR, migration/rollback result, policy test, deployment
compatibility result and capacity impact in REVIEW.md before release.
`, exportName(spec.Name), spec.Name, strings.Join(sortedStrings(spec.Dialects), ", "), strings.Join(kits, ", "))
}

func resolvedKitNames(declared []string) []string {
	kits := make([]adminstudio.Kit, 0, len(declared))
	for _, kit := range declared {
		kits = append(kits, adminstudio.Kit(strings.ToLower(strings.TrimSpace(kit))))
	}
	resolved, err := adminstudio.ResolveKits(kits)
	if err != nil {
		// Validate runs before rendering; retain deterministic output if this
		// helper is ever called independently with an invalid spec.
		return sortedStrings(declared)
	}
	names := make([]string, 0, len(resolved))
	for _, kit := range resolved {
		names = append(names, string(kit))
	}
	return names
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func renderForm(spec Spec) string {
	fields := append([]Field(nil), spec.Fields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- Code generated by hydrogo generator; review policy and copy. -->\n<form method=\"post\" action=\"/admin/%s\" data-hydro-form=\"%s\">\n", spec.Name, spec.Name)
	for _, field := range fields {
		if field.Admin.ReadOnly {
			continue
		}
		label := strings.TrimSpace(field.Admin.Label)
		if label == "" {
			label = exportName(field.Name)
		}
		typ := "text"
		switch strings.ToLower(field.Type) {
		case "int", "integer", "real", "float":
			typ = "number"
		case "bool", "boolean":
			typ = "checkbox"
		}
		required := ""
		if !field.Nullable {
			required = " required"
		}
		fmt.Fprintf(&b, "  <label for=\"%s\">%s</label>\n  <input id=\"%s\" name=\"%s\" type=\"%s\"%s>\n", field.Name, label, field.Name, field.Name, typ, required)
	}
	b.WriteString("  <button type=\"submit\">Kaydet</button>\n</form>\n")
	return b.String()
}

func renderFormSchema(spec Spec) string {
	type schemaField struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Label    string `json:"label"`
		Required bool   `json:"required"`
	}
	fields := append([]Field(nil), spec.Fields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	output := struct {
		ID     string        `json:"id"`
		Title  string        `json:"title"`
		Fields []schemaField `json:"fields"`
	}{ID: spec.Name, Title: strings.TrimSpace(spec.Admin.Label)}
	if output.Title == "" {
		output.Title = exportName(spec.Name)
	}
	for _, field := range fields {
		fieldType := "text"
		switch {
		case field.Admin.Widget == "textarea":
			fieldType = "textarea"
		case strings.Contains(strings.ToLower(field.Name), "email"):
			fieldType = "email"
		case field.Type == "int" || field.Type == "integer" || field.Type == "number" || field.Type == "float":
			fieldType = "number"
		case field.Type == "bool" || field.Type == "boolean":
			fieldType = "checkbox"
		}
		label := strings.TrimSpace(field.Admin.Label)
		if label == "" {
			label = exportName(field.Name)
		}
		output.Fields = append(output.Fields, schemaField{ID: field.Name, Type: fieldType, Label: label, Required: !field.Nullable})
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	return string(data) + "\n"
}

func renderTable(spec Spec) string {
	fields := append([]Field(nil), spec.Fields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- Code generated by hydrogo generator; review pagination, authorization, and copy. -->\n<div class=\"table-wrap\" data-hydro-table=\"%s\">\n  <table>\n    <caption>%s listesi</caption>\n    <thead><tr><th scope=\"col\">ID</th>", spec.Name, exportName(spec.Name))
	for _, field := range fields {
		if field.Admin.VisibleInList {
			fmt.Fprintf(&b, "<th scope=\"col\">%s</th>", htmlLabel(field))
		}
	}
	b.WriteString("</tr></thead>\n    <tbody>\n      <!-- Replace with bounded server-rendered rows. -->\n      <tr><td colspan=\"")
	visible := 1
	for _, field := range fields {
		if field.Admin.VisibleInList {
			visible++
		}
	}
	fmt.Fprintf(&b, "%d\"><span role=\"status\">Kayıt bulunamadı.</span></td></tr>\n    </tbody>\n  </table>\n</div>\n", visible)
	return b.String()
}

func htmlLabel(field Field) string {
	if label := strings.TrimSpace(field.Admin.Label); label != "" {
		return html.EscapeString(label)
	}
	return html.EscapeString(exportName(field.Name))
}

func renderOpenAPI(spec Spec) string {
	fields := append([]Field(nil), spec.Fields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	properties := map[string]any{"id": map[string]any{"type": "string"}}
	for _, field := range fields {
		properties[field.Name] = map[string]any{"type": openAPIType(field.Type)}
	}
	doc := map[string]any{"openapi": "3.1.0", "info": map[string]any{"title": exportName(spec.Name) + " API", "version": "1.0.0"}, "paths": map[string]any{"/api/v1/" + spec.Name: map[string]any{"get": map[string]any{"responses": map[string]any{"200": map[string]any{"description": "OK"}}}}}, "components": map[string]any{"schemas": map[string]any{exportName(spec.Name): map[string]any{"type": "object", "properties": properties}}}}
	data, _ := json.MarshalIndent(doc, "", "  ")
	return string(data) + "\n"
}

func renderTypeScriptClient(spec Spec) string {
	name := exportName(spec.Name)
	fields := append([]Field(nil), spec.Fields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	var shape strings.Builder
	shape.WriteString("  id: string;\n")
	for _, field := range fields {
		fmt.Fprintf(&shape, "  %s: %s;\n", field.Name, tsType(field.Type))
	}
	return fmt.Sprintf("// Code generated by hydrogo generator; review auth/error policy.\nexport interface %s {\n%s}\n\nexport class %sClient {\n  constructor(private readonly baseUrl: string, private readonly fetcher: typeof fetch = fetch) {}\n  async list(): Promise<%s[]> {\n    const response = await this.fetcher(new URL('/api/v1/%s', this.baseUrl));\n    if (!response.ok) throw new Error('HydroGo API request failed');\n    return response.json() as Promise<%s[]>;\n  }\n}\n", name, shape.String(), name, name, spec.Name, name)
}

func renderQueryClient(spec Spec) string {
	name := exportName(spec.Name)
	return fmt.Sprintf(`// Code generated by hydrogo generator; review cache policy before production use.
export interface %sListQuery {
  page?: number;
  pageSize?: number;
  sort?: string;
  order?: 'asc' | 'desc';
  filters?: Record<string, string | number | boolean>;
}

export function %sQueryParams(query: %sListQuery = {}): string {
  const params = new URLSearchParams();
  if (query.page) params.set('page', String(query.page));
  if (query.pageSize) params.set('page_size', String(query.pageSize));
  if (query.sort) params.set('sort', query.sort);
  if (query.order) params.set('order', query.order);
  for (const [key, value] of Object.entries(query.filters ?? {})) params.set('filter.' + key, String(value));
  return params.toString();
}

export const %sCache = {
  list: (query: %sListQuery = {}) => '%s:list:' + %sQueryParams(query),
  invalidate: (id?: string) => id ? ['%s:list', '%s:' + id] : ['%s:list'],
};
`, name, name, name, name, name, spec.Name, name, spec.Name, spec.Name, spec.Name)
}

func renderBulkClient(spec Spec) string {
	name := exportName(spec.Name)
	return fmt.Sprintf(`// Code generated by hydrogo generator; bounded interaction primitives.
export const %sBulkLimit = 500;
export type %sProgress = { completed: number; total: number; state: 'idle' | 'running' | 'success' | 'error' };

export function %sSelect(ids: string[], id: string, selected: boolean): string[] {
  const next = new Set(ids);
  if (selected) next.add(id); else next.delete(id);
  if (next.size > %sBulkLimit) throw new Error('bulk selection limit exceeded');
  return [...next];
}

export function %sProgress(completed: number, total: number, state: %sProgress['state'] = 'running'): %sProgress {
  return { completed: Math.max(0, Math.min(completed, total)), total: Math.max(0, total), state };
}
`, name, name, name, name, name, name, name)
}

func renderUploadClient(spec Spec) string {
	return fmt.Sprintf(`// Code generated by hydrogo generator; provider-neutral upload boundary.
export const %sUploadLimits = { maxBytes: 10 * 1024 * 1024, maxFiles: 20 } as const;

export type UploadProgress = { completedBytes: number; totalBytes: number };
export type UploadResult = { id: string; url?: string; contentType: string; bytes: number };

export function validate%sUpload(file: { size: number; type: string }): string | null {
  if (!Number.isFinite(file.size) || file.size < 0 || file.size > %sUploadLimits.maxBytes) return "file-too-large";
  if (!file.type || file.type.length > 128) return "invalid-content-type";
  return null;
}

// The module owns the signed URL/provider request; this helper never embeds credentials.
export function uploadProgress(completedBytes: number, totalBytes: number): UploadProgress {
  return { completedBytes: Math.max(0, completedBytes), totalBytes: Math.max(0, totalBytes) };
}
`, exportName(spec.Name), exportName(spec.Name), exportName(spec.Name))
}

func renderReactAdapter(spec Spec) string {
	name := exportName(spec.Name)
	return fmt.Sprintf(`// Optional React adapter for %s. Install React/your query library in the Client Edition.
export type %sQueryState<T> = { data?: T; loading: boolean; error?: Error };
export type %sFetcher<T> = (signal?: AbortSignal) => Promise<T>;
export type %sQueryOptions<T> = {
  queryKey: readonly unknown[];
  queryFn: (context?: { signal?: AbortSignal }) => Promise<T>;
  staleTime: number;
};

// Framework-neutral options let React Query, SWR, or a custom hook consume the
// same typed contract without pulling a UI runtime into HydroGo Core.
export function create%sQuery<T>(fetcher: %sFetcher<T>, queryKey: readonly unknown[] = ['%s', 'list']): %sQueryOptions<T> {
  return { queryKey, queryFn: (context) => fetcher(context?.signal), staleTime: 30_000 };
}

export function %sQueryKey(id?: string): readonly string[] {
  return id ? ['%s', id] : ['%s', 'list'];
}
`, name, name, name, name, name, name, spec.Name, name, name, spec.Name, spec.Name)
}

func renderSWRAdapter(spec Spec) string {
	name := exportName(spec.Name)
	return fmt.Sprintf(`// Optional SWR adapter for %s. Install SWR in the Client Edition.
export type %sSWRFetcher<T> = (key: readonly unknown[], signal?: AbortSignal) => Promise<T>;

// This adapter contains no SWR import. Pass the returned key/fetcher to useSWR
// in the client application, keeping HydroGo Core and Compact builds dependency-free.
export function create%sSWR<T>(fetcher: %sSWRFetcher<T>, id?: string) {
  const key = id ? ['%s', id] as const : ['%s', 'list'] as const;
  return { key, fetcher: (signal?: AbortSignal) => fetcher(key, signal), revalidateOnFocus: false };
}
`, name, name, name, name, spec.Name, spec.Name)
}

func renderHTMXAdapter(spec Spec) string {
	return fmt.Sprintf(`// Optional HTMX adapter for %s; no HTMX runtime is bundled.
export const %sHtmx = {
  listTarget: "#%s-list",
  formTarget: "#%s-form",
  headers: { "Accept": "text/html" } as Record<string, string>,
};

export function %sHtmxAttrs(url: string) {
  return { "hx-get": url, "hx-target": %sHtmx.listTarget, "hx-swap": "outerHTML" } as const;
}
`, exportName(spec.Name), exportName(spec.Name), spec.Name, spec.Name, exportName(spec.Name), exportName(spec.Name))
}

func renderVanillaAdapter(spec Spec) string {
	return fmt.Sprintf(`// Dependency-free browser adapter for %s.
export type %sRequest = { baseUrl: string; fetcher?: typeof fetch };

export function create%sClient(options: %sRequest) {
  const fetcher = options.fetcher ?? fetch;
  return { list: (query = "") => fetcher(options.baseUrl + "/api/v1/%s" + query, { headers: { Accept: "application/json" } }) };
}

`, exportName(spec.Name), exportName(spec.Name), exportName(spec.Name), exportName(spec.Name), spec.Name)
}

func renderKitManifest(spec Spec) string {
	kits := resolvedKitNames(spec.Kits)
	quoted := make([]string, len(kits))
	for i, kit := range kits {
		quoted[i] = fmt.Sprintf("%q", kit)
	}
	return fmt.Sprintf("// Generated capability metadata for %s. Implementations stay in the Client Edition.\nexport const %sKits = [%s] as const;\n", exportName(spec.Name), exportName(spec.Name), strings.Join(quoted, ", "))
}

func renderImportExportClient(spec Spec) string {
	name := exportName(spec.Name)
	return fmt.Sprintf(`// Code generated by hydrogo generator; bounded CSV helpers.
export const %sImportLimits = { maxBytes: 5 * 1024 * 1024, maxRows: 5000 } as const;

function csvCell(value: unknown): string {
  const text = String(value ?? '');
  return /[",\r\n]/.test(text) ? '"' + text.replace(/"/g, '""') + '"' : text;
}

export function %sExportCSV(rows: Record<string, unknown>[], columns: string[]): string {
  if (rows.length > %sImportLimits.maxRows) throw new Error('export row limit exceeded');
  return [columns.map(csvCell).join(','), ...rows.map(row => columns.map(column => csvCell(row[column])).join(','))].join('\n');
}

export function %sValidateImport(payload: string): void {
  if (new TextEncoder().encode(payload).byteLength > %sImportLimits.maxBytes) throw new Error('import payload limit exceeded');
  if (payload.split(/\r?\n/).filter(Boolean).length > %sImportLimits.maxRows + 1) throw new Error('import row limit exceeded');
}
`, name, name, name, name, name, name)
}

func tsType(value string) string {
	switch strings.ToLower(value) {
	case "int", "integer", "real", "float":
		return "number"
	case "bool", "boolean":
		return "boolean"
	default:
		return "string"
	}
}

func openAPIType(value string) string {
	switch strings.ToLower(value) {
	case "int", "integer":
		return "integer"
	case "real", "float":
		return "number"
	case "bool", "boolean":
		return "boolean"
	default:
		return "string"
	}
}

func renderService(spec Spec) string {
	name := exportName(spec.Name)
	return fmt.Sprintf(`// Code generated by hydrogo generator; business rules remain explicit.
package generated

import "context"

// %sService is a thin application boundary over the generated repository.
// Add authorization, validation, audit, and transaction policy before use.
type %sService struct {
	Repository %sRepository
}

func New%sService(repository %sRepository) %sService {
	return %sService{Repository: repository}
}

func (s %sService) Get(ctx context.Context, id string) (%s, error) {
	return s.Repository.Get(ctx, id)
}

func (s %sService) Create(ctx context.Context, item %s) error {
	return s.Repository.Create(ctx, item)
}

func (s %sService) Update(ctx context.Context, item %s) error {
	return s.Repository.Update(ctx, item)
}

func (s %sService) Delete(ctx context.Context, id string) error {
	return s.Repository.Delete(ctx, id)
}
`, name, name, name, name, name, name, name, name, name, name, name, name, name, name)
}

func renderPolicy(spec Spec) string {
	name := exportName(spec.Name)
	return fmt.Sprintf(`// Code generated by hydrogo generator; deny by default.
package generated

import "context"

// %sPolicy is intentionally explicit: callers must provide a policy before
// exposing generated mutations or reads through a transport.
type %sPolicy interface {
	CanRead(context.Context, string) bool
	CanCreate(context.Context) bool
	CanUpdate(context.Context, string) bool
	CanDelete(context.Context, string) bool
}
`, name, name)
}

func renderFixtureTest(spec Spec) string {
	name := exportName(spec.Name)
	return fmt.Sprintf(`package generated

import "testing"

func Test%sFixtureShape(t *testing.T) {
	item := %s{ID: "fixture-%s"}
	if item.ID == "" {
		t.Fatal("fixture ID must not be empty")
	}
}
`, name, name, spec.Name)
}

func renderModuleManifest(spec Spec) string {
	fields := append([]Field(nil), spec.Fields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	var b strings.Builder
	fmt.Fprintf(&b, "version: v0\nname: %s\ntable: %s\npermissions:\n", spec.Name, spec.Table)
	permissions := append([]string(nil), spec.Permissions...)
	sort.Strings(permissions)
	for _, permission := range permissions {
		fmt.Fprintf(&b, "  - %s\n", permission)
	}
	b.WriteString("dialects:\n")
	dialects := append([]string(nil), spec.Dialects...)
	sort.Strings(dialects)
	for _, dialect := range dialects {
		fmt.Fprintf(&b, "  - %s\n", dialect)
	}
	b.WriteString("fields:\n")
	for _, field := range fields {
		fmt.Fprintf(&b, "  - name: %s\n    type: %s\n    nullable: %t\n", field.Name, strings.ToLower(field.Type), field.Nullable)
	}
	return b.String()
}

func renderMigration(spec Spec) string {
	fields := append([]Field(nil), spec.Fields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	var b strings.Builder
	fmt.Fprintf(&b, "-- Code generated by hydrogo generator; review before applying.\n-- No DROP/DELETE/destructive statement is generated.\nCREATE TABLE %s (\n    id TEXT PRIMARY KEY", spec.Table)
	for _, field := range fields {
		nullable := " NOT NULL"
		if field.Nullable {
			nullable = ""
		}
		fmt.Fprintf(&b, ",\n    %s %s%s", field.Name, sqlType(field.Type), nullable)
	}
	b.WriteString("\n);\n")
	return b.String()
}

func fieldArgs(fields []Field) []string {
	args := make([]string, 0, len(fields))
	for _, field := range fields {
		args = append(args, "item."+goFieldName(field.Name))
	}
	return args
}

func sqlType(value string) string {
	switch strings.ToLower(value) {
	case "int", "integer":
		return "INTEGER"
	case "bool", "boolean":
		return "BOOLEAN"
	case "real", "float":
		return "REAL"
	default:
		return "TEXT"
	}
}

func renderContractTest(spec Spec) string {
	return fmt.Sprintf("// Code generated by hydrogo generator; add dialect and policy cases.\npackage generated\n\n// %sContractTest is intentionally a review placeholder.\n", exportName(spec.Name))
}

func renderAdminManifest(spec Spec) string {
	collection, err := AdminCollection(spec)
	if err != nil {
		return "{}\n"
	}
	data, err := json.MarshalIndent(collection, "", "  ")
	if err != nil {
		return "{}\n"
	}
	return string(append(data, '\n'))
}

func adminFieldKind(value string) adminstudio.FieldKind {
	switch strings.ToLower(value) {
	case "int", "integer", "real", "float":
		return adminstudio.FieldNumber
	case "bool", "boolean":
		return adminstudio.FieldBoolean
	case "date", "datetime", "timestamp":
		return adminstudio.FieldDate
	case "media", "upload", "file":
		return adminstudio.FieldMedia
	case "richtext", "rich_text":
		return adminstudio.FieldRichText
	case "select", "enum":
		return adminstudio.FieldSelect
	case "relation", "reference":
		return adminstudio.FieldRelation
	case "array", "list":
		return adminstudio.FieldArray
	case "blocks", "block":
		return adminstudio.FieldBlocks
	default:
		return adminstudio.FieldText
	}
}

func exportName(value string) string {
	parts := strings.Split(value, "_")
	for i, part := range parts {
		if part != "" {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func goFieldName(value string) string {
	parts := strings.Split(value, "_")
	for i, part := range parts {
		if part == "id" {
			parts[i] = "ID"
		} else if part != "" {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func goType(value string) string {
	switch strings.ToLower(value) {
	case "int", "integer":
		return "int64"
	case "bool", "boolean":
		return "bool"
	default:
		return "string"
	}
}
