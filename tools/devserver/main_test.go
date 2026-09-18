package main

import (
	"os"
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
