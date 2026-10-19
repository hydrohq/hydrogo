import { type DocSection, page, p, h2, h3, code, tabs, note, tip, warn, table, ul, ol } from './types';

export const gettingStarted: DocSection = {
  id: 'getting-started',
  title: 'Getting started',
  pages: [
    page(
      'install',
      'Install',
      "Get the hydro command on Windows, Linux, macOS, in Docker, or build it from source.",
      p("Every method gives you one executable named `hydro`. Releases are published on GitHub for Linux, macOS and Windows (amd64 and arm64) together with a `SHA256SUMS` file; the install scripts verify the checksum for you."),
      h2('Choose a method'),
      table(
        ['Platform', 'How'],
        ['Windows (installer)', 'Download and run `HydroGo-Setup-<version>-windows-amd64.exe` (or `arm64`). It installs `hydro` and adds it to PATH. Open a new terminal afterwards.'],
        ['Windows (PowerShell)', '`irm .../install.ps1 | iex`, see below.'],
        ['Linux and macOS', '`curl -fsSL .../install.sh | sh`, see below.'],
        ['Docker', '`ghcr.io/ozyildirims/hydrogo` for linux/amd64 and arm64.'],
        ['From source', '`go build -o bin/hydro ./cmd/hydro` with Go 1.27.2 or newer.'],
      ),
      h2('Windows'),
      p("Download the Setup executable from the [latest release](https://github.com/ozyildirims/hydrogo/releases/latest) and run it. Leave **Add hydro to PATH** ticked. No administrator rights are needed: it installs per user, and you can choose to install for all users to use Program Files instead."),
      p("Without the installer, use PowerShell:"),
      code('powershell', "irm https://github.com/ozyildirims/hydrogo/releases/latest/download/install.ps1 | iex"),
      h2('Linux and macOS'),
      code('bash', "curl -fsSL https://github.com/ozyildirims/hydrogo/releases/latest/download/install.sh | sh"),
      p("Pin a specific version with the `HYDRO_VERSION` environment variable:"),
      code('bash', "HYDRO_VERSION=v0.2.0 sh install.sh"),
      h2('Docker'),
      code('bash', "docker run -p 8080:8080 -v hydro-data:/app/data ghcr.io/ozyildirims/hydrogo:latest"),
      p("Mount your own `config.yaml` at `/app/config.yaml` and keep the data directory on a volume. A complete Compose file with Caddy is described in [Docker deployment](doc:deployment/docker)."),
      h2('From source'),
      p("You need Go 1.27.2 or newer (the toolchain is pinned in `go.mod`). The admin panel is committed pre-built, so Node.js is not required to build the binary."),
      code('bash', "git clone https://github.com/ozyildirims/hydrogo.git\ncd hydrogo\ngo build -o bin/hydro ./cmd/hydro"),
      h2('Verify the install'),
      code('bash', "hydro version\nhydro version --json\nhydro help"),
      tip("`hydro help` prints every command. The full list is also on the [CLI reference](doc:reference/cli) page."),
    ),

    page(
      'quick-start',
      'Quick start',
      "From nothing to a running admin panel and API in about a minute.",
      h2('1. Create a project'),
      code('bash', "hydro new mysite --preset blog --admin-email you@example.com\ncd mysite"),
      p("`hydro new` creates the folder and writes a `config.yaml` and the preset's `content-types/`. It then creates the SQLite database and the first admin user. If you do not pass `--admin-password-stdin` (or set `HYDRO_ADMIN_PASSWORD`), a strong password is generated and **printed once**, so copy it."),
      note("Without `--admin-email`, `hydro new` only writes files and prints the next commands. Use that when you want to create the admin later with `hydro create-admin`."),
      p("Provide the password from a pipe instead of generating one:"),
      code('bash', "printf '%s' \"$ADMIN_PASSWORD\" | hydro new mysite --preset blog \\\n  --admin-email you@example.com --admin-password-stdin"),
      h2('2. Start the server'),
      code('bash', "hydro serve --config config.yaml"),
      table(
        ['URL', 'What'],
        ['`http://127.0.0.1:8080/admin`', 'Admin panel. Sign in with the e-mail and password from step 1.'],
        ['`http://127.0.0.1:8080/api/v1`', 'REST API.'],
        ['`http://127.0.0.1:8080/healthz`', 'Liveness probe.'],
      ),
      h2('3. Create content and read it back'),
      p("In the admin, create and publish a news article (or a page), then read it through the API:"),
      code('bash', "curl 'http://127.0.0.1:8080/api/v1/news?locale=en&limit=5'"),
      code('json', `{
  "data": [
    {
      "id": "…",
      "locale": "en",
      "slug": "hello-world",
      "title": "Hello world",
      "status": "published"
    }
  ],
  "metadata": { "limit": 5, "has_more": false }
}`),
      h2('4. Add your own content type'),
      code('bash', "hydro content-type new recipes\n# edit content-types/recipes.yaml, then:\nhydro content-type validate\nhydro serve --config config.yaml"),
      p("The admin now has screens at `/admin/content/recipes` and, with `public: true`, the API serves `/api/v1/content/recipes`. See [Content types](doc:content-types/overview)."),
      h2('5. Build your site'),
      p("Use the API from any frontend. Two starting points are the typed [`@hydrogo/client`](doc:frontends/client) and the server-rendered [SvelteKit starter](doc:frontends/sveltekit-starter)."),
      h2('Use it in the current directory'),
      p("`hydro init` does the same as `hydro new` but scaffolds into the current directory and accepts the same options."),
      code('bash', "mkdir mysite && cd mysite\nhydro init --preset corporate --admin-email you@example.com"),
      warn("Run `hydro new` and `hydro init` on a fresh folder. They write `config.yaml` and `content-types/` and are meant for a new project, not for overwriting one that is already in use."),
    ),

    page(
      'presets',
      'Presets',
      "The five starting points for hydro new, which modules they enable and which content types they copy into your project.",
      p("A preset is a starting point: a list of enabled modules and a set of stock content types copied into your project's `content-types/` folder. Everything is plain files afterwards, so you can delete types you do not need and add your own."),
      code('bash', "hydro new --list-presets"),
      table(
        ['Preset', 'For', 'Content types copied'],
        ['`minimal`', 'Pages and media only', '`site_settings`, `site_navigation`'],
        ['`blog`', 'News or blog posts, pages, comments, galleries', '+ `comments`, `galleries`, `faq`'],
        ['`corporate`', 'Company site', '+ `team`, `testimonials`, `faq`'],
        ['`news`', 'News portal', '+ `homepage_sections`, `masthead`, `columnists`, `official_announcements`, `ads`, `polls`, `videos`, `galleries`, `comments`'],
        ['`docs`', 'Documentation site', '+ `faq`'],
      ),
      h2('Enabled modules per preset'),
      table(
        ['Preset', 'Modules'],
        ['`minimal`', 'core, auth, pages, media, site, collections'],
        ['`blog`', 'core, auth, members, pages, media, site, news, collections, analytics'],
        ['`corporate`', 'core, auth, pages, media, site, collections'],
        ['`news`', 'core, auth, members, pages, media, site, news, collections, analytics'],
        ['`docs`', 'core, auth, pages, media, site, collections'],
      ),
      h2('Changing your mind later'),
      ul(
        "To add a stock type, copy its YAML from the repository's `content-types/` folder into your project and restart.",
        "To turn a module on or off, edit the `modules` list in `config.yaml` and run `hydro migrate`. An inactive module adds no routes, migrations or admin navigation.",
        "Delete any content type file you do not need. Run `hydro content-type validate` afterwards.",
      ),
      note("Stock content types are examples. Fields, labels and options are meant to be edited to fit your project."),
    ),

    page(
      'project-layout',
      'Project layout',
      "What hydro new writes to disk, what is safe to commit, and where each kind of change belongs.",
      h2('A scaffolded project'),
      code('text', `mysite/
├── config.yaml          # runtime configuration (see Configuration)
├── content-types/       # one YAML file per content type
│   ├── site_settings.yaml
│   └── faq.yaml
├── data/                # created at first start: SQLite database and media
│   ├── hydro.db
│   └── media/
├── .gitignore
└── README.md`),
      table(
        ['Path', 'Commit it?', 'Notes'],
        ['`config.yaml`', 'Yes, without secrets', 'Put DSNs, keys and passwords in `HYDRO_*` environment variables or a protected secret file.'],
        ['`content-types/`', 'Yes', 'Your data model. Review changes like code.'],
        ['`data/`', 'No', 'Database and uploaded media. Back it up instead.'],
      ),
      h2('Layout of the HydroGo repository'),
      p("Useful if you build from source or contribute."),
      table(
        ['Path', 'What lives there'],
        ['`cmd/hydro`', 'The CLI and HTTP server entrypoint.'],
        ['`admin/`', 'Svelte 5 + TypeScript + Tailwind admin panel (embedded into the binary).'],
        ['`internal/`', 'Core packages: auth, database, media, collections, search, security, scaffolding.'],
        ['`modules/`', 'Content modules: `pages`, `news`, `members`, `site`, `seo`.'],
        ['`content-types/`', 'Stock YAML content types.'],
        ['`migrations/`', 'SQL migrations for SQLite, PostgreSQL and MySQL.'],
        ['`packages/client`', '`@hydrogo/client`, the typed API client.'],
        ['`examples/`', 'Frontend starters, including this site.'],
        ['`deploy/`, `scripts/`', 'Docker, Caddy, nginx, hosting examples, install scripts, Windows installer.'],
        ['`docs/`', 'Guides, API baselines, JSON Schema and decision records.'],
      ),
      h2('Where a change goes'),
      table(
        ['You are adding…', 'Put it in'],
        ['A simple content type (FAQ, team, events)', 'One YAML file in `content-types/`. No Go, no Svelte.'],
        ['Content module logic', '`modules/<name>/` with `model.go`, a service, `repository/sqlstore/` and `transport/http.go`.'],
        ['A shared capability', '`internal/<name>/`.'],
        ['An admin screen', '`admin/src/routes/<module>/`.'],
        ['A database change', 'A new directory in `migrations/` with SQL for all three dialects.'],
        ['A CLI command', 'Its own file in `cmd/hydro/<name>_cmd.go`.'],
      ),
      tip("Use `hydro dev` (alias of `hydro run`) or the repository's `go run ./tools/devserver` while working on HydroGo itself. It rebuilds on save and seeds disposable demo data."),
    ),
  ],
};
