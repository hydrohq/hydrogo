package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/generator"
	"github.com/ardaozyildirim/hydrogo/internal/release"
)

func generateCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("generate requires a subcommand")
	}
	switch args[0] {
	case "module", "resource", "crud", "page", "form", "api", "admin":
		return generateModuleCommand(args[1:], stdout, stderr)
	case "scaffold":
		return generateScaffoldCommand(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unsupported generate target %q", args[0])
	}
}

// moduleCommand exposes the task-oriented spelling while retaining the
// historical `generate module` command for backwards compatibility.
func moduleCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || (args[0] != "create" && args[0] != "new") {
		return errors.New("module requires the create or new subcommand")
	}
	return generateModuleCommand(args[1:], stdout, stderr)
}

// resourceCommand is the collection-oriented spelling for CRUD scaffolding.
// It intentionally shares the same reviewable generator as module create.
func resourceCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("resource requires create, new or check subcommand")
	}
	if args[0] == "check" {
		return validateResourceCommand(args[1:], stdout, stderr)
	}
	if args[0] != "create" && args[0] != "new" {
		return errors.New("resource requires create, new or check subcommand")
	}
	if !containsString(args[1:], "--spec") {
		return resourceCreateFromFlags(args[1:], stdout, stderr)
	}
	return generateModuleCommand(args[1:], stdout, stderr)
}

func validateResourceCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("resource check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	specPath := flags.String("spec", "", "Versioned module spec (.yaml, .yml, or .json); omit to discover generated spec.json files")
	root := flags.String("root", "", "Generated project root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*root) == "" {
		return errors.New("resource check requires --root and optionally --spec")
	}
	if strings.TrimSpace(*specPath) != "" {
		spec, err := generator.LoadSpec(*specPath)
		if err != nil {
			return err
		}
		report, err := generator.CheckCompatibility(*root, spec)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "resource-compatible collection=%s contract=%s generator=%s spec_sha256=%s files=%d\n", report.Collection, report.ContractVersion, report.GeneratorVersion, report.SpecSHA256, report.FilesChecked)
		return err
	}
	reports, err := generator.CheckGeneratedCompatibility(*root)
	if err != nil {
		return err
	}
	for _, report := range reports {
		if _, err := fmt.Fprintf(stdout, "resource-compatible collection=%s contract=%s generator=%s spec_sha256=%s files=%d\n", report.Collection, report.ContractVersion, report.GeneratorVersion, report.SpecSHA256, report.FilesChecked); err != nil {
			return err
		}
	}
	return nil
}

// apiCommand exposes compatibility checks as a first-class SDK workflow while
// retaining `validate api` as the lower-level spelling.
func apiCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "diff" {
		return errors.New("api requires the diff subcommand")
	}
	return validateAPICommand(args[1:], stdout, stderr)
}

// schemaCommand compares signed release migration ledgers. It deliberately
// reports IDs/counts only; SQL dialect and migration bodies stay internal.
func schemaCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "diff" {
		return errors.New("schema requires the diff subcommand")
	}
	flags := flag.NewFlagSet("schema diff", flag.ContinueOnError)
	flags.SetOutput(stderr)
	baseline := flags.String("baseline", "", "Baseline signed release manifest")
	current := flags.String("current", "", "Current signed release manifest")
	publicKey := flags.String("public-key", "", "Trusted Ed25519 public key (base64)")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*baseline) == "" || strings.TrimSpace(*current) == "" || strings.TrimSpace(*publicKey) == "" {
		return errors.New("schema diff requires --baseline, --current and --public-key")
	}
	base, err := readVerifiedReleaseManifest(*baseline, *publicKey)
	if err != nil {
		return fmt.Errorf("verify baseline manifest: %w", err)
	}
	next, err := readVerifiedReleaseManifest(*current, *publicKey)
	if err != nil {
		return fmt.Errorf("verify current manifest: %w", err)
	}
	diff, err := release.CompareMigrationChecksums(base.MigrationChecksums, next.MigrationChecksums)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "schema-compatible added=%d removed=%d changed=%d\n", len(diff.Added), len(diff.Removed), len(diff.Changed))
	return err
}

