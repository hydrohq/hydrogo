# Getting started

HydroGo is a headless CMS: one binary gives you the admin panel at `/admin` and a JSON API under `/api/v1`. You build the public site as a separate frontend.

## 1. Install the `hydro` command

Download the installer for your system from the [latest release](https://github.com/hydrohq/hydrogo/releases/latest):

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
curl -fsSL https://github.com/hydrohq/hydrogo/releases/latest/download/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://github.com/hydrohq/hydrogo/releases/latest/download/install.ps1 | iex
```

**Docker:** `docker run -p 8080:8080 -e HYDRO_HTTP_ADDRESS=0.0.0.0:8080 -v "$PWD/config.yaml:/app/config.yaml:ro" -v hydro-data:/app/data ghcr.io/hydrohq/hydrogo:latest serve --config /app/config.yaml` (the image has no default command and listens on loopback unless `HYDRO_HTTP_ADDRESS` is set; see [deployment](deployment.md)). To run without managing a server, see the managed platforms section of [deployment](deployment.md#managed-container-platforms-flyio-cloudflare-containers-render-railway) (Fly.io, Cloudflare Containers, Render, Railway).

Check it: `hydro version`. Every download has a checksum in `SHA256SUMS`; the install scripts verify it.

## 2. Create a project

```bash
hydro new mysite --preset blog --admin-email you@example.com
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
| `docs` | documentation site | + FAQ |
| `news` | news portal | + homepage sections, masthead, columnists, announcements, ads, polls, videos |
| `tr-news` | regional Turkish news portal | same editorial modules as `news` with regional defaults |
| `saas` | SaaS landing and leads | features, pricing tiers, FAQs, waitlist, settings |
| `commerce` | storefront: catalog, cart, checkout, orders | categories, products, variants, product images. See [commerce.md](commerce.md) |
| `boutique` | digital boutique & storefront | store settings, boutique categories, products, orders |
| `restaurant` | restaurant & QR digital menu | site settings, menu categories, menu items, reservations |
| `crm` | sales (no website: pages and news are off) | companies, contacts, deals, activities |
| `erp-lite` | light ERP back office (no website) | products, warehouses, accounts, sales orders, invoices, payments, period closings |

`hydro init` does the same in the current directory. Without `--admin-email`, `hydro new` only writes files and prints the next commands.

## 3. Add your own content type

```bash
hydro content-type new recipes
# edit content-types/recipes.yaml, then
hydro content-type validate
hydro serve --config config.yaml
```

The admin gets screens at `/admin/content/recipes` and, with `public: true`, the API gets `/api/v1/content/recipes`. See [content types](content-types.md).

## 4. Connect a frontend or AI assistant

Consume the REST API, SSE streams, and OpenAPI schema (`/api/v1/openapi.json`) from any frontend or coding agent:

- [`@hydrogo/client`](../packages/client): typed TypeScript fetch client for any JavaScript/TypeScript frontend (SvelteKit, Next.js, Astro, React, Vue).
- **AI Coding Agents (`hydro mcp`):** run `hydro mcp --config config.yaml` (or register it in `.cursor/mcp.json` / Claude Desktop) so your AI assistant can inspect schemas (`schema_get`), manage records, and verify audit logs directly.
- [`examples/side-service`](../examples/side-service): reference daemon/side-service integration over the HydroGo API.

Publishing is live immediately: the frontend reads the API on every request (or through a short cache you control). Nothing needs rebuilding when an editor clicks Publish.

## Build from source

Needs Go 1.27+ (the toolchain is pinned in `go.mod`); the admin panel is committed pre-built.

```bash
go build -o bin/hydro ./cmd/hydro
```
