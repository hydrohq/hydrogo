package prompts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPromptRegistry_Embedded(t *testing.T) {
	p, err := Load("copilot_system")
	if err != nil {
		t.Fatalf("Load(copilot_system) failed: %v", err)
	}
	if p.ID != "copilot_system" {
		t.Errorf("expected ID 'copilot_system', got %q", p.ID)
	}
	if p.Version == "" {
		t.Errorf("expected non-empty version")
	}
	if len(p.Hash) != 64 {
		t.Errorf("expected sha256 hex (64 chars), got %q", p.Hash)
	}

	rendered := p.Render(map[string]string{
		"TENANT_ID": "acme-tenant",
	})
	if !contains(rendered, `"acme-tenant"`) {
		t.Errorf("Render failed to replace {TENANT_ID}: %s", rendered)
	}
}

func TestPromptRegistry_OverrideDir(t *testing.T) {
	tempDir := t.TempDir()
	customPrompt := `ID: custom_test
VERSION: 2.1.0
DESCRIPTION: Custom test prompt
---
Hello {NAME}, welcome to {PLACE}.`

	if err := os.WriteFile(filepath.Join(tempDir, "custom_test.prompt"), []byte(customPrompt), 0o644); err != nil {
		t.Fatalf("failed to write temp prompt: %v", err)
	}

	t.Setenv("HYDRO_PROMPTS_DIR", tempDir)

	p, err := Load("custom_test")
	if err != nil {
		t.Fatalf("Load(custom_test) with override failed: %v", err)
	}
	if p.ID != "custom_test" {
		t.Errorf("expected ID 'custom_test', got %q", p.ID)
	}
	if p.Version != "2.1.0" {
		t.Errorf("expected Version '2.1.0', got %q", p.Version)
	}

	rendered := p.Render(map[string]string{
		"NAME":  "Arda",
		"PLACE": "HydroGo",
	})
	if rendered != "Hello Arda, welcome to HydroGo." {
		t.Errorf("unexpected rendered output: %q", rendered)
	}
}

func TestPromptRegistry_List(t *testing.T) {
	list, err := List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}
	if len(list) < 4 {
		t.Errorf("expected at least 4 embedded prompts, got %d", len(list))
	}
}

func TestPromptRegistry_NotFound(t *testing.T) {
	_, err := Load("non_existent_prompt_12345")
	if err == nil {
		t.Errorf("expected error for non-existent prompt")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || filepath.Base(s) != "" && len(s) > 0 && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
