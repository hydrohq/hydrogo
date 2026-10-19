package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCommandScaffoldsAndBootstrapsAnAdmin(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	var stdout, stderr bytes.Buffer
	err := run([]string{"new", "site", "--preset", "docs", "--admin-email", "admin@example.com", "--admin-password-stdin"}, strings.NewReader("Quartz-Meadow-4417-lantern\n"), &stdout, &stderr)
	if err != nil {
		t.Fatalf("hydro new: %v\n%s", err, stderr.String())
	}
	for _, path := range []string{"config.yaml", "content-types/faq.yaml", "data/hydro.db", "README.md"} {
		if _, statErr := os.Stat(filepath.Join(root, "site", path)); statErr != nil {
			t.Errorf("missing %s: %v", path, statErr)
		}
	}
	if !strings.Contains(stdout.String(), "Created admin user: admin@example.com") {
		t.Errorf("admin was not created:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "Generated admin password") {
		t.Error("a password read from stdin must not be echoed back")
	}
}

func TestNewCommandRejectsBadArguments(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{{"new"}, {"new", "a", "b"}, {"init", "extra"}, {"new", "x", "--preset", "nope"}} {
		var out bytes.Buffer
		if err := run(args, strings.NewReader(""), &out, &out); err == nil {
			t.Errorf("hydro %v should fail", args)
		}
	}
}

func TestContentTypeNewThenValidate(t *testing.T) {
	t.Chdir(t.TempDir())
	var out bytes.Buffer
	if err := run([]string{"content-type", "new", "recipes"}, strings.NewReader(""), &out, &out); err != nil {
		t.Fatalf("new: %v\n%s", err, out.String())
	}
	out.Reset()
	if err := run([]string{"content-type", "validate"}, strings.NewReader(""), &out, &out); err != nil {
		t.Fatalf("validate: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "ok  recipes") {
		t.Errorf("unexpected output:\n%s", out.String())
	}
	if err := run([]string{"content-type", "new", "recipes"}, strings.NewReader(""), &out, &out); err == nil {
		t.Error("creating an existing content type must fail")
	}
	if err := run([]string{"content-type", "new", "Bad-ID"}, strings.NewReader(""), &out, &out); err == nil {
		t.Error("an invalid id must be rejected")
	}
}
