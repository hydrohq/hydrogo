package projectgen

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	themev1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/config"
)

var (
	ErrProjectExists      = errors.New("project directory already exists")
	ErrInvalidProjectName = errors.New("project name must be 1-64 alphanumeric characters with underscores or hyphens and contain no path separators")
	ErrPathTraversal      = errors.New("path traversal rejected in project name or root")
	ErrUnsupportedPreset  = errors.New("unsupported project preset")
	projectNamePattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)
)

type Options struct {
	Root         string `json:"root"`
	Name         string `json:"name"`
	Preset       string `json:"preset"`
	Profile      string `json:"profile"`
	Database     string `json:"database"`
	Presentation string `json:"presentation"`
	Edition      string `json:"edition"`
	Demo         bool   `json:"demo"`
	E2E          bool   `json:"e2e"`
	Git          bool   `json:"git"`
}

type Lock struct {
	LockVersion int           `json:"lock_version"`
	ProjectID   string        `json:"project_id"`
	ProjectName string        `json:"project_name"`
	Edition     string        `json:"edition"`
	Preset      string        `json:"preset"`
	Core        CoreLock      `json:"core"`
	Generator   GeneratorLock `json:"generator"`
	Features    []string      `json:"features"`
	Licensing   LicenseLock   `json:"licensing"`
}

type CoreLock struct {
	Version      string `json:"version"`
	Distribution string `json:"distribution"`
	Module       string `json:"module"`
}

type GeneratorLock struct {
	Version     string `json:"version"`
	GeneratedAt string `json:"generated_at"`
}

type LicenseLock struct {
	Required           bool   `json:"required"`
	LicenseFile        string `json:"license_file"`
	GracePeriodAllowed bool   `json:"grace_period_allowed"`
}

