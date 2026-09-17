package projectgen

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	themev1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/config"
)

func TestGenerateAllPresets(t *testing.T) {
	presets := []string{"news", "commerce", "corporate", "portal", "headless"}

	for _, preset := range presets {
		t.Run(preset, func(t *testing.T) {
			tempDir := t.TempDir()
			projectName := "test-" + preset + "-project"

			projectPath, err := Generate(Options{
				Root:     tempDir,
				Name:     projectName,
				Preset:   preset,
				Profile:  "compact",
				Database: "sqlite",
				Demo:     true,
				E2E:      true,
			})
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			// 1. Verify required files exist
			requiredFiles := []string{
				"config.yaml",
				"hydrogo.lock",
				"README.md",
				"PROJECT_STATUS.md",
				"TASKS.md",
				"LICENSE",
				"NOTICE",
				"docs/ARCHITECTURE.md",
				"docs/OPERATIONS.md",
				"docs/E2E_GUIDE.md",
				"UPGRADE.md",
				"e2e/manifest.json",
				"frontend/starter.json",
				"frontend/README.md",
				"frontend/contracts.ts",
				"custom/README.md",
				"migrations/README.md",
				"seed/README.md",
			}

			for _, file := range requiredFiles {
				fullPath := filepath.Join(projectPath, filepath.FromSlash(file))
				if _, err := os.Stat(fullPath); err != nil {
					t.Errorf("expected required file %q to exist: %v", file, err)
				}
			}
			contracts, err := os.ReadFile(filepath.Join(projectPath, "frontend", "contracts.ts"))
			if err != nil {
				t.Fatalf("read generated frontend contracts: %v", err)
			}
			for _, marker := range []string{"HYDRO_CONTRACT_VERSION = 'v1'", "EditorialWorkflow", "FormDefinition", "SearchQuery", "CapabilityKit", "AnalyticsConsent"} {
				if !strings.Contains(string(contracts), marker) {
					t.Fatalf("generated frontend contracts missing %q", marker)
				}
			}
			var frontendManifest struct {
				DesignSystem  string   `json:"design_system"`
				DesignSystems []string `json:"design_systems"`
			}
			manifestBytes, err := os.ReadFile(filepath.Join(projectPath, "frontend", "starter.json"))
			if err != nil {
				t.Fatalf("read frontend starter manifest: %v", err)
			}
			if err := json.Unmarshal(manifestBytes, &frontendManifest); err != nil {
				t.Fatalf("decode frontend starter manifest: %v", err)
			}
			if frontendManifest.DesignSystem != "native" {
				t.Fatalf("default frontend design system = %q, want native", frontendManifest.DesignSystem)
			}
			if !contains(frontendManifest.DesignSystems, "shadcn") || !contains(frontendManifest.DesignSystems, "material") {
				t.Fatalf("frontend design system adapters = %#v", frontendManifest.DesignSystems)
			}

			// 2. Verify lockfile JSON parsing
			lockBytes, err := os.ReadFile(filepath.Join(projectPath, "hydrogo.lock"))
			if err != nil {
				t.Fatalf("read hydrogo.lock error: %v", err)
			}
			var lock Lock
			if err := json.Unmarshal(lockBytes, &lock); err != nil {
				t.Fatalf("unmarshal hydrogo.lock error: %v", err)
			}
			if lock.Preset != preset {
				t.Errorf("expected preset %q in lock, got %q", preset, lock.Preset)
			}
			if lock.ProjectName != projectName {
				t.Errorf("expected project name %q in lock, got %q", projectName, lock.ProjectName)
			}

			// 3. Verify config.yaml parsing
			cfgPath := filepath.Join(projectPath, "config.yaml")
			cfg, err := config.Load(cfgPath, nil, config.Overrides{})
			if err != nil {
				t.Fatalf("config.Load error: %v", err)
			}
			if cfg.Theme == "" {
				t.Errorf("expected non-empty theme in config")
			}

			// 4. Verify theme bundle loads via extension/v1.LoadTheme
			if preset != "headless" {
				themeDir := filepath.Join(projectPath, "themes")
				bundle, err := themev1.LoadTheme(themeDir, cfg.Theme)
				if err != nil {
					t.Fatalf("LoadTheme(%q) failed: %v", cfg.Theme, err)
				}
				if bundle.Manifest.Name != cfg.Theme {
					t.Errorf("expected theme name %q, got %q", cfg.Theme, bundle.Manifest.Name)
				}
			}

			// 5. Verify Core source files are NOT copied
			coreInternalPath := filepath.Join(projectPath, "internal")
			if _, err := os.Stat(coreInternalPath); !os.IsNotExist(err) {
				t.Errorf("Core internal/ source directory must never be copied into generated project")
			}
		})
	}
}

