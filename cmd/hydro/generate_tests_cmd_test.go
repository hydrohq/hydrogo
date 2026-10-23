package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateTestsWritesOneSpecPerContentTypePlusHelpers(t *testing.T) {
	dir := t.TempDir()
	typesDir := filepath.Join(dir, "content-types")
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "id: widgets\nlabel: Widgets\nfields:\n  - {id: name, label: Name, kind: text, required: true}\n"
	if err := os.WriteFile(filepath.Join(typesDir, "widgets.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "e2e", "generated")

	var stdout, stderr bytes.Buffer
	if err := run([]string{"generate", "tests", "--dir", typesDir, "--out", out}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("generate tests failed: %v, stderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "generated 2 file(s)") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}

	helpers, err := os.ReadFile(filepath.Join(out, "content", "_helpers.ts"))
	if err != nil || !strings.Contains(string(helpers), "DO NOT EDIT") {
		t.Fatalf("helpers.ts missing or not generated: %v", err)
	}
	spec, err := os.ReadFile(filepath.Join(out, "content", "widgets.spec.ts"))
	if err != nil {
		t.Fatalf("widgets.spec.ts not written: %v", err)
	}
	if !strings.Contains(string(spec), `PATH = "/api/v1/admin/content/widgets"`) {
		t.Fatalf("spec does not target widgets: %s", spec)
	}
}

func TestGenerateTestsRejectsUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"generate", "nope"}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("expected an error for an unknown generate subcommand")
	}
}

func TestGenerateTestsWithNoContentTypesGeneratesNothing(t *testing.T) {
	dir := t.TempDir()
	typesDir := filepath.Join(dir, "content-types")
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "e2e", "generated")

	var stdout, stderr bytes.Buffer
	if err := run([]string{"generate", "tests", "--dir", typesDir, "--out", out}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("generate tests failed: %v, stderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "nothing generated") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(out, "content")); !os.IsNotExist(err) {
		t.Fatalf("expected no output directory to be created, stat err = %v", err)
	}
}
