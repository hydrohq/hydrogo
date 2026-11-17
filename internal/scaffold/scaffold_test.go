package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hydrohq/hydrogo/internal/config"
	"github.com/hydrohq/hydrogo/internal/contenttypes"
)

func TestEveryPresetScaffoldsAValidProject(t *testing.T) {
	for _, preset := range Presets() {
		t.Run(preset.Name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "site")
			if _, err := Create(Options{Dir: dir, Preset: preset.Name}); err != nil {
				t.Fatal(err)
			}
			types, err := contenttypes.LoadDir(filepath.Join(dir, "content-types"))
			if err != nil {
				t.Fatalf("content types do not load: %v", err)
			}
			if len(types) != len(preset.ContentTypes) {
				t.Fatalf("got %d content types, want %d", len(types), len(preset.ContentTypes))
			}
			cfg, err := config.Load(filepath.Join(dir, "config.yaml"), func(string) (string, bool) { return "", false }, config.Overrides{})
			if err != nil {
				t.Fatalf("config does not load: %v", err)
			}
			if len(cfg.Modules) != len(preset.Modules) {
				t.Fatalf("modules = %v, want %v", cfg.Modules, preset.Modules)
			}
		})
	}
}

func TestCreateWritesAnE2EHarness(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "site")
	if _, err := Create(Options{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		filepath.Join(dir, "e2e", "package.json"),
		filepath.Join(dir, "e2e", "playwright.config.ts"),
		filepath.Join(dir, "e2e", "README.md"),
		filepath.Join(dir, "e2e", "generated", "content", ".gitkeep"),
		filepath.Join(dir, "e2e", "custom", ".gitkeep"),
	} {
		if _, err := os.Stat(want); err != nil {
			t.Fatalf("expected %s to exist: %v", want, err)
		}
	}
	pkg, err := os.ReadFile(filepath.Join(dir, "e2e", "package.json"))
	if err != nil || !strings.Contains(string(pkg), "@playwright/test") {
		t.Fatalf("e2e/package.json missing @playwright/test: %v", err)
	}
}

func TestCreateRefusesToOverwriteExistingE2EDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "e2e"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(Options{Dir: dir}); err == nil {
		t.Fatal("expected an error when e2e/ already exists")
	}
}

func TestCreateRefusesToOverwrite(t *testing.T) {
	dir := t.TempDir()
	if _, err := Create(Options{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("custom"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(Options{Dir: dir}); err == nil {
		t.Fatal("expected an error when config.yaml already exists")
	}
	got, _ := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if string(got) != "custom" {
		t.Fatal("existing config was modified")
	}
}

func TestUnknownPreset(t *testing.T) {
	if _, err := Create(Options{Dir: t.TempDir(), Preset: "nope"}); err == nil {
		t.Fatal("expected unknown preset error")
	}
}

func TestStarterRejectionDirectsToCloud(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "site")
	_, err := Create(Options{Dir: dir, Preset: "blog", Starter: "sveltekit"})
	if err == nil || !strings.Contains(err.Error(), "Hydro Cloud") {
		t.Fatalf("expected starter rejection redirecting to Hydro Cloud, got %v", err)
	}
}

func TestCRMPresetLogsAdminWrites(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "crm")
	if _, err := Create(Options{Dir: dir, Preset: "crm"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil || !strings.Contains(string(body), "log_request_bodies: true") {
		t.Fatalf("the crm config must log admin writes: %v\n%s", err, body)
	}
}
