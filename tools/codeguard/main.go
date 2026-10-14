// Command codeguard ratchets two code-shape rules for tracked Go sources:
//
//   - HTML markup must not live in Go string literals (use .templ files or
//     theme HTML instead);
//   - a file must not grow past maxLines;
//   - tests must not be disabled by renaming them (skip_TestX, xTestX): a
//     failing test is fixed, rewritten or deleted, never hidden.
//
// Existing violations are recorded in a baseline file. A baselined file may
// shrink but not get worse, and files not in the baseline must comply. Run
// with -update after reducing violations to tighten the baseline.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const maxLines = 800

// htmlTag matches an opening tag of an element that only appears in markup.
// disabledTest matches test functions renamed so `go test` ignores them.
var disabledTest = regexp.MustCompile(`(?m)^func ([A-Za-z]+_|x|X|_)(Test|Benchmark|Fuzz)[A-Za-z0-9_]*\(`)

var htmlTag = regexp.MustCompile(`<(?:!doctype|html|head|body|div|span|section|article|main|nav|header|footer|aside|button|form|input|label|select|textarea|table|thead|tbody|tr|td|th|ul|ol|li|p|a|h[1-6]|img|svg|style|script)[\s>/]`)

type entry struct {
	HTMLStrings int `json:"html_strings,omitempty"`
	Lines       int `json:"lines,omitempty"`
}

type baseline map[string]entry

func main() {
	baselinePath := flag.String("baseline", "tools/codeguard/baseline.json", "baseline file")
	update := flag.Bool("update", false, "rewrite the baseline from the current tree")
	flag.Parse()

	files, err := trackedGoFiles()
	if err != nil {
		fail(err)
	}
	current, err := measure(files)
	if err != nil {
		fail(err)
	}
	if *update {
		if err := write(*baselinePath, current); err != nil {
			fail(err)
		}
		fmt.Printf("codeguard: baseline updated (%d files)\n", len(current))
		return
	}
	known, err := read(*baselinePath)
	if err != nil {
		fail(err)
	}
	problems := compare(known, current)
	hidden, err := disabledTests()
	if err != nil {
		fail(err)
	}
	problems = append(problems, hidden...)
	for _, problem := range problems {
		fmt.Fprintln(os.Stderr, "codeguard:", problem)
	}
	if len(problems) != 0 {
		os.Exit(1)
	}
	fmt.Printf("codeguard: OK (%d files over budget, all within baseline)\n", len(current))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "codeguard:", err)
	os.Exit(1)
}

func trackedGoFiles() ([]string, error) {
	// Include untracked files too, so a new file is checked before it is added.
	out, err := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard", "*.go").Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var files []string
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if name == "" || strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, "_templ.go") || strings.HasPrefix(name, "docs/") {
			continue
		}
		files = append(files, filepath.ToSlash(name))
	}
	return files, nil
}

// measure returns only files that break a rule; compliant files are omitted.
func measure(files []string) (baseline, error) {
	result := baseline{}
	for _, name := range files {
		data, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		var e entry
		if lines := strings.Count(string(data), "\n"); lines > maxLines {
			e.Lines = lines
		}
		count, err := htmlStrings(name, data)
		if err != nil {
			return nil, err
		}
		e.HTMLStrings = count
		if e != (entry{}) {
			result[name] = e
		}
	}
	return result, nil
}

func htmlStrings(name string, data []byte) (int, error) {
	file, err := parser.ParseFile(token.NewFileSet(), name, data, parser.SkipObjectResolution)
	if err != nil {
		return 0, err
	}
	count := 0
	ast.Inspect(file, func(node ast.Node) bool {
		if lit, ok := node.(*ast.BasicLit); ok && lit.Kind == token.STRING && htmlTag.MatchString(strings.ToLower(lit.Value)) {
			count++
		}
		return true
	})
	return count, nil
}

func compare(known, current baseline) []string {
	var problems []string
	for name, now := range current {
		was, baselined := known[name]
		if now.HTMLStrings > was.HTMLStrings {
			if baselined && was.HTMLStrings > 0 {
				problems = append(problems, fmt.Sprintf("%s: HTML string literals grew from %d to %d; move markup into a .templ file", name, was.HTMLStrings, now.HTMLStrings))
			} else {
				problems = append(problems, fmt.Sprintf("%s: %d string literal(s) contain HTML; move markup into a .templ file or theme HTML", name, now.HTMLStrings))
			}
		}
		if now.Lines > 0 && now.Lines > max(was.Lines, maxLines) {
			if was.Lines > 0 {
				problems = append(problems, fmt.Sprintf("%s: grew from %d to %d lines; it is already over %d, split it instead of adding to it", name, was.Lines, now.Lines, maxLines))
			} else {
				problems = append(problems, fmt.Sprintf("%s: %d lines exceeds the %d-line limit; split the file", name, now.Lines, maxLines))
			}
		}
	}
	sort.Strings(problems)
	return problems
}

func read(path string) (baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result baseline
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return result, nil
}

func write(path string, value baseline) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// disabledTests reports test functions that were renamed to hide them.
func disabledTests() ([]string, error) {
	out, err := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard", "*_test.go").Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var problems []string
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if name == "" || strings.HasPrefix(name, "docs/") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		for _, match := range disabledTest.FindAllString(string(data), -1) {
			problems = append(problems, fmt.Sprintf("%s: disabled test %q; fix, rewrite or delete it instead of hiding it", filepath.ToSlash(name), strings.TrimSuffix(strings.TrimPrefix(match, "func "), "(")))
		}
	}
	return problems, nil
}
