// Command importguard ensures that Client Editions only import declared
// HydroGo public surfaces. It does not restrict standard library or unrelated
// third-party dependencies.
package main

import (
	"errors"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	root := flag.String("root", "", "Client Edition source directory")
	module := flag.String("module", "github.com/ardaozyildirim/hydrogo", "HydroGo module path")
	allowed := flag.String("allow", "github.com/ardaozyildirim/hydrogo/extension", "comma-separated allowed HydroGo import prefixes")
	flag.Parse()

	if *root == "" {
		fmt.Fprintln(os.Stderr, "importguard: -root is required")
		os.Exit(2)
	}

	violations, err := inspect(*root, *module, splitAllowed(*allowed))
	if err != nil {
		fmt.Fprintln(os.Stderr, "importguard:", err)
		os.Exit(1)
	}
	if len(violations) != 0 {
		for _, violation := range violations {
			fmt.Fprintln(os.Stderr, "importguard:", violation)
		}
		os.Exit(1)
	}
}

func inspect(root, module string, allowed []string) ([]string, error) {
	if module == "" {
		return nil, errors.New("module path is required")
	}

	var violations []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return fmt.Errorf("parse import in %s: %w", path, err)
			}
			if isHydroImport(importPath, module) && !matchesAllowed(importPath, allowed) {
				relative, relErr := filepath.Rel(root, path)
				if relErr != nil {
					relative = path
				}
				violations = append(violations, fmt.Sprintf("%s imports forbidden package %s", relative, importPath))
			}
		}
		return nil
	})
	return violations, err
}

func isHydroImport(importPath, module string) bool {
	return importPath == module || strings.HasPrefix(importPath, module+"/")
}

func matchesAllowed(importPath string, allowed []string) bool {
	for _, prefix := range allowed {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	return false
}

func splitAllowed(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
