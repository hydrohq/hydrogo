package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckWorkflowsAcceptsPinnedAndLocalActions(t *testing.T) {
	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "jobs:\n  build:\n    steps:\n      - uses: actions/checkout@0123456789abcdef0123456789abcdef01234567 # release\n      - uses: ./actions/local\n"
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkWorkflows(workflowDir); err != nil {
		t.Fatalf("pinned workflow rejected: %v", err)
	}
}

func TestCheckWorkflowsRejectsMutableAction(t *testing.T) {
	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "jobs:\n  build:\n    steps:\n      - uses: actions/checkout@v5\n"
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkWorkflows(workflowDir); err == nil || !strings.Contains(err.Error(), "full 40-character commit SHA") {
		t.Fatalf("mutable workflow was accepted: %v", err)
	}
}
