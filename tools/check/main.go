// Command check runs the fast local verification every change must pass:
// build, vet, formatting, repository guards and the test suite. It is the
// single entrypoint behind `make check` and the pre-push hook, and works the
// same on Linux, macOS and Windows.
//
//	go run ./tools/check            # everything
//	go run ./tools/check -short     # skip the test suite
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type step struct {
	name string
	args []string
	// check inspects command output for steps that signal failure by
	// printing instead of exiting non-zero (gofmt -l).
	check func(output string) error
}

func main() {
	short := flag.Bool("short", false, "skip the test suite")
	flag.Parse()

	steps := []step{
		{name: "build", args: []string{"go", "build", "./..."}},
		{name: "vet", args: []string{"go", "vet", "./..."}},
		{name: "gofmt", args: []string{"gofmt", "-l", "cmd", "internal", "modules", "extension", "tools", "tests", "migrations", "static"}, check: func(output string) error {
			if strings.TrimSpace(output) != "" {
				return fmt.Errorf("run gofmt -w on:\n%s", output)
			}
			return nil
		}},
		{name: "code shape (HTML in Go strings, file size)", args: []string{"go", "run", "./tools/codeguard"}},
		{name: "migrations", args: []string{"go", "run", "./tools/migrationguard", "-root", "migrations"}},
		{name: "module boundaries", args: []string{"go", "run", "./tools/moduleguard", "-root", "modules", "-allow", "modules/api", "-dependencies", "modules/dependencies.json"}},
		{name: "theme asset versions", args: []string{"go", "run", "./tools/assetguard", "-root", "."}},
	}
	if !*short {
		steps = append(steps, step{name: "tests", args: []string{"go", "test", "./...", "-count=1", "-timeout=10m"}})
	}

	started := time.Now()
	for _, s := range steps {
		fmt.Printf("==> %s\n", s.name)
		cmd := exec.Command(s.args[0], s.args[1:]...)
		output, err := cmd.CombinedOutput()
		if err == nil && s.check != nil {
			err = s.check(string(output))
		}
		if err != nil {
			os.Stdout.Write(output)
			fmt.Fprintf(os.Stderr, "FAIL: %s: %v\n", s.name, err)
			os.Exit(1)
		}
	}
	fmt.Printf("PASS: all checks passed in %s\n", time.Since(started).Round(time.Second))
}
