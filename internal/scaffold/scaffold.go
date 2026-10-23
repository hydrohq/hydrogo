// Package scaffold creates the files of a new HydroGo project: a config, the
// content types of a chosen preset, a .gitignore and a short README.
package scaffold

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	contenttypedata "github.com/ardaozyildirim/hydrogo/content-types"
	crmstarter "github.com/ardaozyildirim/hydrogo/examples/crm-starter"
	sveltekitstarter "github.com/ardaozyildirim/hydrogo/examples/sveltekit-starter"
)

// Preset is a starting point for a project: the modules it enables and the
// stock content types it copies into the project's content-types directory.
type Preset struct {
	Name         string
	Description  string
	Modules      []string
	ContentTypes []string
	// Config is extra YAML appended to the generated config.yaml.
	Config string
}

var presets = []Preset{
	{
		Name:         "minimal",
		Description:  "pages, media and site settings only",
		Modules:      []string{"core", "auth", "pages", "media", "site", "collections"},
		ContentTypes: []string{"site_settings", "site_navigation"},
	},
	{
		Name:         "blog",
		Description:  "news/blog posts, pages, media, comments and galleries",
		Modules:      []string{"core", "auth", "members", "pages", "media", "site", "news", "collections", "analytics"},
		ContentTypes: []string{"site_settings", "site_navigation", "comments", "galleries", "faq"},
	},
	{
		Name:         "corporate",
		Description:  "company site: pages, team, testimonials, FAQ",
		Modules:      []string{"core", "auth", "pages", "media", "site", "collections"},
		ContentTypes: []string{"site_settings", "site_navigation", "team", "testimonials", "faq"},
	},
	{
		Name:         "news",
		Description:  "news portal: homepage sections, columnists, polls, videos, ads",
		Modules:      []string{"core", "auth", "members", "pages", "media", "site", "news", "collections", "analytics"},
		ContentTypes: []string{"site_settings", "site_navigation", "homepage_sections", "masthead", "columnists", "official_announcements", "ads", "polls", "videos", "galleries", "comments"},
	},
	{
		Name:         "crm",
		Description:  "customer relationship management: companies, contacts, deals and activities",
		Modules:      []string{"core", "auth", "media", "collections"},
		ContentTypes: []string{"crm/companies", "crm/contacts", "crm/deals", "crm/activities"},
		Config:       "audit:\n  # Who changed what: every admin write is logged with its redacted body (see the Audit screen).\n  log_request_bodies: true\n",
	},
	{
		Name:         "erp-lite",
		Description:  "light ERP: products, warehouses, accounts, sales orders, invoices and payments",
		Modules:      []string{"core", "auth", "media", "collections"},
		ContentTypes: []string{"erp/products", "erp/warehouses", "erp/accounts", "erp/sales_orders", "erp/order_lines", "erp/invoices", "erp/invoice_lines", "erp/payments", "erp/period_closings"},
		Config:       "audit:\n  # Tamper evidence: every audit event is chained; check it with hydro audit verify.\n  chain: true\n  log_request_bodies: true\n",
	},
	{
		Name:         "docs",
		Description:  "documentation site: pages, FAQ and navigation",
		Modules:      []string{"core", "auth", "pages", "media", "site", "collections"},
		ContentTypes: []string{"site_settings", "site_navigation", "faq"},
	},
}

// Presets returns the available presets sorted by name.
func Presets() []Preset {
	out := append([]Preset(nil), presets...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Lookup finds a preset by name.
func Lookup(name string) (Preset, error) {
	for _, preset := range presets {
		if preset.Name == name {
			return preset, nil
		}
	}
	names := make([]string, 0, len(presets))
	for _, preset := range Presets() {
		names = append(names, preset.Name)
	}
	return Preset{}, fmt.Errorf("unknown preset %q (available: %s)", name, strings.Join(names, ", "))
}

// Options configure Create.
type Options struct {
	// Dir is the project directory. It is created when missing.
	Dir string
	// Name is the project name shown in the generated README.
	Name string
	// Preset is the preset name; empty means "minimal".
	Preset string
	// Starter names a frontend starter to copy into <Dir>/frontend. Empty
	// means none; see Starters.
	Starter string
}

// Starters lists the frontend starters Create can copy into a project.
func Starters() []string { return []string{"sveltekit", "crm"} }

// copyStarter writes the embedded frontend starter into dir.
func copyStarter(name, dir string) error {
	var source fs.FS
	switch name {
	case "sveltekit":
		source = sveltekitstarter.FS
	case "crm":
		source = crmstarter.FS
	default:
		return fmt.Errorf("unknown starter %q (available: %s)", name, strings.Join(Starters(), ", "))
	}
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("%s already exists; refusing to overwrite", dir)
	}
	return fs.WalkDir(source, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755) // #nosec G301 -- project directory; the service account may differ from the author
		}
		data, readErr := fs.ReadFile(source, path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(target, data, 0o644) // #nosec G306 -- project source file with no secrets (those live in HYDRO_* variables); the service account may differ from the author
	})
}

