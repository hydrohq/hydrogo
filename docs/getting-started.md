# Getting started

HydroGo is a headless CMS: one binary gives you the admin panel at `/admin` and a JSON API under `/api/v1`. You build the public site as a separate frontend.

## 1. Install the `hydro` command

Download the installer for your system from the [latest release](https://github.com/ardaozyildirim/hydrogo/releases/latest):

| System | CPU | File | Install |
|---|---|---|---|
| Windows | x64 / ARM64 | `HydroGo-Setup-<version>-windows-amd64.exe` / `-arm64.exe` | Run it, keep "Add hydro to PATH" ticked, open a new terminal. Installs per user, no administrator rights needed |
| macOS | Apple silicon / Intel | `HydroGo-<version>-macos-arm64.pkg` / `-amd64.pkg` | Open it, or `sudo installer -pkg HydroGo-*.pkg -target /`. The package is unsigned: on first open use right-click > Open |
| Debian, Ubuntu | amd64 / arm64 | `hydrogo_<version>_amd64.deb` / `_arm64.deb` | `sudo apt install ./hydrogo_*.deb` |
| Fedora, RHEL, openSUSE | x86_64 / aarch64 | `hydrogo-<version>-1.x86_64.rpm` / `.aarch64.rpm` | `sudo dnf install ./hydrogo-*.rpm` |
| Any Linux or macOS | amd64 / arm64 | `hydrogo_v<version>_<os>_<arch>.tar.gz` | Unpack and put `hydro` on your `PATH` |

The one-line scripts (`install.sh`, `install.ps1`) pick the right archive for your system automatically.

**Linux / macOS:**

```bash
curl -fsSL https://github.com/ardaozyildirim/hydrogo/releases/latest/download/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://github.com/ardaozyildirim/hydrogo/releases/latest/download/install.ps1 | iex
```

**Docker:** `docker run -p 8080:8080 -v hydro-data:/app/data ghcr.io/ardaozyildirim/hydrogo:latest` (mount your own `config.yaml` at `/app/config.yaml`; see [deployment](deployment.md)).

While the repository is private, the `curl`/`irm` one-liners and `docker pull` cannot download anonymously. Sign in to GitHub and download the installer (`HydroGo-Setup-<version>-windows-<arch>.exe`) or the archive for your platform from the repository's **Releases** page instead; the scripts work unchanged once the repository is public.

Check it: `hydro version`. Every download has a checksum in `SHA256SUMS`; the install scripts verify it.

## 2. Create a project

```bash
hydro new mysite --preset blog --starter sveltekit --admin-email you@example.com
cd mysite
hydro serve --config config.yaml
```

Open http://127.0.0.1:8080/admin. `hydro new` creates the folder, a `config.yaml`, the content types of the preset, the SQLite database, and the admin user. If you do not pass `--admin-password-stdin` (or set `HYDRO_ADMIN_PASSWORD`), a strong password is generated and printed once.

Presets (`hydro new --list-presets`):

| Preset | For | Content types |
|---|---|---|
| `minimal` | pages and media only | site settings, navigation |
| `blog` | posts and pages | + comments, galleries, FAQ |
| `corporate` | company site | + team, testimonials, FAQ |
| `news` | news portal | + homepage sections, masthead, columnists, announcements, ads, polls, videos |
| `docs` | documentation site | + FAQ |
| `crm` | sales (no website: pages and news are off) | companies, contacts, deals, activities; `--starter crm` adds a sales panel |
| `erp-lite` | light ERP back office (no website) | products, warehouses, accounts, sales orders and lines, invoices and lines, payments |
| `commerce` | storefront: catalog, cart, checkout, orders | categories, products, variants; `--starter commerce` adds a storefront. See [commerce.md](commerce.md) |

`hydro init` does the same in the current directory. Without `--admin-email`, `hydro new` only writes files and prints the next commands.

## 3. Add your own content type

```bash
hydro content-type new recipes
# edit content-types/recipes.yaml, then
hydro content-type validate
hydro serve --config config.yaml
```

The admin gets screens at `/admin/content/recipes` and, with `public: true`, the API gets `/api/v1/content/recipes`. See [content types](content-types.md).

## 4. Build the site

Use the API from any frontend. Two ready starting points:

- [`examples/sveltekit-starter`](../examples/sveltekit-starter): server-rendered SvelteKit site (SEO-friendly, no rebuild when content changes). `--starter sveltekit` copies it into `frontend/`; then `cd frontend && npm install && npm run dev`.
- [`examples/crm-starter`](../examples/crm-starter): a sales panel (deal board, companies, contacts, activities) for `--preset crm` projects. `--starter crm` copies it in.
- [`examples/commerce-starter`](../examples/commerce-starter): a storefront (catalog, cart, checkout, order history) for `--preset commerce` projects. `--starter commerce` copies it in.
- [`@hydrogo/client`](../packages/client): typed fetch client for any JavaScript/TypeScript frontend.

Publishing is live immediately: the frontend reads the API on every request (or through a short cache you control). Nothing needs rebuilding when an editor clicks Publish.

## Build from source

Needs Go 1.27+ (the toolchain is pinned in `go.mod`); the admin panel is committed pre-built.

```bash
go build -o bin/hydro ./cmd/hydro
```
