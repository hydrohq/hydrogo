package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type CompatibilityReport struct {
	Collection       string
	GeneratorVersion string
	SpecSHA256       string
	FilesChecked     int
}

var requiredGeneratedFiles = []string{"model.go", "repository.go", "service.go", "policy.go", "transport.go", "form.html", "table.html", "openapi.json", "client.ts", "query.ts", "bulk.ts", "import-export.ts", "migration.sql", "contract_test.go", "fixture_test.go", "admin_manifest.json", "module.yaml", "REVIEW.md"}

// CheckCompatibility verifies generated artifacts without rewriting them.
func CheckCompatibility(root string, spec Spec) (CompatibilityReport, error) {
	if err := Validate(spec); err != nil {
		return CompatibilityReport{}, err
	}
	canonical, err := canonicalSpec(spec)
	if err != nil {
		return CompatibilityReport{}, err
	}
	hash := sha256.Sum256(canonical)
	expected := hex.EncodeToString(hash[:])
	lockData, err := os.ReadFile(filepath.Join(root, "generated", "generator.lock"))
	if err != nil {
		return CompatibilityReport{}, fmt.Errorf("read generator lock: %w", err)
	}
	var lock Lock
	if err := json.Unmarshal(lockData, &lock); err != nil {
		return CompatibilityReport{}, fmt.Errorf("parse generator lock: %w", err)
	}
	if lock.GeneratorVersion == "" || lock.SpecSHA256 == "" {
		return CompatibilityReport{}, errors.New("generator lock is incomplete")
	}
	if lock.SpecSHA256 != expected {
		return CompatibilityReport{}, fmt.Errorf("generated spec hash mismatch: lock=%s expected=%s", lock.SpecSHA256, expected)
	}
	collectionRoot := filepath.Join(root, "generated", spec.Name)
	for _, name := range requiredGeneratedFiles {
		info, err := os.Lstat(filepath.Join(collectionRoot, name))
		if err != nil {
			return CompatibilityReport{}, fmt.Errorf("missing generated artifact %s: %w", name, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return CompatibilityReport{}, fmt.Errorf("generated artifact %s is not a regular file", name)
		}
	}
	return CompatibilityReport{Collection: spec.Name, GeneratorVersion: lock.GeneratorVersion, SpecSHA256: lock.SpecSHA256, FilesChecked: len(requiredGeneratedFiles)}, nil
}
