// Package generator implements the deterministic, deny-by-default collection
// generator described by ADR-0010. It emits reviewable skeletons only.
package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	registry, err := adminstudio.NewRegistry([]adminstudio.Collection{{ID: spec.Name, Label: label, Description: spec.Admin.Description, Fields: fields, Actions: actions}})
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
		"migration.sql":       renderMigration(spec),
		"contract_test.go":    renderContractTest(spec),
		"fixture_test.go":     renderFixtureTest(spec),
		"admin_manifest.json": renderAdminManifest(spec),
		"module.yaml":         renderModuleManifest(spec),
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
		return adminstudio.FieldStatus
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
