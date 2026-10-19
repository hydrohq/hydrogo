# Getting started

HydroGo is a headless CMS: one binary gives you the admin panel at `/admin` and a JSON API under `/api/v1`. You build the public site as a separate frontend.

## 1. Install the `hydro` command

**Windows:** download `HydroGo-Setup-<version>-windows-amd64.exe` (or `arm64`) from the [latest release](https://github.com/ozyildirims/hydrogo/releases/latest) and run it. Leave "Add hydro to PATH" ticked, then open a new terminal. No administrator rights are needed (it installs per user; choose "install for all users" to use Program Files).

**Linux / macOS:**

```bash
curl -fsSL https://github.com/ozyildirims/hydrogo/releases/latest/download/install.sh | sh
```

**Windows without the installer (PowerShell):**

```powershell
irm https://github.com/ozyildirims/hydrogo/releases/latest/download/install.ps1 | iex
```

**Docker:** `docker run -p 8080:8080 -v hydro-data:/app/data ghcr.io/ozyildirims/hydrogo:latest` (mount your own `config.yaml` at `/app/config.yaml`; see [deployment](deployment.md)).

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
- [`@hydrogo/client`](../packages/client): typed fetch client for any JavaScript/TypeScript frontend.

Publishing is live immediately: the frontend reads the API on every request (or through a short cache you control). Nothing needs rebuilding when an editor clicks Publish.

## Build from source

Needs Go 1.27+ (the toolchain is pinned in `go.mod`); the admin panel is committed pre-built.

```bash
go build -o bin/hydro ./cmd/hydro
```
