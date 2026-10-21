# Changelog

All notable changes to HydroGo are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

- `examples/crm-starter`: a SvelteKit sales panel (deal board, companies, contacts, activities, tasks; English and Turkish) for `--preset crm` projects. HydroGo's session cookie stays on the panel server, the browser only gets an opaque cookie. `hydro new --starter crm` copies it into `frontend/`; CI runs its checks and tests.
- `hydro new --preset crm`: a CRM backend with no pages or news (modules core, auth, media, collections) and four stock types, companies, contacts, deals and activities (`content-types/crm/`), linked by relations and filterable by stage, owner, due date and more.
### Added
- CSV transfer for content types: `GET /api/v1/admin/content/<type>/_export` (list filters apply; row access, field permissions and spreadsheet-formula defusing included; up to 50,000 records) and `POST .../_import` (CSV body, header of field ids, up to 1,000 rows, all-or-nothing, `?dry_run=1` to validate).
- Field permissions and aggregation: a field can name a `permission`; without it the field is left out of admin reads, ignored on writes (stored values stay), and cannot be filtered or sorted by. `GET /api/v1/admin/content/<type>/_aggregate?group_by=&sum=` returns counts and exact decimal sums (per group), following the same filters, row access and field permissions (up to 20,000 matching records). Types with field permissions cannot be `public`.
- Row access: `access: {owner: owner, all: deals.read_all}` limits users without the `all` permission to records whose `owner` text field holds their e-mail address (lists, reads, edits, deletes, revisions, `expand` and `_batch`); they become the owner of what they create. Admins and holders of the permission see everything. The public API is unaffected.
- Content types can declare `permissions: {read: crm.deals.read, write: crm.deals.write}`: the type is then guarded by those instead of the generic content permissions (batch writes included). The names, and the `permission` of state steps, join the permission catalog, so they can be granted per user on the Users screen.
- Admin: navigation, dashboard widgets and notifications of modules that are not enabled are hidden, so a `crm` project no longer shows Pages/News/Members or makes 404 calls.
- Admin: select rows in a content type's list and delete them in one atomic batch; types with `states` show the allowed steps as "Move to <state>" buttons in the edit window; server-set fields (sequence numbers, computed values, states) are shown read-only.
- Rules and state machines on content types: `rules` (`assert`, and `when` + `readonly` to lock records) and `states` (a select field that only changes through `POST /api/v1/admin/content/<type>/<id>/state`, with optional per-step condition and permission). Conditions use an expression language with comparisons, `&& || !`, text, `empty()` and exact decimal arithmetic.
- Money and computed fields: `decimals: 2` fixes a number field's decimal places (more places are rejected, never rounded), and `computed: "quantity * unit_price"` makes the server calculate a number field on every save with exact decimal arithmetic (`+ - * /`, parentheses, `abs`, `min`, `max`, `round`). Computed values are stored, filterable, read-only in the admin, and ignore client input.
- Sequence fields: `sequence: {prefix: "INV-{yyyy}-", pad: 5}` on a text field numbers new records without gaps (per tenant, type and prefix period). The number is taken in the record's own transaction, never changes, and a client value is ignored. New table `hydro_collection_sequences` (all three dialects).
- `POST /api/v1/admin/content/_batch`: up to 200 create/update/delete operations on content records in one transaction (all or nothing), with `$ref:N` to refer to a record created earlier in the same batch; errors name the failing operation.
- `GET /api/v1/admin/branding` (public): admin name, logo, accent colour and default language, so the sign-in page is branded and in the project's language.
- `GET /api/v1/admin/me` returns the saved admin `locale` (a browser-bound cookie set by `PUT /api/v1/admin/account/locale`); `PUT` with an empty locale clears it.
- `examples/sveltekit-starter`: `npm run smoke` starts the production build against a stub API and checks 404 and 502 handling; CI runs it.
- Content type filters: `filterable` now also works on boolean, date, datetime and single relation fields, and the API accepts `filter[field][ne|gt|gte|lt|lte|in|null]=value` besides the exact match (see `docs/content-types.md`).
- Content type APIs (public and admin) accept `?expand=field,field` on lists and single records: the records that relation fields point to come back under `expanded` in the same response. The public API only expands relations to public types and hides related drafts.
- Cursor paging for content type lists (public and admin API): when `metadata.has_more` is true, `metadata.next_cursor` holds a cursor to pass as `?after=` for the next page. It has no offset limit and works with the default order and `sort=[-]created_at|updated_at`.
- PostgreSQL: fields marked `filterable` (and the `status` of drafts types) get a database index, built at startup and dropped when the flag is removed. On a table of one million records a filtered or sorted list page went from 100–600 ms to about 1 ms in a local test, and cursor paging stays fast deep into the list. Other databases are unchanged. Adding the first indexes to a large table briefly blocks writes while they build.

### Fixed
- Saving a record with a `many: true` relation field panicked instead of checking the related records.
- Projects without the `pages` module (such as the `crm` preset) failed to sign in because `audit_events` was owned by `pages`; the audit log is now created by a core migration (no change where the pages migrations already built it).

### Changed
- Content types can hold up to 100,000 records per tenant by default (was a fixed 10,000); set `limits.records_per_type` in `config.yaml` to change it, or a negative value to remove the cap. Creating a record no longer runs a `COUNT(*)` when there is no cap. `?offset=` stays limited to 10,000.
- Admin: the English and Turkish catalogs are split per area under `admin/src/lib/i18n/messages/`; editor, chart axis, status/role labels and more screens are translated.

