package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartLocalDependenciesNone(t *testing.T) {
	cleanup, redisURL, err := startLocalDependencies(t.TempDir(), "none", "unused.yml")
	if err != nil {
		t.Fatalf("none mode: %v", err)
	}
	cleanup()
	if redisURL != "" {
		t.Fatalf("none mode redis URL = %q", redisURL)
	}
}

func TestTemplSourceDetectionSkipsGeneratedAndDependencyFolders(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".tmp", "dev"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".tmp", "dev", "ignored.templ"), []byte("ignored"), 0o600); err != nil {
		t.Fatal(err)
	}
	if hasTemplSources(root) {
		t.Fatal("generated dev cache should not enable the Templ watcher")
	}
	if err := os.MkdirAll(filepath.Join(root, "views"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "views", "page.templ"), []byte("templ Page() {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !hasTemplSources(root) {
		t.Fatal("Templ source should enable generation/watch mode")
	}
}

func TestEmbeddedCSSTriggersBuildButEditionThemeCSSReloadsFromDisk(t *testing.T) {
	if isAsset("/repo/modules/designsystem/assets/designsystem.css") {
		t.Fatal("embedded module CSS must rebuild into the binary")
	}
	if !isAsset("/repo/themes/corporate/static/theme.css") {
		t.Fatal("edition theme CSS should be served from disk")
	}
	if watched("/repo/modules/ui/page.templ") {
		t.Fatal("Templ source must wait for generated Go output before rebuilding")
	}
}

func TestStartLocalDependenciesRejectsUnknownMode(t *testing.T) {
	if _, _, err := startLocalDependencies(t.TempDir(), "sometimes", "unused.yml"); err == nil {
		t.Fatal("expected unknown dependency mode error")
	}
}

func TestEnvBoolDisabled(t *testing.T) {
	for _, value := range []string{"0", "false", "no", "off"} {
		t.Setenv("HYDRO_DEV_INDICATOR", value)
		if !envBoolDisabled("HYDRO_DEV_INDICATOR") {
			t.Fatalf("%q should disable the indicator", value)
		}
	}
	for _, value := range []string{"", "1", "true", "yes", "on"} {
		if err := os.Setenv("HYDRO_DEV_INDICATOR", value); err != nil {
			t.Fatal(err)
		}
		if envBoolDisabled("HYDRO_DEV_INDICATOR") {
			t.Fatalf("%q should keep the indicator enabled", value)
		}
	}
}

func TestDevClientKeepsReadyPillCompact(t *testing.T) {
	for _, want := range []string{"HYDROGO", "hdev-phase-tag", "HydroGo Development Status", "Show HydroGo development status"} {
		if !strings.Contains(devClient, want) {
			t.Fatalf("dev client missing %q", want)
		}
	}
	if !strings.Contains(devClient, `hdev-value" hidden`) {
		t.Fatal("dev client should keep detailed status out of the compact pill")
	}
}
