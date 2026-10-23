package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/testgen"
)

// generateCommand dispatches "hydro generate <noun>". Only "tests" exists
// today; the verb is namespaced so other generators (for example a future
// schema or client regenerator) can land as siblings without a new top-level
// command.
func generateCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "tests" {
		return errors.New("generate requires the tests subcommand: hydro generate tests")
	}
	return generateTestsCommand(args[1:], stdout, stderr)
}

// generateTestsCommand writes one Playwright spec per content type into
// <out>/content/, proving every admin role gets exactly the read/write
// access its permissions declare. See internal/testgen's package doc for
// scope. Safe to run repeatedly: every file it writes starts with a
// "DO NOT EDIT" header and nothing outside <out>/content/ is touched.
func generateTestsCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("generate tests", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("dir", "content-types", "content types directory")
	out := flags.String("out", filepath.Join("e2e", "generated"), "output directory")
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("generate tests takes no positional arguments")
	}

	// loadMaintenanceConfig lets --config override content_types_dir; --dir
	// wins when both are given, matching `hydro content-type`'s convention.
	contentTypesDir := *dir
	if !flagWasSet(flags, "dir") {
		if cfg, err := loadMaintenanceConfig(flags, values); err == nil && cfg.ContentTypesDir != "" {
			contentTypesDir = cfg.ContentTypesDir
		}
	}

	definitions, err := contenttypes.LoadDir(contentTypesDir)
	if err != nil {
		return fmt.Errorf("load content types: %w", err)
	}
	if len(definitions) == 0 {
		fmt.Fprintf(stdout, "no content types in %s; nothing generated\n", contentTypesDir)
		return nil
	}

	files := testgen.Generate(definitions)
	contentDir := filepath.Join(*out, "content")
	if err := os.MkdirAll(contentDir, 0o755); err != nil { // #nosec G301 -- project directory; the service account may differ from the author
		return err
	}
	for _, file := range files {
		path := filepath.Join(*out, filepath.FromSlash(file.Path))
		if err := os.WriteFile(path, []byte(file.Content), 0o644); err != nil { // #nosec G306 -- project test file with no secrets (credentials come from HYDRO_E2E_* env vars)
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	fmt.Fprintf(stdout, "generated %d file(s) in %s from %d content type(s)\n", len(files), contentDir, len(definitions))
	return nil
}

func flagWasSet(flags *flag.FlagSet, name string) bool {
	set := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}