func resourceCreateFromFlags(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("resource create", flag.ContinueOnError)
	flags.SetOutput(stderr)
	name := flags.String("name", "", "Resource name (snake_case)")
	table := flags.String("table", "", "Database table (defaults to resource name)")
	out := flags.String("out", "generated", "Output root for generated artifacts")
	adminLabel := flags.String("admin-label", "", "Admin collection label")
	var fieldArgs, permissionArgs, dialectArgs repeatedFlag
	flags.Var(&fieldArgs, "field", "Field name:type[:nullable]")
	flags.Var(&permissionArgs, "permission", "Permission name (repeatable)")
	flags.Var(&dialectArgs, "dialect", "Database dialect sqlite|postgres (repeatable)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*name) == "" || len(fieldArgs) == 0 {
		return errors.New("resource create requires --name and at least one --field")
	}
	if strings.TrimSpace(*table) == "" {
		*table = *name
	}
	fields := make([]generator.Field, 0, len(fieldArgs))
	for _, raw := range fieldArgs {
		parts := strings.Split(raw, ":")
		if len(parts) < 2 || len(parts) > 3 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return fmt.Errorf("invalid --field %q; expected name:type[:nullable]", raw)
		}
		nullable := len(parts) == 3 && strings.EqualFold(strings.TrimSpace(parts[2]), "nullable")
		fields = append(fields, generator.Field{Name: strings.TrimSpace(parts[0]), Type: strings.TrimSpace(parts[1]), Nullable: nullable, Admin: generator.FieldAdmin{Label: strings.TrimSpace(parts[0]), VisibleInList: true}})
	}
	permissions := append([]string(nil), permissionArgs...)
	if len(permissions) == 0 {
		permissions = []string{strings.TrimSpace(*name) + ".read", strings.TrimSpace(*name) + ".write"}
	}
	dialects := append([]string(nil), dialectArgs...)
	if len(dialects) == 0 {
		dialects = []string{"sqlite", "postgres"}
	}
	spec := generator.Spec{Name: strings.TrimSpace(*name), Table: strings.TrimSpace(*table), Fields: fields, Permissions: permissions, Dialects: dialects, Admin: generator.Admin{Label: strings.TrimSpace(*adminLabel), UseAsTitle: resourceCreateTitleField(fields)}}
	lock, err := generator.Generate(*out, spec)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "resource-created collection=%s out=%s spec_sha256=%s generator=%s\n", spec.Name, *out, lock.SpecSHA256, lock.GeneratorVersion)
	return err
}

func resourceCreateTitleField(fields []generator.Field) string {
	for _, field := range fields {
		if field.Name == "title" && (field.Type == "string" || field.Type == "text") {
			return field.Name
		}
	}
	for _, field := range fields {
		if field.Type == "string" || field.Type == "text" {
			return field.Name
		}
	}
	return ""
}

func generateModuleCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("generate module", flag.ContinueOnError)
	flags.SetOutput(stderr)
	specPath := flags.String("spec", "", "Versioned module spec (.yaml, .yml, or .json)")
	outputRoot := flags.String("out", "", "Output root for generated review artifacts")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*specPath) == "" || strings.TrimSpace(*outputRoot) == "" {
		return errors.New("generate module requires --spec and --out")
	}
	spec, err := generator.LoadSpec(*specPath)
	if err != nil {
		return err
	}
	lock, err := generator.Generate(*outputRoot, spec)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "generated collection=%s spec_sha256=%s generator=%s\n", spec.Name, lock.SpecSHA256, lock.GeneratorVersion)
	return err
}

func generateScaffoldCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("generate scaffold", flag.ContinueOnError)
	flags.SetOutput(stderr)
	blueprintPath := flags.String("blueprint", "", "Blueprint YAML/JSON path")
	outputRoot := flags.String("out", ".", "Client Edition output root")
	verify := flags.Bool("verify", true, "Run bounded post-generation gofmt and tests when a Go module is present")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*blueprintPath) == "" {
		return errors.New("generate scaffold requires --blueprint")
	}
	spec, err := generator.LoadBlueprint(*blueprintPath)
	if err != nil {
		return err
	}
	lock, err := generator.Generate(*outputRoot, spec)
	if err != nil {
		return err
	}
	customRoot := filepath.Join(*outputRoot, "custom")
	if err := os.MkdirAll(customRoot, 0o755); err != nil {
		return err
	}
	customReadme := filepath.Join(customRoot, spec.Name+"_policy.go")
	if _, err := os.Stat(customReadme); errors.Is(err, os.ErrNotExist) {
		content := fmt.Sprintf("// Client Edition policy for %s. Generated artifacts stay under generated/%s.\npackage custom\n\n// Add authorization, workflow, and tenant policy here.\n", spec.Name, spec.Name)
		if err := os.WriteFile(customReadme, []byte(content), 0o644); err != nil {
			return err
		}
	}
	hooksFile := filepath.Join(customRoot, spec.Name+"_hooks.go")
	if _, err := os.Stat(hooksFile); errors.Is(err, os.ErrNotExist) {
		content := fmt.Sprintf(`// Client Edition lifecycle hooks for %s.
// This file is human-owned; generated artifacts stay under generated/%s.
package custom


// %sHooks is an explicit extension point. Keep database, authorization and
// tenant decisions in the owning service; dispatch side effects through the
// durable outbox rather than performing network I/O inline.
type %sHooks struct{}

func (%sHooks) BeforeCreate(context.Context, any) error { return nil }
func (%sHooks) AfterCreate(context.Context, any) error  { return nil }
func (%sHooks) BeforeUpdate(context.Context, any, any) error { return nil }
func (%sHooks) AfterUpdate(context.Context, any, any) error  { return nil }
func (%sHooks) BeforeDelete(context.Context, any) error { return nil }
func (%sHooks) AfterDelete(context.Context, any) error  { return nil }
`, spec.Name, spec.Name, generatorExportName(spec.Name), generatorExportName(spec.Name), generatorExportName(spec.Name), generatorExportName(spec.Name), generatorExportName(spec.Name), generatorExportName(spec.Name), generatorExportName(spec.Name), generatorExportName(spec.Name))
		if err := os.WriteFile(hooksFile, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if *verify {
		if err := verifyGeneratedScaffold(*outputRoot); err != nil {
			return fmt.Errorf("generated scaffold verification: %w", err)
		}
	}
	_, err = fmt.Fprintf(stdout, "scaffold-created collection=%s generated=%s custom=%s spec_sha256=%s\n", spec.Name, filepath.Join(*outputRoot, "generated", spec.Name), customRoot, lock.SpecSHA256)
	return err
}

// generatorExportName mirrors the generator's stable Go identifier convention
// for scaffold-owned extension points without exposing generator internals to
// the CLI package.
func generatorExportName(value string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(value), func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

func verifyGeneratedScaffold(root string) error {
	var err error
	root, err = filepath.Abs(root)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	files := make([]string, 0)
	for _, dir := range []string{filepath.Join(root, "generated"), filepath.Join(root, "custom")} {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr == nil && info != nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
				files = append(files, path)
			}
			return nil
		})
	}
	if len(files) == 0 {
		return nil
	}
	formatArgs := append([]string{"-w"}, files...)
	format := exec.CommandContext(ctx, "gofmt", formatArgs...)
	format.Dir = root
	if output, err := format.CombinedOutput(); err != nil {
		return fmt.Errorf("gofmt: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	test := exec.CommandContext(ctx, "go", "test", "./...")
	test.Dir = root
	if output, err := test.CombinedOutput(); err != nil {
		return fmt.Errorf("go test: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func validateAPICommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("validate api", flag.ContinueOnError)
	flags.SetOutput(stderr)
	baseline := flags.String("baseline", "", "Baseline OpenAPI JSON document")
	current := flags.String("current", "", "Current OpenAPI JSON document")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*baseline) == "" || strings.TrimSpace(*current) == "" {
		return errors.New("validate api requires --baseline and --current")
	}
	report, err := generator.CheckAPICompatibility(*baseline, *current)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "api-compatible paths=%d methods=%d properties=%d\n", report.PathsChecked, report.MethodsChecked, report.PropertiesChecked)
	return err
}

func validateGeneratorCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("validate generator", flag.ContinueOnError)
	flags.SetOutput(stderr)
	specPath := flags.String("spec", "", "Versioned module spec (.yaml, .yml, or .json)")
	root := flags.String("root", "", "Generated project root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*specPath) == "" || strings.TrimSpace(*root) == "" {
		return errors.New("validate generator requires --spec and --root")
	}
	spec, err := generator.LoadSpec(*specPath)
	if err != nil {
		return err
	}
	report, err := generator.CheckCompatibility(*root, spec)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "compatible collection=%s contract=%s generator=%s spec_sha256=%s files=%d\n", report.Collection, report.ContractVersion, report.GeneratorVersion, report.SpecSHA256, report.FilesChecked)
	return err
}
