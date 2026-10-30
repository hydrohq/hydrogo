package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContentTypeCLIJSONAndDryRun(t *testing.T) {
	tempDir := t.TempDir()
	typesDir := filepath.Join(tempDir, "content-types")
	if err := os.MkdirAll(typesDir, 0o750); err != nil {
		t.Fatal(err)
	}

	// 1. Create a content type with "new"
	var stdout, stderr bytes.Buffer
	if err := run([]string{"content-type", "new", "posts", "--dir=" + typesDir}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("content-type new failed: %v (%s)", err, stderr.String())
	}

	// 2. Test "list --json"
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"content-type", "list", "--dir=" + typesDir, "--json"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("content-type list --json failed: %v (%s)", err, stderr.String())
	}

	var listOutput []struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &listOutput); err != nil {
		t.Fatalf("unmarshal list output: %v (%s)", err, stdout.String())
	}
	if len(listOutput) != 1 || listOutput[0].ID != "posts" {
		t.Fatalf("expected 1 content type 'posts', got: %+v", listOutput)
	}

	// 3. Test "validate --json"
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"content-type", "validate", "--dir=" + typesDir, "--json"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("content-type validate --json failed: %v (%s)", err, stderr.String())
	}

	var valOutput struct {
		Valid bool `json:"valid"`
		Count int  `json:"count"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &valOutput); err != nil {
		t.Fatalf("unmarshal validate output: %v", err)
	}
	if !valOutput.Valid || valOutput.Count != 1 {
		t.Fatalf("expected valid=true, count=1, got: %+v", valOutput)
	}

	// 4. Test "apply --file ... --dry-run --json"
	recipeYaml := `id: recipes
label: Recipes
public: true
fields:
  - id: title
    label: Title
    kind: text
    required: true
  - id: ingredients
    label: Ingredients
    kind: richtext
`
	sourceFile := filepath.Join(tempDir, "recipes_source.yaml")
	if err := os.WriteFile(sourceFile, []byte(recipeYaml), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"content-type", "apply", "--file=" + sourceFile, "--dir=" + typesDir, "--dry-run", "--json"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("content-type apply --dry-run failed: %v (%s)", err, stderr.String())
	}

	var dryRunOutput struct {
		DryRun      bool   `json:"dry_run"`
		Action      string `json:"action"`
		ContentType string `json:"content_type"`
		Applied     bool   `json:"applied"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &dryRunOutput); err != nil {
		t.Fatalf("unmarshal apply dry-run: %v", err)
	}
	if !dryRunOutput.DryRun || dryRunOutput.Applied || dryRunOutput.Action != "create" || dryRunOutput.ContentType != "recipes" {
		t.Fatalf("unexpected dry-run output: %+v", dryRunOutput)
	}

	// Verify target file was NOT created during dry-run
	targetFile := filepath.Join(typesDir, "recipes.yaml")
	if _, err := os.Stat(targetFile); !os.IsNotExist(err) {
		t.Fatalf("expected recipes.yaml not to exist after dry-run, stat err: %v", err)
	}

	// 5. Test real "apply --file ... --json"
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"content-type", "apply", "--file=" + sourceFile, "--dir=" + typesDir, "--json"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("content-type apply failed: %v (%s)", err, stderr.String())
	}

	var applyOutput struct {
		DryRun      bool   `json:"dry_run"`
		Action      string `json:"action"`
		ContentType string `json:"content_type"`
		Applied     bool   `json:"applied"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &applyOutput); err != nil {
		t.Fatalf("unmarshal apply output: %v", err)
	}
	if applyOutput.DryRun || !applyOutput.Applied || applyOutput.ContentType != "recipes" {
		t.Fatalf("unexpected apply output: %+v", applyOutput)
	}

	// Verify target file exists now
	if _, err := os.Stat(targetFile); err != nil {
		t.Fatalf("expected recipes.yaml to exist after apply: %v", err)
	}

	// 6. Test "diff --json"
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"content-type", "diff", "--dir=" + typesDir, "--json"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("content-type diff --json failed: %v (%s)", err, stderr.String())
	}

	var diffOutput struct {
		Drift        bool `json:"drift"`
		ContentTypes []struct {
			ID string `json:"id"`
		} `json:"content_types"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &diffOutput); err != nil {
		t.Fatalf("unmarshal diff output: %v", err)
	}
	if diffOutput.Drift || len(diffOutput.ContentTypes) != 2 {
		t.Fatalf("expected 2 content types in diff output, got: %+v", diffOutput)
	}
}
