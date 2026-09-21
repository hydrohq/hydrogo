package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/pathguard"
)

type CompatibilityReport struct {
	Collection       string
	GeneratorVersion string
	SpecSHA256       string
	FilesChecked     int
}

var requiredGeneratedFiles = []string{"spec.json", "model.go", "repository.go", "service.go", "policy.go", "transport.go", "form.html", "form.schema.json", "table.html", "openapi.json", "client.ts", "query.ts", "bulk.ts", "import-export.ts", "upload.ts", "react.ts", "swr.ts", "htmx.ts", "vanilla.ts", "kits.ts", "index.ts", "migration.sql", "migration.yaml", "contract_test.go", "transport_test.go", "fixture_test.go", "admin_manifest.json", "module.yaml", "REVIEW.md", "RUNTIME_WIRING.md"}

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
		artifactPath := filepath.Join(collectionRoot, name)
		info, err := os.Lstat(artifactPath)
		if err != nil {
			return CompatibilityReport{}, fmt.Errorf("missing generated artifact %s: %w", name, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return CompatibilityReport{}, fmt.Errorf("generated artifact %s is not a regular file", name)
		}
		if len(lock.ArtifactsSHA256) > 0 {
			expectedArtifact, ok := lock.ArtifactsSHA256[name]
			if !ok || expectedArtifact == "" {
				return CompatibilityReport{}, fmt.Errorf("generator lock has no checksum for artifact %s", name)
			}
			data, err := os.ReadFile(artifactPath)
			if err != nil {
				return CompatibilityReport{}, fmt.Errorf("read generated artifact %s: %w", name, err)
			}
			digest := sha256.Sum256(data)
			if actual := hex.EncodeToString(digest[:]); actual != expectedArtifact {
				return CompatibilityReport{}, fmt.Errorf("generated artifact %s checksum mismatch", name)
			}
		}
	}
	return CompatibilityReport{Collection: spec.Name, GeneratorVersion: lock.GeneratorVersion, SpecSHA256: lock.SpecSHA256, FilesChecked: len(requiredGeneratedFiles)}, nil
}

// CheckGeneratedCompatibility discovers every generated collection beneath
// root and validates it from its embedded spec.json. This keeps project checks
// dependency-free and makes the generated spec the single source for a
// reviewable resource set.
func CheckGeneratedCompatibility(root string) ([]CompatibilityReport, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("generated root is required")
	}
	generatedRoot := filepath.Join(root, "generated")
	if err := pathguard.RejectSymlinkComponents(generatedRoot); err != nil {
		return nil, fmt.Errorf("generated root: %w", err)
	}
	entries, err := os.ReadDir(generatedRoot)
	if err != nil {
		return nil, fmt.Errorf("read generated root: %w", err)
	}
	reports := make([]CompatibilityReport, 0, len(entries))
	for _, entry := range entries {
		if entry.Name() == "generator.lock" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("inspect generated entry %s: %w", entry.Name(), err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, fmt.Errorf("generated collection %s is not a real directory", entry.Name())
		}
		collectionRoot := filepath.Join(generatedRoot, entry.Name())
		if err := pathguard.RejectSymlinkComponents(collectionRoot); err != nil {
			return nil, fmt.Errorf("generated collection %s: %w", entry.Name(), err)
		}
		spec, err := LoadSpec(filepath.Join(collectionRoot, "spec.json"))
		if err != nil {
			return nil, fmt.Errorf("load generated collection %s spec: %w", entry.Name(), err)
		}
		report, err := CheckCompatibility(root, spec)
		if err != nil {
			return nil, fmt.Errorf("check generated collection %s: %w", entry.Name(), err)
		}
		reports = append(reports, report)
	}
	if len(reports) == 0 {
		return nil, errors.New("generated root contains no collections")
	}
	sort.Slice(reports, func(i, j int) bool { return reports[i].Collection < reports[j].Collection })
	return reports, nil
}
