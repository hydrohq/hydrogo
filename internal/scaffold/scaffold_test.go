package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
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
