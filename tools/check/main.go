// Command check runs the fast local verification every change must pass:
// build, vet, formatting, repository guards and the test suite. It is the
// single entrypoint behind `make check` and the pre-push hook, and works the
// same on Linux, macOS and Windows.
//
//	go run ./tools/check            # everything
//	go run ./tools/check -short     # skip the test suite
//	go run ./tools/check -e2e       # also the browser journeys (needs Playwright's Chromium)
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type step struct {
	name string
	dir  string
	args []string
	// check inspects command output for steps that signal failure by
	// printing instead of exiting non-zero (gofmt -l).
	check func(output string) error
}

func main() {
	short := flag.Bool("short", false, "skip the test suite")
	e2e := flag.Bool("e2e", false, "also run the browser journeys (needs Playwright's Chromium: npx playwright install chromium)")
	audit := flag.Bool("audit", false, "also scan Go and npm dependencies for known vulnerabilities (needs network access)")
	flag.Parse()

	steps := []step{
		{name: "build", args: []string{"go", "build", "./..."}},
		{name: "vet", args: []string{"go", "vet", "./..."}},
		{name: "gofmt", args: []string{"gofmt", "-l", "cmd", "internal", "modules", "extension", "tools", "tests", "migrations"}, check: func(output string) error {
			if strings.TrimSpace(output) != "" {
				return fmt.Errorf("run gofmt -w on:\n%s", output)
			}
			return nil
		}},
		{name: "code shape (HTML in Go strings, file size)", args: []string{"go", "run", "./tools/codeguard"}},
		{name: "migrations", args: []string{"go", "run", "./tools/migrationguard", "-root", "migrations"}},
		{name: "module boundaries", args: []string{"go", "run", "./tools/moduleguard", "-root", "modules", "-allow", "modules/api", "-dependencies", "modules/dependencies.json",
			"-deny-modules-root", "internal",
			// Pre-existing debt (found by the 2026-10-24 audit, not yet fixed):
			// internal/mailer/sqlstore.Spool.Enqueue takes a modules/site type
			// directly. New code must not add a second one of these; see
			// docs/decisions for the commerce module's own write-up of why it
			// deliberately did not reuse this spool.
			"-deny-modules-exceptions", "internal/mailer/sqlstore/spool.go",
		}},
	}
	if !*short {
		steps = append(steps, step{name: "tests", args: []string{"go", "test", "./...", "-count=1", "-timeout=10m"}},
			// The real binary against a throw-away project: the security promises and
			// the API's speed on 10,000 records. About 20 seconds, no credentials.
			step{name: "smoke (real server: security and performance)", args: []string{"go", "test", "-tags", "smoke", "./tests/smoke", "-count=1", "-timeout=10m"}},
			step{name: "side-service example", dir: "examples/side-service", args: []string{"go", "test", "./...", "-count=1"}})

		npm := "npm"
		if runtime.GOOS == "windows" {
			npm = "npm.cmd"
		}
		if _, err := exec.LookPath(npm); err == nil {
			steps = append(steps,
				step{name: "admin svelte-check", dir: "admin", args: []string{npm, "run", "check"}},
				step{name: "admin unit tests", dir: "admin", args: []string{npm, "test"}},
				step{name: "admin build", dir: "admin", args: []string{npm, "run", "build"}},
			)
			steps = append(steps, starterSteps(npm)...)
			if *e2e {
				steps = append(steps, e2eSteps(npm)...)
			}
		}
	}

	if *audit {
		steps = append(steps, auditSteps()...)
	}

	started := time.Now()
	for _, s := range steps {
		fmt.Printf("==> %s\n", s.name)
		cmd := exec.Command(s.args[0], s.args[1:]...) // #nosec G204 -- developer tool running the project's own checks, no shell
		if s.dir != "" {
			cmd.Dir = s.dir
		}
		output, err := cmd.CombinedOutput()
		if err == nil && s.check != nil {
			err = s.check(string(output))
		}
		if err != nil {
			_, _ = os.Stdout.Write(output)
			fmt.Fprintf(os.Stderr, "FAIL: %s: %v\n", s.name, err)
			os.Exit(1)
		}
	}
	fmt.Printf("PASS: all checks passed in %s\n", time.Since(started).Round(time.Second))
}

// starterSteps type-checks, tests and builds the starter frontends that
// `hydro new --starter ...` copies into new projects. They are the first thing a
// new user sees, so a broken one must stop a change like any other.
func starterSteps(npm string) []step {
	starters := []struct {
		dir   string
		tests bool
	}{{"examples/crm-starter", true}, {"examples/commerce-starter", false}, {"examples/sveltekit-starter", false}, {"examples/hydro-website", true}}
	var steps []step
	for _, s := range starters {
		if _, err := os.Stat(s.dir + "/node_modules"); err != nil {
			steps = append(steps, step{name: s.dir + " install", dir: s.dir, args: []string{npm, "ci"}})
		}
		steps = append(steps, step{name: s.dir + " svelte-check", dir: s.dir, args: []string{npm, "run", "check"}})
		if s.tests {
			steps = append(steps, step{name: s.dir + " unit tests", dir: s.dir, args: []string{npm, "test"}})
		}
		steps = append(steps, step{name: s.dir + " build", dir: s.dir, args: []string{npm, "run", "build"}})
	}
	return steps
}

// auditSteps scans for known vulnerabilities. It needs network access to the
// Go vulnerability database and the npm registry, so it is opt-in (-audit);
// the scheduled security workflow runs the same commands.
func auditSteps() []step {
	steps := []step{{name: "govulncheck", args: []string{"go", "run", "golang.org/x/vuln/cmd/govulncheck@v1.8.0", "./..."}}}
	npm := "npm"
	if runtime.GOOS == "windows" {
		npm = "npm.cmd"
	}
	if _, err := exec.LookPath(npm); err != nil {
		return steps
	}
	for _, dir := range []string{"admin", "e2e", "packages/client", "examples/sveltekit-starter", "examples/crm-starter", "examples/commerce-starter"} {
		steps = append(steps, step{name: "npm audit " + dir, dir: dir, args: []string{npm, "audit", "--omit=dev", "--audit-level=high"}})
	}
	// examples/hydropost-news-starter uses pnpm (pnpm-lock.yaml, no
	// package-lock.json), so npm audit there would just fail with ENOLOCK
	// instead of actually scanning anything.
	pnpm := "pnpm"
	if runtime.GOOS == "windows" {
		pnpm = "pnpm.cmd"
	}
	if exec.Command(pnpm, "--version").Run() == nil {
		steps = append(steps, step{name: "pnpm audit examples/hydropost-news-starter", dir: "examples/hydropost-news-starter", args: []string{pnpm, "audit", "--prod", "--audit-level", "high"}})
	}
	return steps
}

// e2eSteps runs the browser journeys in e2e/journeys: real Chromium against a
// real server and the CRM starter, started and stopped by the suite itself.
func e2eSteps(npm string) []step {
	var steps []step
	if _, err := os.Stat("e2e/node_modules"); err != nil {
		steps = append(steps, step{name: "e2e install", dir: "e2e", args: []string{npm, "ci"}})
	}
	return append(steps, step{name: "e2e browser journeys", dir: "e2e", args: []string{npm, "run", "journeys"}})
}