// Create writes a new project into opts.Dir. It refuses to overwrite an
// existing config.yaml or content-types directory, so running it twice in the
// same directory never destroys work.
func Create(opts Options) (Preset, error) {
	if opts.Preset == "" {
		opts.Preset = "minimal"
	}
	preset, err := Lookup(opts.Preset)
	if err != nil {
		return Preset{}, err
	}
	if strings.TrimSpace(opts.Dir) == "" {
		return Preset{}, errors.New("project directory is required")
	}
	if opts.Name == "" {
		abs, absErr := filepath.Abs(opts.Dir)
		if absErr != nil {
			return Preset{}, absErr
		}
		opts.Name = filepath.Base(abs)
	}
	if opts.Starter != "" && !slices.Contains(Starters(), opts.Starter) {
		return Preset{}, fmt.Errorf("unknown starter %q (available: %s)", opts.Starter, strings.Join(Starters(), ", "))
	}
	for _, existing := range []string{"config.yaml", "content-types", "frontend", "e2e"} {
		if _, statErr := os.Stat(filepath.Join(opts.Dir, existing)); statErr == nil {
			return Preset{}, fmt.Errorf("%s already exists in %s; refusing to overwrite", existing, opts.Dir)
		}
	}
	if err := os.MkdirAll(filepath.Join(opts.Dir, "content-types"), 0o755); err != nil { // #nosec G301 -- project directory; the service account may differ from the author
		return Preset{}, err
	}
	if err := os.MkdirAll(filepath.Join(opts.Dir, "data"), 0o755); err != nil { // #nosec G301 -- project directory; the service account may differ from the author
		return Preset{}, err
	}
	for _, id := range preset.ContentTypes {
		data, readErr := fs.ReadFile(contenttypedata.FS, id+".yaml")
		id = filepath.Base(id)
		if readErr != nil {
			return Preset{}, fmt.Errorf("preset %s: content type %s: %w", preset.Name, id, readErr)
		}
		if err := os.WriteFile(filepath.Join(opts.Dir, "content-types", id+".yaml"), data, 0o644); err != nil { // #nosec G306 -- project source file with no secrets (those live in HYDRO_* variables); the service account may differ from the author
			return Preset{}, err
		}
	}
	files := map[string]string{
		"config.yaml": renderConfig(preset),
		".gitignore":  "/data/\n*.db\n*.db-shm\n*.db-wal\n.env\n",
		"README.md":   renderReadme(opts.Name, preset),
	}
	for name, content := range files {
		path := filepath.Join(opts.Dir, name)
		if _, statErr := os.Stat(path); statErr == nil && name != "config.yaml" {
			continue // keep an existing README / .gitignore
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil { // #nosec G306 -- project source file with no secrets (those live in HYDRO_* variables); the service account may differ from the author
			return Preset{}, err
		}
	}
	if opts.Starter != "" {
		if err := copyStarter(opts.Starter, filepath.Join(opts.Dir, "frontend")); err != nil {
			return Preset{}, err
		}
	}
	if err := writeE2EHarness(opts.Dir); err != nil {
		return Preset{}, err
	}
	return preset, nil
}

// writeE2EHarness sets up a Playwright E2E directory for the project: a
// config, dependencies, and the generated-vs-hand-written split "hydro
// generate tests" and project authors both rely on. It is always written
// (not gated by --starter), since role-permission tests target the admin
// API, which every project has.
func writeE2EHarness(dir string) error {
	e2eDir := filepath.Join(dir, "e2e")
	for _, sub := range []string{filepath.Join(e2eDir, "generated", "content"), filepath.Join(e2eDir, "custom")} {
		if err := os.MkdirAll(sub, 0o755); err != nil { // #nosec G301 -- project directory; the service account may differ from the author
			return err
		}
	}
	files := map[string]string{
		filepath.Join(e2eDir, "package.json"):                     e2ePackageJSON,
		filepath.Join(e2eDir, "playwright.config.ts"):             e2ePlaywrightConfig,
		filepath.Join(e2eDir, "global-setup.ts"):                  e2eGlobalSetup,
		filepath.Join(e2eDir, ".gitignore"):                       "/.auth/\n/artifacts/\n",
		filepath.Join(e2eDir, "README.md"):                        e2eReadme,
		filepath.Join(e2eDir, "generated", "content", ".gitkeep"): "",
		filepath.Join(e2eDir, "custom", ".gitkeep"):               "",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil { // #nosec G306 -- project source file with no secrets (those live in HYDRO_* variables); the service account may differ from the author
			return err
		}
	}
	return nil
}

const e2ePackageJSON = `{
  "name": "e2e",
  "private": true,
  "type": "module",
  "scripts": {
    "test": "playwright test"
  },
  "devDependencies": {
    "@playwright/test": "^1.64.1"
  }
}
`

const e2ePlaywrightConfig = `import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  outputDir: "./artifacts",
  timeout: 30_000,
  // global-setup.ts logs in once per role and saves the session for every
  // test to reuse — HydroGo's own login rate limit (5/minute per IP) is easy
  // to trip if each test logs in on its own, especially with fullyParallel.
  globalSetup: "./global-setup.ts",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? "line" : "list",
  use: {
    baseURL: process.env.HYDRO_E2E_BASE_URL ?? "http://127.0.0.1:8080",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
`

const e2eGlobalSetup = `import { request } from "@playwright/test";
import { mkdirSync, writeFileSync } from "node:fs";

// Logs in once per seeded role (see "hydro seed-test-users") before any
// generated or hand-written test runs, and saves each session to .auth/ for
// _helpers.ts's login() to reuse. A role whose HYDRO_E2E_<ROLE>_EMAIL /
// _PASSWORD aren't set is skipped: a project may only seed the roles it
// actually tests.
const BASE_URL = process.env.HYDRO_E2E_BASE_URL ?? "http://127.0.0.1:8080";
const ROLES = ["admin", "editor", "viewer"] as const;

export default async function globalSetup(): Promise<void> {
  mkdirSync(".auth", { recursive: true });
  for (const role of ROLES) {
    const prefix = ` + "`HYDRO_E2E_${role.toUpperCase()}`" + `;
    const email = process.env[` + "`${prefix}_EMAIL`" + `];
    const password = process.env[` + "`${prefix}_PASSWORD`" + `];
    if (!email || !password) continue;

    const ctx = await request.newContext({ baseURL: BASE_URL });
    const csrfRes = await ctx.get("/csrf-token");
    const { token } = (await csrfRes.json()) as { token: string };
    const loginRes = await ctx.post("/auth/login", {
      headers: { "x-csrf-token": token },
      form: { email, password },
    });
    if (!loginRes.ok()) {
      throw new Error(` + "`global-setup: login as ${role} failed with status ${loginRes.status()}`" + `);
    }
    await ctx.storageState({ path: ` + "`.auth/${role}.json`" + ` });
    // storageState only carries cookies; the CSRF header value is separate.
    writeFileSync(` + "`.auth/${role}.csrf`" + `, token);
    await ctx.dispose();
  }
}
`

const e2eReadme = `# Tests for this project

` + "`hydro generate tests`" + ` regenerates ` + "`generated/content/*.spec.ts`" + ` from
` + "`content-types/*.yaml`" + `: for every content type, one test per admin role
proving it gets exactly the read/write access the type's permissions declare.
Run it again any time a content type changes; the generated files start with
"DO NOT EDIT" and are safe to overwrite.

Hand-written journeys (anything the generator can't know — business flows,
UI screenshots, multi-step scenarios) belong in ` + "`custom/`" + ` and are never
touched by the generator.

Setup, once:

` + "```" + `
hydro serve --config ../config.yaml &
hydro seed-test-users --config ../config.yaml   # prints HYDRO_E2E_*_EMAIL/PASSWORD
npm install
npx playwright install chromium
` + "```" + `

Then set the printed env vars and run:

` + "```" + `
hydro generate tests --config ../config.yaml
npm test
` + "```" + `
`

func renderConfig(preset Preset) string {
	return fmt.Sprintf(`# HydroGo configuration (preset: %s).
# Secrets and deployment values belong in HYDRO_* environment variables;
# see config.example.yaml in the HydroGo repository for every option.
profile: compact
modules: [%s]
http:
  address: "127.0.0.1:8080"
  # Set to the canonical HTTPS origin when you deploy.
  public_origin: ""
database:
  dialect: "sqlite"
  dsn: "data/hydro.db"
media:
  dir: "data/media"
content_types_dir: "content-types"
security:
  # Turn both on when the site is served over HTTPS.
  secure_cookies: false
  enable_hsts: false
%s`, preset.Name, strings.Join(preset.Modules, ", "), preset.Config)
}

func renderReadme(name string, preset Preset) string {
	return fmt.Sprintf(`# %s

A HydroGo project (preset: %s, %s).

    hydro serve --config config.yaml

Then open http://127.0.0.1:8080/admin. The REST API is under /api/v1.

- Content types live in content-types/*.yaml. Add a file, restart, and the admin
  screens and /api/v1/content/<id> endpoint appear.
- Data (SQLite database and uploaded media) lives in data/. Back it up with
  hydro backup.
- Build your site as a separate frontend on the API; see the HydroGo docs.
`, name, preset.Name, preset.Description)
}
