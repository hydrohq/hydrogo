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
				"e2e/manifest.json",
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
	for _, expected := range []string{"/api/v1/pages/by-slug?locale=", "/api/v1/news?locale="} {
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