## [0.2.0] - 2026-10-19

### Added
- **`hydro new DIR` / `hydro init`**: scaffold a project (config, content types, SQLite database, admin user) from a preset: `minimal`, `blog`, `corporate`, `news`, `docs`.
- **`hydro content-type new|validate`** and a JSON Schema for content type YAML (`docs/schemas/content-type.schema.json`).
- **Releases and installers**: tagged releases publish binaries for Linux, macOS and Windows (amd64, arm64), a Windows `Setup.exe` that adds `hydro` to PATH, macOS `.pkg` installers, Linux `.deb` and `.rpm` packages, `install.sh`, `install.ps1`, `SHA256SUMS` and a multi-arch container image on GHCR (`.github/workflows/release.yml`).
- **`@hydrogo/client`** (`packages/client`, private, never published to npm; moved from `examples/headless-starter`): typed client, now with `listContent` / `getContent` for content types.
- **`examples/sveltekit-starter`**: server-rendered starter (home, news list/article, pages, locale prefixes, canonical/Open Graph/JSON-LD, `sitemap.xml`, `rss.xml`, `robots.txt`, Dockerfile). `hydro new --starter sveltekit` copies it into `<project>/frontend`.
- **Admin language**: English and Turkish admin UI; each user picks theirs under Account, and the project default is set under Site settings (`admin.default_locale`).
- **Admin branding**: panel name, logo URL and accent colour under Site settings (`admin.brand_name`, `admin.brand_logo_url`, `admin.brand_color`).
- **Audit request content**: with `audit.log_request_bodies: true` (`HYDRO_AUDIT_LOG_REQUEST_BODIES`) every admin API write is recorded as an `admin.request` event with a redacted body, shown in the audit log.
- Docs: getting started, content types, deployment, upgrading and versioning; `CONTRIBUTING.md`; issue and PR templates; ADR 0007.
- `hydro version` reports the release version injected at build time.

### Changed
- Stock content types (`content-types/`) now have English labels and descriptions; ids and option values are unchanged, so existing data is unaffected.
- Admin lists and the audit log now read the `metadata.total` field the API actually sends (they showed 0 results before).
- **Breaking for client code:** `@hydrogo/client` pages and news methods now unwrap the `{data, metadata}` envelope the API returns (they previously returned the raw envelope typed as the payload).
- `examples/svelte-starter` no longer links to a specific deployment.

### Removed
- Deployment-specific and stale files: `scripts/deploy-ardalabs.ps1`, `tools/agencybot`, `security-report/`, `PRODUCT.md`, `DESIGN.md`, `module-specs/` (its generator command no longer exists), `docs/archive/` (still in git history), and the tracked `config.yaml` (now ignored; use `config.example.yaml` or `hydro new`).

## [0.1.0] - 2026-10-19

### Added
- **Single-Binary Headless CMS Core**: Core engine providing pages, news, media, members, site configuration, and collections over a clean REST API.
- **Embedded Svelte 5 Admin Panel**: Full SPA panel embedded in Go binary (`/admin`) powered by shadcn-svelte and TanStack Query.
- **Schema-Driven Content Types**: Declarative YAML schema engine (`content-types/`) generating API endpoints and admin UI automatically.
- **Multi-Dialect Database Engine**: Full support and automated migrations for SQLite, PostgreSQL (with RLS), and MySQL.
- **FastCGI & CGI Production Runtime**: Native `hydro serve --fastcgi` and `--cgi` modes for shared hosting and Apache/Nginx web servers.
- **Native View Analytics**: Atomic, privacy-preserving view counter (`POST /api/v1/news/views`, `GET /api/v1/news/views`) integrated with `analytics_content_daily`.
- **Social Crawler Interceptor**: Server-side metadata pre-rendering (`bot-meta.php`) serving OpenGraph, Twitter Cards, and NewsArticle JSON-LD to WhatsApp, Facebook, X (Twitter), LinkedIn, Telegram, Discord, and Google Inspection bots.
- **Self-Hosted Privacy Typography**: Self-hosted Public Sans fonts (`@fontsource/public-sans`) removing Google Fonts tracking.
- **Automated Deployment Pipeline**: Automated build and FTP deployment toolchain (`scripts/deploy-ardalabs.ps1`) supporting `-FrontendOnly` hot updates.

### Changed
- **Decoupled View Tracking**: Replaced fragile, file-based SvelteKit view counter (`views.json`) with persistent Go backend API endpoints.
- **Endpoint Hardening**: Added token authorization guards to `/api/admin/rss-ingest` and `/api/translate`.
- **TTS Abuse Prevention**: Protected `/api/tts` with sliding-window rate limiting (10 req/min) and 1,000 character payload limits.

### Deprecated
- **Legacy Commerce Peripherals**: Frozen legacy commerce tables and `internal/contracts/customerorders` capability contracts (ADR 0006).

### Security
- **Credential Sanitation**: Completely removed hardcoded fallback passwords and admin credentials across all starter scripts and seeds.
- **Secure Cookie & HSTS Presets**: Enforced strict transport and cookie security presets for production HTTPS environments.
