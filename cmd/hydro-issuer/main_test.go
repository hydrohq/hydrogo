package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIssuerGeneratesAndIssuesLicense(t *testing.T) {
	dir := t.TempDir()
	pubPath := filepath.Join(dir, "public.key")
	privPath := filepath.Join(dir, "private.key")
	licensePath := filepath.Join(dir, "hydro.lic")
	var out bytes.Buffer
	if err := run([]string{"keygen", "--public-out", pubPath, "--private-out", privPath}, &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "keys-generated") {
		t.Fatalf("unexpected keygen output: %s", out.String())
	}
	out.Reset()
	if err := run([]string{"issue", "--customer", "Acme", "--project-id", "proj-1", "--private-key", privPath, "--out", licensePath}, &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "license-issued") {
		t.Fatalf("unexpected issue output: %s", out.String())
	}
	if data, err := os.ReadFile(licensePath); err != nil || !bytes.Contains(data, []byte(`"claims"`)) {
		t.Fatalf("license was not written: err=%v", err)
	}
}

func TestIssuerRejectsUnexpectedArguments(t *testing.T) {
	if err := run([]string{"keygen", "unexpected"}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected unexpected argument error")
	}
}
