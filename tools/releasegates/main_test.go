package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAcceptsCandidateWithPendingExternalGates(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "evidence.md"), []byte("evidence"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{
		FormatVersion: 1, GeneratedDate: "2026-09-19",
		Gates: []Gate{
			{ID: "local", Category: "automated", Status: "verified", Evidence: []Evidence{{Path: "evidence.md"}}},
			{ID: "restore", Category: "external", Status: "pending", Note: "requires a clean target"},
		},
		Profiles: []Profile{{Name: "standard", Claim: "candidate", RequiredGates: []string{"local", "restore"}}},
	}
	if err := validate(manifest, root); err != nil {
		t.Fatalf("candidate should validate: %v", err)
	}
}

func TestValidateRejectsReadyProfileWithPendingGate(t *testing.T) {
	manifest := Manifest{
		FormatVersion: 1, GeneratedDate: "2026-09-19",
		Gates:    []Gate{{ID: "restore", Category: "external", Status: "pending"}},
		Profiles: []Profile{{Name: "standard", Claim: "ready", RequiredGates: []string{"restore"}}},
	}
	if err := validate(manifest, t.TempDir()); err == nil {
		t.Fatal("ready profile with pending gate must fail")
	}
}

func TestValidateRejectsMissingEvidence(t *testing.T) {
	manifest := Manifest{
		FormatVersion: 1, GeneratedDate: "2026-09-19",
		Gates: []Gate{{ID: "local", Category: "automated", Status: "verified", Evidence: []Evidence{{Path: "missing.md"}}}},
	}
	if err := validate(manifest, t.TempDir()); err == nil {
		t.Fatal("missing evidence must fail")
	}
}
