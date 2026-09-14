package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectVendorReferencesFindsUnreferencedRuntimeAsset(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "static", "vendor", "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "static", "vendor", "demo", "used.js"), []byte("window.used=true"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "static", "vendor", "demo", "unused.css"), []byte(".unused{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal.go"), []byte(`package fixture
const asset = "/vendor/demo/used.js"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	unused, err := inspectVendorReferences(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(unused) != 1 || unused[0] != "static/vendor/demo/unused.css" {
		t.Fatalf("unused = %v", unused)
	}
}

func TestInspectVendorReferencesAllowsNoVendorDirectory(t *testing.T) {
	if unused, err := inspectVendorReferences(t.TempDir()); err != nil || len(unused) != 0 {
		t.Fatalf("missing vendor root = %v/%v", unused, err)
	}
}