func Generate(opts Options) (string, error) {
	if strings.TrimSpace(opts.Root) == "" {
		opts.Root = "."
	}
	// Path traversal protection
	if strings.Contains(opts.Name, "/") || strings.Contains(opts.Name, "\\") || strings.Contains(opts.Name, "..") {
		return "", ErrPathTraversal
	}
	if !projectNamePattern.MatchString(opts.Name) {
		return "", ErrInvalidProjectName
	}
	if !isPreset(opts.Preset) {
		return "", fmt.Errorf("%w: %q (supported: news, commerce, corporate, portal, headless)", ErrUnsupportedPreset, opts.Preset)
	}
	if opts.Profile == "" {
		opts.Profile = "compact"
	}
	if opts.Database == "" {
		opts.Database = "sqlite"
	}
	if opts.Presentation == "" {
		if opts.Preset == "headless" {
			opts.Presentation = "headless"
		} else {
			opts.Presentation = "integrated"
		}
	}
	if opts.Edition == "" {
		opts.Edition = "starter"
	}
	if opts.Database != "sqlite" && opts.Database != "postgres" {
		return "", fmt.Errorf("unsupported database %q (supported: sqlite, postgres)", opts.Database)
	}
	if opts.Presentation != "integrated" && opts.Presentation != "headless" {
		return "", fmt.Errorf("unsupported presentation %q (supported: integrated, headless)", opts.Presentation)
	}

	rootAbs, err := filepath.Abs(opts.Root)
	if err != nil {
		return "", err
	}
	projectRoot := filepath.Join(rootAbs, opts.Name)
	cleanProjectRoot := filepath.Clean(projectRoot)

	// Ensure project directory stays inside root
	rel, err := filepath.Rel(rootAbs, cleanProjectRoot)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
		return "", ErrPathTraversal
	}

	if _, err := os.Stat(cleanProjectRoot); err == nil {
		return "", fmt.Errorf("%w: %s", ErrProjectExists, cleanProjectRoot)
	} else if !os.IsNotExist(err) {
		return "", err
	}

	features := presetFeatures(opts.Preset)
	if opts.Presentation == "headless" && !contains(features, "headless") {
		features = append(features, "headless")
	}
	sort.Strings(features)

	projectID := "proj-" + strings.ToLower(opts.Name)
	lock := Lock{
		LockVersion: 1,
		ProjectID:   projectID,
		ProjectName: opts.Name,
		Edition:     opts.Edition,
		Preset:      opts.Preset,
		Core: CoreLock{
			Module:       "github.com/ardaozyildirim/hydrogo",
			Version:      "v0.1.0",
			Distribution: "licensed_binary",
		},
		Generator: GeneratorLock{
			Version:     "0.1.0",
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		},
		Features: features,
		Licensing: LicenseLock{
			Required:           true,
			LicenseFile:        "hydro.lic",
			GracePeriodAllowed: true,
		},
	}

	themeName := presetTheme(opts.Preset)

	files := map[string]string{
		"config.yaml":          configTemplate(opts, lock, themeName),
		"hydrogo.lock":         mustJSON(lock),
		"README.md":            readmeTemplate(opts, lock),
		"PROJECT_STATUS.md":    projectStatusTemplate(opts),
		"TASKS.md":             tasksTemplate(opts),
		"LICENSE":              licenseDraftTemplate(opts),
		"NOTICE":               noticeTemplate(),
		"docs/ARCHITECTURE.md": architectureDocTemplate(opts, lock),
		"docs/OPERATIONS.md":   operationsDocTemplate(opts),
		"docs/E2E_GUIDE.md":    e2eDocTemplate(opts),
		"e2e/manifest.json":    e2eManifest(opts),
		"custom/README.md":     "# Custom Business Logic\n\nPlace client-specific handlers, extension hooks, and integrations here.\nHydroGo Core remains external and is not modified in this project.\n",
		"migrations/README.md": "# Project Migrations\n\nPlace project-specific SQL migrations here. HydroGo Core migrations are applied automatically by the licensed runtime.\n",
	}

	// Preset theme files
	themeFiles := presetThemeFiles(themeName, opts)
	for path, content := range themeFiles {
		files[path] = content
	}

	// Demo seed files
	if opts.Demo {
		seedFiles := presetSeedFiles(opts.Preset, opts.Name)
		for path, content := range seedFiles {
			files[path] = content
		}
	}

	// Headless client if requested
	if opts.Presentation == "headless" || opts.Preset == "headless" {
		headlessFiles := headlessClientFiles(opts.Name)
		for path, content := range headlessFiles {
			files[path] = content
		}
	}

	if err := os.MkdirAll(cleanProjectRoot, 0o755); err != nil {
		return "", err
	}

	for name, body := range files {
		path := filepath.Join(cleanProjectRoot, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return "", err
		}
	}

	// Validate generated theme with runtime loader
	if opts.Presentation != "headless" {
		themeDir := filepath.Join(cleanProjectRoot, "themes")
		if _, err := themev1.LoadTheme(themeDir, themeName); err != nil {
			_ = os.RemoveAll(cleanProjectRoot)
			return "", fmt.Errorf("generated theme validation failed: %w", err)
		}
	}

	// Validate generated config.yaml with config.Load
	cfgPath := filepath.Join(cleanProjectRoot, "config.yaml")
	if _, err := config.Load(cfgPath, nil, config.Overrides{}); err != nil {
		_ = os.RemoveAll(cleanProjectRoot)
		return "", fmt.Errorf("generated config.yaml validation failed: %w", err)
	}

	if opts.Git {
		cmd := exec.Command("git", "init")
		cmd.Dir = cleanProjectRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			// Don't fail the whole generation if git is missing, but return diagnostic
			_ = out
		}
	}

	return cleanProjectRoot, nil
}

func isPreset(value string) bool {
	switch strings.ToLower(value) {
	case "news", "commerce", "corporate", "portal", "headless":
		return true
	default:
		return false
	}
}

func presetTheme(preset string) string {
	switch strings.ToLower(preset) {
	case "news":
		return "newsroom"
	case "commerce":
		return "commerce"
	case "portal":
		return "portal"
	case "headless":
		return "api"
	default:
		return "corporate"
	}
}

func presetFeatures(preset string) []string {
	switch strings.ToLower(preset) {
	case "news":
		return []string{"pages", "news", "media", "members", "site"}
	case "commerce":
		return []string{"pages", "commerce", "media", "members", "site"}
	case "portal":
		return []string{"pages", "news", "media", "members", "site"}
	case "headless":
		return []string{"pages", "news", "media", "members"}
	default:
		return []string{"pages", "media", "site"}
	}
}

