package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"version"}, &stdout, &stderr); err != nil {
		t.Fatalf("run version: %v", err)
	}
	if got := stdout.String(); got != version+"\n" {
		t.Fatalf("version output = %q, want %q", got, version+"\n")
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"unknown"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run unknown command succeeded")
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("usage missing from stderr: %q", stderr.String())
	}
}
