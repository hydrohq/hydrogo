package main

import "testing"

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
