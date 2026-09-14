package v1

import (
	"fmt"
	"sort"
	"strings"
)

// ValidateDependencyGraph verifies that module dependencies are known and
// acyclic. The input is module name -> required module names. Names are
// compared case-insensitively after trimming whitespace.
func ValidateDependencyGraph(graph map[string][]string) error {
	normalized := make(map[string][]string, len(graph))
	for rawName, rawDeps := range graph {
		name := normalizeModuleName(rawName)
		if name == "" {
			return fmt.Errorf("module name cannot be empty")
		}
		if _, exists := normalized[name]; exists {
			return fmt.Errorf("duplicate module name %q", name)
		}
		deps := make([]string, 0, len(rawDeps))
		seen := make(map[string]struct{}, len(rawDeps))
		for _, rawDep := range rawDeps {
			dep := normalizeModuleName(rawDep)
			if dep == "" {
				return fmt.Errorf("module %q has an empty dependency", name)
			}
			if _, exists := seen[dep]; exists {
				continue
			}
			seen[dep] = struct{}{}
			deps = append(deps, dep)
		}
		sort.Strings(deps)
		normalized[name] = deps
	}

	for name, deps := range normalized {
		for _, dep := range deps {
			if _, exists := normalized[dep]; !exists {
				return fmt.Errorf("module %q depends on unknown module %q", name, dep)
			}
		}
	}

	state := make(map[string]uint8, len(normalized))
	var visit func(string, []string) error
	visit = func(name string, path []string) error {
		switch state[name] {
		case 1:
			return fmt.Errorf("module dependency cycle: %s -> %s", strings.Join(path, " -> "), name)
		case 2:
			return nil
		}
		state[name] = 1
		for _, dep := range normalized[name] {
			if err := visit(dep, append(path, name)); err != nil {
				return err
			}
		}
		state[name] = 2
		return nil
	}
	for name := range normalized {
		if err := visit(name, nil); err != nil {
			return err
		}
	}
	return nil
}

func normalizeModuleName(name string) string { return strings.ToLower(strings.TrimSpace(name)) }
