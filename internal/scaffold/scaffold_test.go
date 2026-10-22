package scaffold

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	sveltekitstarter "github.com/ardaozyildirim/hydrogo/examples/sveltekit-starter"
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

func TestCreateCopiesFrontendStarter(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "site")
	if _, err := Create(Options{Dir: dir, Preset: "blog", Starter: "sveltekit"}); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"frontend/package.json", "frontend/src/app.html", "frontend/src/lib/hydrogo.ts", "frontend/src/routes/[[lang=locale]]/+page.svelte", "frontend/.env.example"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
	if _, err := Create(Options{Dir: filepath.Join(t.TempDir(), "x"), Starter: "nope"}); err == nil {
		t.Error("unknown starter accepted")
	}
}

func TestCreateCopiesCRMStarter(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "crm")
	if _, err := Create(Options{Dir: dir, Preset: "crm", Starter: "crm"}); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"frontend/package.json", "frontend/src/hooks.server.ts", "frontend/src/routes/deals/+page.svelte", "frontend/scripts/seed.mjs", "frontend/.env.example"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}

// The starter vendors packages/client so it installs without a registry; the
// two copies must not drift.
func TestStarterVendoredClientMatchesPackage(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("..", "..", "packages", "client", "src", "hydrogo.ts"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := fs.ReadFile(sveltekitstarter.FS, "src/lib/hydrogo.ts")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Error("examples/sveltekit-starter/src/lib/hydrogo.ts differs from packages/client/src/hydrogo.ts; copy it over")
	}
}
