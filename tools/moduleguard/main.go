// Command moduleguard enforces the declared direct dependency graph between
// HydroGo modules. Composition packages may be explicitly allowlisted.
package main

import (
	"encoding/json"
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
	allow := flag.String("allow", "modules/api", "comma-separated aggregate roots allowed to import sibling modules")
	dependenciesPath := flag.String("dependencies", "modules/dependencies.json", "declared direct module dependency graph")
	flag.Parse()

	allowed := map[string]bool{}
	for _, item := range strings.Split(*allow, ",") {
		if strings.TrimSpace(item) != "" {
			allowed[filepath.ToSlash(filepath.Clean(strings.TrimSpace(item)))] = true
		}
	}
	dependencies, err := readDependencies(*dependenciesPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	violations, err := checkModules(*root, allowed, dependencies)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		fmt.Fprintln(os.Stderr, "moduleguard: module dependency contract failed:")
		for _, violation := range violations {
			fmt.Fprintln(os.Stderr, " -", violation)
		}
		os.Exit(1)
	}
	fmt.Println("moduleguard: OK")
}

func readDependencies(path string) (map[string]map[string]bool, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- developer tool reading the module files it walked
	if err != nil {
		return nil, fmt.Errorf("read module dependencies %s: %w", path, err)
	}
	var raw map[string][]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode module dependencies %s: %w", path, err)
	}
	dependencies := make(map[string]map[string]bool, len(raw))
	for module, imports := range raw {
		module = filepath.ToSlash(filepath.Clean(module))
		dependencies[module] = make(map[string]bool, len(imports))
		for _, imported := range imports {
			imported = filepath.ToSlash(filepath.Clean(imported))
			if dependencies[module][imported] {
				return nil, fmt.Errorf("duplicate module dependency %s -> %s", module, imported)
			}
			dependencies[module][imported] = true
		}
	}
	return dependencies, nil
}

func checkModules(root string, allowed map[string]bool, dependencies map[string]map[string]bool) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read module root %s: %w", root, err)
	}
	modules := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			modules[filepath.ToSlash(filepath.Join(root, entry.Name()))] = true
		}
	}

	var violations []string
	graph := make(map[string][]string, len(dependencies))
	for from, imports := range dependencies {
		if !modules[from] {
			violations = append(violations, "dependency declaration has unknown module "+from)
		}
		for to := range imports {
			if !modules[to] {
				violations = append(violations, fmt.Sprintf("dependency declaration %s -> %s names an unknown module", from, to))
				continue
			}
			if from == to {
				violations = append(violations, "self dependency declared for "+from)
				continue
			}
			graph[from] = append(graph[from], to)
		}
	}
	violations = append(violations, dependencyCycles(graph)...)

	used := make(map[string]bool)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
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
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) == 0 || parts[0] == "." {
			return nil
		}
		from := filepath.ToSlash(filepath.Join(root, parts[0]))
		if allowed[from] {
			return nil
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")
			if !strings.HasPrefix(importPath, modulePrefix) {
				continue
			}
			rest := strings.TrimPrefix(importPath, modulePrefix)
			to := filepath.ToSlash(filepath.Join(root, strings.Split(rest, "/")[0]))
			if to == from {
				continue
			}
			if !dependencies[from][to] {
				violations = append(violations, fmt.Sprintf("%s: undeclared module dependency %s -> %s", path, from, to))
				continue
			}
			used[from+"\x00"+to] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for from, imports := range dependencies {
		for to := range imports {
			if modules[from] && modules[to] && !used[from+"\x00"+to] {
				violations = append(violations, fmt.Sprintf("declared module dependency %s -> %s is unused", from, to))
			}
		}
	}
	return violations, nil
}

func dependencyCycles(graph map[string][]string) []string {
	state := make(map[string]uint8)
	var cycles []string
	var visit func(string, []string)
	visit = func(module string, path []string) {
		if state[module] == 2 {
			return
		}
		if state[module] == 1 {
			start := 0
			for index, item := range path {
				if item == module {
					start = index
					break
				}
			}
			cycles = append(cycles, "module dependency cycle: "+strings.Join(append(path[start:], module), " -> "))
			return
		}
		state[module] = 1
		path = append(path, module)
		for _, dependency := range graph[module] {
			visit(dependency, path)
		}
		state[module] = 2
	}
	modules := make([]string, 0, len(graph))
	for module := range graph {
		modules = append(modules, module)
	}
	sort.Strings(modules)
	for _, module := range modules {
		visit(module, nil)
	}
	return cycles
}