func presetThemeFiles(themeName string, opts Options) map[string]string {
	manifest := themev1.ThemeManifest{
		Name:    themeName,
		Version: "1.0.0",
		Slots:   []string{"header", "content", "footer"},
		Assets:  []string{"static/theme.css"},
	}
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")

	baseLayout := `<!doctype html>
<html lang="{{.Locale}}">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} - HydroGo</title>
  <link rel="stylesheet" href="/theme/static/theme.css">
</head>
<body>
  <header class="site-header">
    <div class="header-inner">
      <a href="/" class="brand-logo">{{.Title}}</a>
      {{slot "header"}}
    </div>
  </header>
  <main class="site-main">
    {{slot "content"}}
  </main>
  <footer class="site-footer">
    <div class="footer-inner">
      <p>&copy; 2026 ` + opts.Name + `. Powered by HydroGo.</p>
      {{slot "footer"}}
    </div>
  </footer>
</body>
</html>
`

	cssContent := `:root {
  --primary: #155dfc;
  --primary-hover: #124ecc;
  --bg: #f8fafc;
  --text: #0f172a;
  --text-muted: #64748b;
  --card-bg: #ffffff;
  --border: #e2e8f0;
  --font-sans: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #090d16;
    --text: #f1f5f9;
    --text-muted: #94a3b8;
    --card-bg: #131a29;
    --border: #1e293b;
  }
}
* { box-sizing: border-box; }
body {
  margin: 0;
  font-family: var(--font-sans);
  background: var(--bg);
  color: var(--text);
  line-height: 1.6;
}
.site-header {
  border-bottom: 1px solid var(--border);
  background: var(--card-bg);
  position: sticky;
  top: 0;
  z-index: 50;
}
.header-inner, .footer-inner {
  max-width: 1200px;
  margin: 0 auto;
  padding: 16px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.brand-logo {
  font-weight: 700;
  font-size: 1.25rem;
  color: var(--primary);
  text-decoration: none;
}
.site-main {
  max-width: 1200px;
  margin: 0 auto;
  padding: 32px 20px;
}
.site-footer {
  border-top: 1px solid var(--border);
  background: var(--card-bg);
  margin-top: 64px;
}
`

	return map[string]string{
		filepath.ToSlash(filepath.Join("themes", themeName, "theme.json")):           string(manifestJSON) + "\n",
		filepath.ToSlash(filepath.Join("themes", themeName, "layouts", "base.html")): baseLayout,
		filepath.ToSlash(filepath.Join("themes", themeName, "static", "theme.css")):  cssContent,
	}
}

func presetSeedFiles(preset, projectName string) map[string]string {
	now := time.Now().UTC().Format(time.RFC3339)
	readme := "# Seed Data\n\nThis directory contains baseline demonstration and starter data.\nTo import, run `hydro seed-demo --config config.yaml` or use the Admin Studio.\n"

	data := map[string]any{
		"preset":       preset,
		"project_name": projectName,
		"seeded_at":    now,
	}
	dataBytes, _ := json.MarshalIndent(data, "", "  ")

	return map[string]string{
		"seed/README.md": readme,
		"seed/data.json": string(dataBytes) + "\n",
	}
}

