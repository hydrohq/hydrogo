package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"version"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("run version: %v", err)
	}
	if got := stdout.String(); got != version+"\n" {
		t.Fatalf("version output = %q, want %q", got, version+"\n")
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"unknown"}, strings.NewReader(""), &stdout, &stderr)
	if err == nil {
		t.Fatal("run unknown command succeeded")
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("usage missing from stderr: %q", stderr.String())
	}
}

func TestRunMigrateAndCreateAdmin(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	var stdout, stderr bytes.Buffer

	// 1. Run migrate
	err := run([]string{"migrate", "--database-dialect", "sqlite", "--database-dsn", dbPath}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("migrate failed: %v, stderr: %s", err, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"upgrade", "check", "--database-dialect", "sqlite", "--database-dsn", dbPath, "--capacity-profile", "compact"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("upgrade check failed: %v, stderr: %s", err, stderr.String())
	}
	if got := stdout.String(); got != "upgrade-ready profile=compact database=sqlite\n" {
		t.Fatalf("unexpected upgrade output: %q", got)
	}

	// 2. Run create-admin
	stdout.Reset()
	stderr.Reset()
	err = run([]string{
		"create-admin",
		"--email", "admin@example.com",
		"--password-stdin",
		"--database-dialect", "sqlite",
		"--database-dsn", dbPath,
	}, strings.NewReader("StrongPassword123!\n"), &stdout, &stderr)

	if err != nil {
		t.Fatalf("create-admin failed: %v, stderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Created admin user: admin@example.com") {
		t.Fatalf("unexpected create-admin output: %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"doctor", "--database-dialect", "sqlite", "--database-dsn", dbPath, "--capacity-profile", "compact"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("doctor failed: %v, stderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "profile=compact database=sqlite") {
		t.Fatalf("unexpected doctor output: %s", stdout.String())
	}
}
