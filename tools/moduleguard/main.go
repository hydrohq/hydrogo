// Command moduleguard prevents accidental direct dependencies between HydroGo
// modules. Composition/aggregate packages may be explicitly allowlisted.
package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const modulePrefix = "github.com/ardaozyildirim/hydrogo/modules/"

func main() {
	root := flag.String("root", "modules", "module source root")
	allow := flag.String("allow", "modules/api", "comma-separated module roots allowed to import sibling modules")
	flag.Parse()
	allowed := map[string]bool{}
	for _, item := range strings.Split(*allow, ",") {
		if strings.TrimSpace(item) != "" {
			allowed[strings.TrimSpace(item)] = true
		}
	}

	var violations []string
	err := filepath.WalkDir(*root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		rel, err := filepath.Rel(*root, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) == 0 || parts[0] == "." {
			return nil
		}
		from := filepath.ToSlash(filepath.Join(*root, parts[0]))
		if allowed[from] {
			return nil
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")
			if !strings.HasPrefix(importPath, modulePrefix) {
				continue
			}
			rest := strings.TrimPrefix(importPath, modulePrefix)
			to := filepath.ToSlash(filepath.Join(*root, strings.Split(rest, "/")[0]))
			if to != from {
				violations = append(violations, fmt.Sprintf("%s: %s -> %s", path, from, to))
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		fmt.Fprintln(os.Stderr, "moduleguard: forbidden direct module imports:")
		for _, violation := range violations {
			fmt.Fprintln(os.Stderr, " -", violation)
		}
		os.Exit(1)
	}
	fmt.Println("moduleguard: OK")
}