func TestFrontendManifestCarriesSelectedDesignSystem(t *testing.T) {
	project, err := Generate(Options{Root: t.TempDir(), Name: "shadcn-project", Preset: "dashboard", DesignSystem: "shadcn"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(project, "frontend", "starter.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		DesignSystem string `json:"design_system"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.DesignSystem != "shadcn" {
		t.Fatalf("selected design system = %q, want shadcn", manifest.DesignSystem)
	}
}

func TestGenerateMinimalProductionPresetAliases(t *testing.T) {
	for _, preset := range []string{"compact-site", "editorial"} {
		t.Run(preset, func(t *testing.T) {
			project, err := Generate(Options{Root: t.TempDir(), Name: "minimal-" + preset, Preset: preset, Profile: "compact", Database: "sqlite"})
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := config.Load(filepath.Join(project, "config.yaml"), nil, config.Overrides{})
			if err != nil {
				t.Fatal(err)
			}
			if preset == "compact-site" && (contains(cfg.Modules, "commerce") || contains(cfg.Modules, "news")) {
				t.Fatalf("compact-site enabled an unrelated module: %#v", cfg.Modules)
			}
			if preset == "editorial" && contains(cfg.Modules, "commerce") {
				t.Fatalf("editorial enabled commerce: %#v", cfg.Modules)
			}
		})
	}
}

func TestGenerateProductPresetAliases(t *testing.T) {
	for _, preset := range []string{"membership", "dashboard", "erp"} {
		project, err := Generate(Options{Root: t.TempDir(), Name: "preset-" + preset, Preset: preset, Profile: "compact", Database: "sqlite"})
		if err != nil {
			t.Fatalf("%s: %v", preset, err)
		}
		cfg, err := config.Load(filepath.Join(project, "config.yaml"), nil, config.Overrides{})
		if err != nil {
			t.Fatalf("%s config: %v", preset, err)
		}
		if len(cfg.Modules) == 0 {
			t.Fatalf("%s has no modules", preset)
		}
	}
}

func TestGenerateNormalizesMixedCasePreset(t *testing.T) {
	project, err := Generate(Options{Root: t.TempDir(), Name: "mixed-case", Preset: "NeWs", Profile: "compact", Database: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(filepath.Join(project, "config.yaml"), nil, config.Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(cfg.Modules, "news") || contains(cfg.Modules, "commerce") {
		t.Fatalf("mixed-case news selected wrong modules: %#v", cfg.Modules)
	}
	lockBytes, err := os.ReadFile(filepath.Join(project, "hydrogo.lock"))
	if err != nil {
		t.Fatal(err)
	}
	var lock Lock
	if err := json.Unmarshal(lockBytes, &lock); err != nil {
		t.Fatal(err)
	}
	if lock.Preset != "news" {
		t.Fatalf("normalized lock preset = %q, want news", lock.Preset)
	}
}

func TestGenerateNormalizesBuildOptions(t *testing.T) {
	project, err := Generate(Options{Root: t.TempDir(), Name: "mixed-options", Preset: "news", Profile: "STANDARD", Database: "SQLITE", Presentation: "INTEGRATED"})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(filepath.Join(project, "config.yaml"), nil, config.Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if string(cfg.Profile) != "standard" || cfg.Database.Dialect != "sqlite" {
		t.Fatalf("normalized options produced profile=%q database=%q", cfg.Profile, cfg.Database.Dialect)
	}
}

func TestPresetFrontendRoutesAreDeterministicAndCovered(t *testing.T) {
	tests := map[string][]string{
		"news": {"home", "news_room", "member_login"}, "commerce": {"home", "shop", "cart"},
		"corporate": {"home"}, "portal": {"home"}, "headless": {"home"},
	}
	for preset, wantNames := range tests {
		first, second := presetFrontendRoutes(preset), presetFrontendRoutes(preset)
		if len(first) != len(wantNames) || len(second) != len(first) {
			t.Fatalf("preset %q route count = %d/%d, want %d", preset, len(first), len(second), len(wantNames))
		}
		for i, want := range wantNames {
			if first[i]["name"] != want || first[i]["name"] != second[i]["name"] || first[i]["path"] != second[i]["path"] || first[i]["test_id"] != second[i]["test_id"] {
				t.Fatalf("preset %q route %d is not deterministic/expected: %#v", preset, i, first[i])
			}
		}
	}
}

func TestFrontendBuildManifestOverridesPresetDeterministically(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "frontend.json")
	if err := os.WriteFile(manifest, []byte(`{"routes":[{"name":"zeta","path":"/zeta","test_id":"route-z"},{"name":"alpha","path":"/alpha","test_id":"route-a"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	project, err := Generate(Options{Root: root, Name: "manifest-project", Preset: "news", FrontendManifest: manifest})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(project, "e2e", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		Frontend struct {
			Routes []map[string]any `json:"routes"`
		} `json:"frontend"`
	}
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatal(err)
	}
	if len(output.Frontend.Routes) != 2 || output.Frontend.Routes[0]["name"] != "alpha" || output.Frontend.Routes[1]["name"] != "zeta" {
		t.Fatalf("routes not sorted/ingested: %#v", output.Frontend.Routes)
	}
}

func TestFrontendBuildManifestRejectsUnsafeRoutes(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "frontend.json")
	if err := os.WriteFile(manifest, []byte(`{"routes":[{"name":"bad","path":"//evil","test_id":"bad"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(Options{Root: root, Name: "unsafe-manifest", Preset: "news", FrontendManifest: manifest}); err == nil {
		t.Fatal("expected unsafe frontend route to be rejected")
	}
}

func TestGeneratedHeadlessClientUsesPublishedEndpoints(t *testing.T) {
	projectPath, err := Generate(Options{Root: t.TempDir(), Name: "headless-client", Preset: "headless"})
	if err != nil {
		t.Fatal(err)
	}
	client, err := os.ReadFile(filepath.Join(projectPath, "client", "hydrogo.ts"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(client)
	for _, expected := range []string{"/api/v1/pages/by-slug?locale=", "/api/v1/news?locale=", "AbortController", "response_too_large", "timeoutMs", "getReader", "releaseLock", "MAX_RESPONSE_BYTES", "HydroMutationOptions", "HydroRequestOptions", "baseUrl must be an absolute HTTP(S) URL", "signal: options.signal", "requestId", "x-request-id", "validateIdempotencyKey", "Idempotency-Key", "async mutation", "raw.trim() === ''", "undefined as T", "abortFromCaller", "callerSignal?.removeEventListener", "new Headers(init.headers)", "headers.has('Accept')"} {
		if !strings.Contains(source, expected) {
			t.Fatalf("generated client does not contain %q", expected)
		}
	}
	if strings.Contains(source, "/api/v1/pages/${locale}/${slug}") || strings.Contains(source, "/api/v1/news/${locale}") {
		t.Fatal("generated client contains obsolete endpoint patterns")
	}
}

func TestGenerateRejectsExistingDirectory(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "existing-proj"

	// Create first time
	_, err := Generate(Options{
		Root:   tempDir,
		Name:   projectName,
		Preset: "news",
	})
	if err != nil {
		t.Fatalf("first generation failed: %v", err)
	}

	// Second time must fail with ErrProjectExists
	_, err = Generate(Options{
		Root:   tempDir,
		Name:   projectName,
		Preset: "news",
	})
	if !errors.Is(err, ErrProjectExists) {
		t.Errorf("expected ErrProjectExists, got %v", err)
	}
}

func TestGenerateRejectsPathTraversal(t *testing.T) {
	tempDir := t.TempDir()

	traversalNames := []string{
		"../escaped",
		"..",
		"foo/bar",
		"foo\\bar",
		"/root-dir",
	}

	for _, name := range traversalNames {
		_, err := Generate(Options{
			Root:   tempDir,
			Name:   name,
			Preset: "news",
		})
		if err == nil {
			t.Errorf("expected error for traversal name %q, but got nil", name)
		}
	}
}

func TestGenerateRejectsInvalidPreset(t *testing.T) {
	tempDir := t.TempDir()

	_, err := Generate(Options{
		Root:   tempDir,
		Name:   "valid-name",
		Preset: "invalid-preset-xyz",
	})
	if !errors.Is(err, ErrUnsupportedPreset) {
		t.Errorf("expected ErrUnsupportedPreset, got %v", err)
	}
}

func TestGeneratePreservesCustomFiles(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "custom-test"

	projectPath, err := Generate(Options{
		Root:   tempDir,
		Name:   projectName,
		Preset: "news",
	})
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	customFile := filepath.Join(projectPath, "custom", "my_hook.go")
	customContent := "// custom user code\npackage custom\n"
	if err := os.WriteFile(customFile, []byte(customContent), 0o644); err != nil {
		t.Fatalf("write custom file failed: %v", err)
	}

	data, err := os.ReadFile(customFile)
	if err != nil || string(data) != customContent {
		t.Errorf("custom file not preserved correctly")
	}
}

func TestValidateFrontendManifestRejectsOversizedFileAndRouteFields(t *testing.T) {
	t.Run("oversized", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "frontend.json")
		if err := os.WriteFile(path, []byte(strings.Repeat("x", maxFrontendManifestBytes+1)), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ValidateFrontendManifest(path); err == nil {
			t.Fatal("oversized frontend manifest accepted")
		}
	})
	t.Run("long-route", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "frontend.json")
		body := `{"routes":[{"name":"` + strings.Repeat("a", 129) + `","path":"/safe","test_id":"safe"}]}`
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ValidateFrontendManifest(path); err == nil {
			t.Fatal("long route name accepted")
		}
	})
}