func headlessClientFiles(projectName string) map[string]string {
	pkgJSON := `{
  "name": "` + strings.ToLower(projectName) + `-client",
  "version": "1.0.0",
  "private": true,
  "description": "Typed HydroGo TypeScript API client",
  "main": "hydrogo.ts",
  "scripts": {
    "test": "node --test"
  }
}
`
	tsClient := `/**
 * HydroGo Typed API Client
 * Compatible with Node.js, Bun, Deno, Next.js, and browser runtimes.
 */
export interface HydroConfig {
  baseUrl: string;
  apiKey?: string;
}

export interface Page {
  id: string;
  slug: string;
  title: string;
  body: string;
  locale: string;
  updated_at: string;
}

export interface NewsItem {
  id: string;
  slug: string;
  title: string;
  excerpt: string;
  category: string;
  body: string;
  published_at: string;
}

export class HydroClient {
  constructor(private config: HydroConfig) {}

  private async request<T>(path: string): Promise<T> {
    const url = new URL(path, this.config.baseUrl);
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    if (this.config.apiKey) {
      headers['Authorization'] = ` + "`Bearer ${this.config.apiKey}`" + `;
    }
    const res = await fetch(url.toString(), { headers });
    if (!res.ok) {
      throw new Error(` + "`HydroGo API error: ${res.status} ${res.statusText}`" + `);
    }
    return res.json();
  }

  async getPage(locale: string, slug: string): Promise<Page> {
    return this.request<Page>(` + "`/api/v1/pages/${locale}/${slug}`" + `);
  }

  async listNews(locale: string = 'tr'): Promise<NewsItem[]> {
    return this.request<NewsItem[]>(` + "`/api/v1/news/${locale}`" + `);
  }
}
`
	return map[string]string{
		"client/package.json": pkgJSON,
		"client/hydrogo.ts":   tsClient,
		"client/README.md":    "# Headless Client\n\nImport `HydroClient` from `./hydrogo.ts` into your frontend application.\n",
	}
}

func mustJSON(value any) string {
	data, _ := json.MarshalIndent(value, "", "  ")
	return string(data) + "\n"
}

func configTemplate(opts Options, lock Lock, themeName string) string {
	dsn := fmt.Sprintf("data/%s.sqlite", opts.Name)
	if opts.Database == "postgres" {
		dsn = fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:5432/%s?sslmode=disable", opts.Name)
	}

	return fmt.Sprintf(`profile: %s
modules:
%shttp:
  address: "127.0.0.1:8080"
database:
  dialect: %s
  dsn: "%s"
media:
  dir: "data/media"
theme_dir: "themes"
theme: "%s"
i18n:
  enabled: false
  default_locale: "tr"
  locales: ["tr"]
security:
  secure_cookies: false
license:
  required: true
  file: "hydro.lic"
  project_id: "%s"
  edition: "%s"
  features: [%s]
`, opts.Profile, moduleLines(lock.Features), opts.Database, dsn, themeName, lock.ProjectID, lock.Edition, quoteList(lock.Features))
}

func moduleLines(features []string) string {
	var b strings.Builder
	for _, f := range features {
		if f != "headless" {
			b.WriteString("  - " + f + "\n")
		}
	}
	return b.String()
}

func quoteList(values []string) string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(out, ", ")
}

func readmeTemplate(opts Options, lock Lock) string {
	return fmt.Sprintf(`# %s

HydroGo Client Edition project scaffolded with the **%s** preset.

> HydroGo Core is distributed separately as a licensed runtime binary or private Go module.
> This repository contains only client-specific themes, configurations, migrations, and extension hooks.

## Quick Start

1. **Verify or obtain your license:**
   Place your vendor-signed `+"`hydro.lic`"+` in the project root. (Project ID: `+"`%s`"+`)

2. **Run database migrations:**
   `+"```powershell"+`
   hydro migrate --config config.yaml
   `+"```"+`

3. **Create the initial administrator:**
   `+"```powershell"+`
   hydro create-admin --config config.yaml --email admin@example.test --password-stdin
   `+"```"+`

4. **Start the server:**
   `+"```powershell"+`
   hydro serve --config config.yaml
   `+"```"+`

## Connecting to Git

`+"```powershell"+`
git init
git add .
git commit -m "Initial commit for %s"
git remote add origin <your-git-repository-url>
git push -u origin main
`+"```"+`
`, opts.Name, opts.Preset, lock.ProjectID, opts.Name)
}

func projectStatusTemplate(opts Options) string {
	return fmt.Sprintf(`# %s - Project Status

- **Preset:** %s
- **Profile:** %s
- **Database:** %s
- **Presentation:** %s
- **Status:** Scaffolded (Pilot Candidate)

## Readiness Checklist
- [ ] Database DSN and credentials configured in `+"`config.yaml`"+`
- [ ] `+"`hydro.lic`"+` license placed and verified with `+"`hydro license verify --license hydro.lic`"+`
- [ ] Admin user created
- [ ] Theme customized in `+"`themes/`"+`
- [ ] E2E tests verified with `+"`npm test`"+`
`, opts.Name, opts.Preset, opts.Profile, opts.Database, opts.Presentation)
}

