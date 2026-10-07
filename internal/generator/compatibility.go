package generator

import (
	"bytes"
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
	ContractVersion  string
	SpecSHA256       string
	FilesChecked     int
}

var requiredGeneratedFiles = []string{"spec.json", "model.go", "repository.go", "relations_store.go", "service.go", "validation.go", "policy.go", "permissions.go", "transport.go", "runtime.go", "workflow.json", "relations.json", "form.html", "form.schema.json", "table.html", "openapi.json", "client.ts", "query.ts", "bulk.ts", "import-export.ts", "upload.ts", "react.ts", "swr.ts", "htmx.ts", "vanilla.ts", "kits.ts", "index.ts", "migration.sql", "migration.go", "migration.yaml", "contract_test.go", "transport_test.go", "fixture_test.go", "admin_manifest.json", "admin/views.json", "module.yaml", "REVIEW.md", "RUNTIME_WIRING.md"}

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
	if version := strings.TrimSpace(lock.ContractVersion); version != "" && version != CollectionContractVersion {
		return CompatibilityReport{}, fmt.Errorf("unsupported generated contract version %q", version)
	}
	resourceLock, hasResourceLock := lock.Resources[spec.Name]
	if hasResourceLock {
		lock.GeneratorVersion = resourceLock.GeneratorVersion
		lock.ContractVersion = resourceLock.ContractVersion
		lock.SpecSHA256 = resourceLock.SpecSHA256
		lock.ArtifactsSHA256 = resourceLock.ArtifactsSHA256
	}
	if lock.GeneratorVersion == "" || lock.SpecSHA256 == "" {
		return CompatibilityReport{}, errors.New("generator lock is incomplete")
	}
	contractVersion := strings.TrimSpace(lock.ContractVersion)
	if contractVersion == "" {
		// Locks created before contract_version was introduced represented v0.
		contractVersion = CollectionContractVersion
	}
	if contractVersion != CollectionContractVersion {
		return CompatibilityReport{}, fmt.Errorf("unsupported generated contract version %q", contractVersion)
	}
	collectionRoot := filepath.Join(root, "generated", spec.Name)
	specPath := filepath.Join(collectionRoot, "spec.json")
	if err := pathguard.RejectSymlinkComponents(collectionRoot); err != nil {
		return CompatibilityReport{}, fmt.Errorf("generated spec path: %w", err)
	}
	persistedSpec, err := LoadSpec(specPath)
	if err != nil {
		return CompatibilityReport{}, fmt.Errorf("load generated resource spec: %w", err)
	}
	persistedCanonical, err := canonicalSpec(persistedSpec)
	if err != nil {
		return CompatibilityReport{}, err
	}
	if !bytes.Equal(canonical, persistedCanonical) {
		return CompatibilityReport{}, errors.New("embedded resource spec does not match requested contract")
	}
	versionBytes, err := os.ReadFile(specPath)
	if err != nil {
		return CompatibilityReport{}, fmt.Errorf("read generated resource spec version: %w", err)
	}
	var document SpecDocument
	if err := json.Unmarshal(versionBytes, &document); err != nil {
		return CompatibilityReport{}, fmt.Errorf("read generated resource contract version: %w", err)
	}
	if document.Version != contractVersion {
		return CompatibilityReport{}, fmt.Errorf("generated lock contract version %q does not match embedded spec version %q", contractVersion, document.Version)
	}
	if lock.SpecSHA256 != expected {
		return CompatibilityReport{}, fmt.Errorf("generated spec hash mismatch: lock=%s expected=%s", lock.SpecSHA256, expected)
	}
	requiredFiles := append([]string(nil), requiredGeneratedFiles...)
	if spec.Versioned {
		requiredFiles = append(requiredFiles, "migration_2.sql", "migration_2.yaml")
	}
	if spec.Revisions {
		requiredFiles = append(requiredFiles, "revisions_migration.sql", "revisions_migration.yaml", "revision_contract.go")
	}
	if spec.Drafts {
		requiredFiles = append(requiredFiles, "drafts_migration.sql", "drafts_migration.yaml", "draft_contract.go")
	}
	if spec.Localization {
		requiredFiles = append(requiredFiles, "localization_migration_sqlite.sql", "localization_migration_postgres.sql", "localization_migration.yaml", "localization_contract.go")
	}
	for _, name := range requiredFiles {
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
	return CompatibilityReport{Collection: spec.Name, GeneratorVersion: lock.GeneratorVersion, ContractVersion: contractVersion, SpecSHA256: lock.SpecSHA256, FilesChecked: len(requiredFiles)}, nil
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
		if entry.Name() == "generator.lock" || entry.Name() == "admin_registry.json" || entry.Name() == "admin_registry.go" || entry.Name() == "admin_registration.go" || entry.Name() == "route_registration.go" {
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
	expectedRegistry, err := renderAdminRegistry(generatedRoot)
	if err != nil {
		return nil, fmt.Errorf("build generated Admin registry: %w", err)
	}
	registryPath := filepath.Join(generatedRoot, "admin_registry.json")
	registryInfo, err := os.Lstat(registryPath)
	if err != nil || registryInfo.Mode()&os.ModeSymlink != 0 || !registryInfo.Mode().IsRegular() {
		return nil, errors.New("generated Admin registry is missing or invalid")
	}
	actualRegistry, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, fmt.Errorf("read generated Admin registry: %w", err)
	}
	if !bytes.Equal(actualRegistry, expectedRegistry) {
		return nil, errors.New("generated Admin registry does not match resource specifications")
	}
	expectedRegistration, err := renderAdminRegistration(expectedRegistry)
	if err != nil {
		return nil, fmt.Errorf("build generated Admin registration: %w", err)
	}
	registrationPath := filepath.Join(generatedRoot, "admin_registration.go")
	registrationInfo, err := os.Lstat(registrationPath)
	if err != nil || registrationInfo.Mode()&os.ModeSymlink != 0 || !registrationInfo.Mode().IsRegular() {
		return nil, errors.New("generated Admin registration is missing or invalid")
	}
	actualRegistration, err := os.ReadFile(registrationPath)
	if err != nil || !bytes.Equal(actualRegistration, expectedRegistration) {
		return nil, errors.New("generated Admin registration does not match resource specifications")
	}
	expectedRoutes, hasModule, err := expectedGeneratedRouteRegistration(root, generatedRoot)
	if err != nil {
		return nil, fmt.Errorf("build generated resource route registration: %w", err)
	}
	routesPath := filepath.Join(generatedRoot, "route_registration.go")
	routesInfo, routesErr := os.Lstat(routesPath)
	if hasModule {
		if routesErr != nil || routesInfo.Mode()&os.ModeSymlink != 0 || !routesInfo.Mode().IsRegular() {
			return nil, errors.New("generated resource route registration is missing or invalid")
		}
		actualRoutes, err := os.ReadFile(routesPath)
		if err != nil || !bytes.Equal(actualRoutes, expectedRoutes) {
			return nil, errors.New("generated resource route registration does not match resource specifications")
		}
	} else if routesErr == nil {
		return nil, errors.New("generated resource route registration is stale without a Go module")
	} else if !os.IsNotExist(routesErr) {
		return nil, fmt.Errorf("inspect generated resource route registration: %w", routesErr)
	}
	registryGo, err := os.ReadFile(filepath.Join(generatedRoot, "admin_registry.go"))
	if err != nil || !bytes.Contains(registryGo, []byte("//go:embed admin_registry.json")) {
		return nil, errors.New("generated Admin registry embed is missing or invalid")
	}
	sort.Slice(reports, func(i, j int) bool { return reports[i].Collection < reports[j].Collection })
	return reports, nil
}

func expectedGeneratedRouteRegistration(root, generatedRoot string) ([]byte, bool, error) {
	packagePath, found, err := generatedPackageImportPath(root)
	if err != nil || !found {
		return nil, found, err
	}
	resources, err := generatedRouteResources(generatedRoot)
	if err != nil {
		return nil, false, err
	}
	source, err := renderGeneratedRouteRegistration(packagePath, resources)
	return source, true, err
}
