package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDeploymentPresetsDeclareExplicitModuleSets(t *testing.T) {
	want := map[string][]string{
		"compact-site.yaml": {"core", "auth", "members", "pages", "media", "site"},
		"editorial.yaml":    {"core", "auth", "members", "pages", "media", "site", "news"},
		"headless.yaml":     {"core", "auth", "pages", "media", "site", "news"},
	}
	for name, modules := range want {
		path := filepath.Join("..", "..", "config", "presets", name)
		lookup := func(key string) (string, bool) {
			if key == "HYDRO_DATABASE_DSN" {
				return "file:./data/hydro.db?_busy_timeout=5000&_journal_mode=WAL", true
			}
			return "", false
		}
		cfg, err := Load(path, lookup, Overrides{})
		if err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
		if !reflect.DeepEqual(cfg.Modules, modules) {
			t.Fatalf("%s modules = %#v, want %#v", name, cfg.Modules, modules)
		}
		if cfg.Database.Dialect != "sqlite" || cfg.Profile != "compact" {
			t.Fatalf("%s must default to compact SQLite, got profile=%q dialect=%q", name, cfg.Profile, cfg.Database.Dialect)
		}
	}
}
