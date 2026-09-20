package main

import (
	"os"
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
