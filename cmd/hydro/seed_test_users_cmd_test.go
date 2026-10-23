package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestSeedTestUsersCreatesOneAccountPerRoleIdempotently(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	var stdout, stderr bytes.Buffer

	if err := run([]string{"migrate", "--database-dialect", "sqlite", "--database-dsn", dbPath}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("migrate failed: %v, stderr: %s", err, stderr.String())
	}

	stdout.Reset()
	if err := run([]string{"seed-test-users", "--database-dialect", "sqlite", "--database-dsn", dbPath}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("seed-test-users failed: %v, stderr: %s", err, stderr.String())
	}
	first := stdout.String()
	for _, want := range []string{
		"HYDRO_E2E_ADMIN_EMAIL=test-admin@hydrogo.local",
		"HYDRO_E2E_EDITOR_EMAIL=test-editor@hydrogo.local",
		"HYDRO_E2E_VIEWER_EMAIL=test-viewer@hydrogo.local",
	} {
		if !strings.Contains(first, want) {
			t.Fatalf("seed-test-users output missing %q:\n%s", want, first)
		}
	}

	// Rerunning must not error (idempotent) and should rotate the password,
	// not fail with "user already exists".
	stdout.Reset()
	if err := run([]string{"seed-test-users", "--database-dialect", "sqlite", "--database-dsn", dbPath}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("second seed-test-users run failed: %v, stderr: %s", err, stderr.String())
	}
	second := stdout.String()
	if second == first {
		t.Fatal("second run printed the same passwords; expected rotated passwords")
	}
}