func tasksTemplate(opts Options) string {
	return fmt.Sprintf(`# %s Tasks

- [ ] Complete environment setup and database initialization.
- [ ] Obtain vendor license `+"`hydro.lic`"+` matching project ID.
- [ ] Customize the `+"`%s`"+` theme in `+"`themes/`"+`.
- [ ] Add project content, pages, and media assets.
- [ ] Execute Playwright E2E test suite.
`, opts.Name, presetTheme(opts.Preset))
}

func licenseDraftTemplate(opts Options) string {
	return `================================================================================
CLIENT EDITION SOFTWARE LICENSE
[LAWYER REVIEW REQUIRED / HUKUKİ İNCELEME GEREKLİDİR]
================================================================================

This repository contains client-specific customization code, theme templates, and
configuration files for the ` + opts.Name + ` project.

HydroGo Core runtime components are proprietary and governed by the commercial
license issued to the customer. Execution and compilation of this project with
HydroGo Core requires a valid, active commercial license certificate (hydro.lic).
`
}

func noticeTemplate() string {
	return `HydroGo Client Edition
Copyright (c) HydroGo Authors & Contributors. All rights reserved.

This project relies upon the HydroGo runtime platform and associated open-source
dependencies. See the platform NOTICE file for third-party copyright attributions.
`
}

func architectureDocTemplate(opts Options, lock Lock) string {
	return fmt.Sprintf(`# Architecture Overview - %s

## Core Isolation
HydroGo Core is decoupled from this project. The client project contains:
- **Themes:** `+"`themes/%s`"+`
- **Configuration:** `+"`config.yaml`"+`
- **Lockfile:** `+"`hydrogo.lock`"+` (Core Version: %s, Preset: %s)
- **Custom Hooks:** `+"`custom/`"+`
- **Migrations:** `+"`migrations/`"+`

## Feature Matrix
Enabled modules: %s
`, opts.Name, presetTheme(opts.Preset), lock.Core.Version, opts.Preset, strings.Join(lock.Features, ", "))
}

func operationsDocTemplate(opts Options) string {
	return `# Operations Runbook

## Health Check
- URL: ` + "`http://127.0.0.1:8080/healthz`" + `
- Expected HTTP status: 200 OK

## License Status
Check your license status anytime:
` + "```powershell\nhydro license status --license hydro.lic\n```" + `
`
}

func e2eDocTemplate(opts Options) string {
	return `# E2E Testing Guide

This project includes metadata-driven Playwright tests generated from ` + "`e2e/manifest.json`" + `.

To run tests:
` + "```powershell\nnpm test\n```" + `
`
}

func e2eManifest(opts Options) string {
	routes := []map[string]any{
		{"name": "home", "path": "/", "public": true, "test_id": "public-home"},
	}
	if opts.Preset == "news" {
		routes = append(routes,
			map[string]any{"name": "news_room", "path": "/public/news/tr/html", "public": true, "test_id": "news-feed"},
			map[string]any{"name": "member_login", "path": "/member/login", "public": true, "test_id": "member-login"},
		)
	} else if opts.Preset == "commerce" {
		routes = append(routes,
			map[string]any{"name": "shop", "path": "/shop", "public": true, "test_id": "product-catalog"},
			map[string]any{"name": "cart", "path": "/cart", "public": true, "test_id": "shopping-cart"},
		)
	}

	data := map[string]any{
		"version":  1,
		"preset":   opts.Preset,
		"base_url": "http://127.0.0.1:8080",
		"frontend": map[string]any{
			"routes": routes,
		},
		"media": map[string]any{
			"route":          "/admin/media",
			"upload_enabled": true,
		},
	}
	dataBytes, _ := json.MarshalIndent(data, "", "  ")
	return string(dataBytes) + "\n"
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}
