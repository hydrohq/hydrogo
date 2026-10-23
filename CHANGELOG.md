# Changelog

All notable changes to HydroGo are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

- `hydro generate tests` writes a Playwright spec per content type (`e2e/generated/content/<id>.spec.ts`) proving every admin role gets exactly the read/write access its declared permissions say it should — no hand-written test needed for a new content type. `hydro new` now always scaffolds an `e2e/` Playwright harness, and `hydro seed-test-users` creates the admin/editor/viewer accounts the generated specs log in as. See `docs/decisions/0011-generated-role-permission-tests.md`.
- `examples/crm-starter`: a SvelteKit sales panel (deal board, companies, contacts, activities, tasks; English and Turkish) for `--preset crm` projects. HydroGo's session cookie stays on the panel server, the browser only gets an opaque cookie. `hydro new --starter crm` copies it into `frontend/`; CI runs its checks and tests.
- `hydro new --preset crm`: a CRM backend with no pages or news (modules core, auth, media, collections) and four stock types, companies, contacts, deals and activities (`content-types/crm/`), linked by relations and filterable by stage, owner, due date and more.
- `scripts/ci-local.sh` runs the CI jobs on a developer machine with act.
### Fixed
- The Dependency review workflow pinned `actions/setup-go` to a commit that does not exist (one wrong character) and could not start; `tools/actionguard` now also rejects one action version pinned to two different commits.
- `.dockerignore` skips `.cache`, so a local `docker build` no longer sends gigabytes of cached bundles.
- PostgreSQL: webhooks and any other outbox delivery never went out; the worker could not read back the timestamps it had stored (TEXT column, Go's `String()` form). Outbox lease tests now also run on PostgreSQL and MySQL.
- `hydro backup/restore postgres` failing on the forced row-level security of the audit tables now says to set `HYDRO_PG_PRIVILEGED_ROLE`; `docs/deployment.md` describes what that role needs.
- Maintenance commands (migrate, rollback, db, upgrade check, backup, restore, audit, create-admin, service-account, tenant, jobs, ban) no longer need the hook and webhook secrets in their environment; `serve`, `doctor` and `validate` still do.
- A state-changing request that a browser marks `Sec-Fetch-Site: cross-site` is refused with 403, even with a valid CSRF token.
- `hydro new --preset crm` turns on admin write logging (`audit.log_request_bodies`), as erp-lite does.
- crm-starter: amounts with cents keep them (`TRY 1,234.50`), and the dashboard says "1 deal".
- A missing or wrong CSRF token answers 403 in the API error envelope (with the request id) instead of plain text.
- `hydro upgrade check` on a database that still has pending migrations says which one and to run `hydro migrate`, instead of "sql: no rows in result set".
- `hydro help` lists the `crm` and `erp-lite` presets and the `--starter` option.
- Deleting a record that other records still point to answers 409 ("other records still refer to this one") on SQLite, PostgreSQL and MySQL, instead of 500.
- The audit screen works in projects without the pages module (crm, erp-lite); the route was only registered with pages.
- Hook events for a state step and a delete carry the stored record's fields, so a service rule such as `total > 0` sees the data instead of rejecting every step.
- Admin: relation fields are searchable pickers that show names (also in lists) instead of raw id boxes; editing a record no longer saves unset numbers as 0; lists page and search on the server, so every record is reachable beyond the first 25; the System and Audit screens stop calling endpoints their project does not have; the rich-text editor no longer trips the content security policy; a refused state step shows the rule's message once.
- MySQL: concurrent audited writes no longer deadlock on the audit chain head (found by running the new SQL on a real server), and a transaction the database aborts for a lock race (deadlock, lock wait timeout, serialization failure) now answers 409 so the request can be retried, instead of 500.

### Added
- Admin: a "Services" tab under System lists the configured hook services (host, calls and failures since start, last call, content types using them, schedules). It appears only when `hooks:` is configured; `GET /api/v1/admin/system/services`.
- The admin schema leaves out fields whose `permission` the signed-in user lacks, so forms and lists no longer show a label for a value the API never sends.
- `@hydrogo/client`: `listContent` takes `after` (cursor paging), `expand` and filter operators (`{amount: {gte: "100"}}`), and the list metadata types `next_cursor`.
- Content events: webhooks can subscribe to `content.created`, `content.updated` and `content.deleted`. They are written to the outbox in the record's transaction and delivered signed with retries, in every preset (core migration `202610210004_core_outbox_events` creates the outbox where the pages module did not).
- `hydro dev` in a project directory (one with a `config.yaml`) runs `hydro serve --dev` and any number of side services in one terminal: `hydro dev --service ledger="go run ./automation"`. Output is prefixed per process, Ctrl-C stops all of them, and if one exits the others stop too.
- `examples/side-service`: a standard-library Go service that verifies HydroGo's signed hook and schedule requests (HMAC, replay window) and holds your rules; `tools/check` runs its tests.
- `hydro audit relink` links audit events written while `audit.chain` was off; the server warns at start when a chain exists but the option is off. A revoked service account can be created again with `hydro service-account create`.
- Tamper-evident audit log: `audit.chain: true` links every new audit event into a per-tenant SHA-256 chain (migration `202610210003_audit_chain`); `hydro audit verify` recomputes it and checks an `--anchor` saved outside the database, `hydro audit head` prints the heads to save. On by default with `--preset erp-lite`.
- Period lock: `period_lock: {field: issue_date, closings: period_closings, through: closed_through}` refuses to create, change, step or delete records dated on or before the latest closing date. The erp-lite preset closes periods with the append-only `period_closings` type (permission `erp.periods.close`) and locks orders, invoices and payments.
- Rollup fields: `rollup: {from: order_lines, via: order, sum: line_total}` makes a number field the server-kept total (or count, without `sum`) of the records of another type pointing at this one. It is updated in the same transaction as every create, update, delete or move of such a record, and a parent locked by a readonly rule refuses changes to its lines. Computed fields can use rollups, so `total` can be `subtotal - discount + tax`. The erp-lite preset uses it for order and invoice totals.
- `hydro new --preset erp-lite`: a light ERP backend (no pages or news) with products, warehouses, accounts, sales orders with lines, invoices with lines and payments (`content-types/erp/`). It uses the new capabilities: gapless document numbers, exact money with computed totals, state machines with step permissions (`erp.invoices.post`, `erp.invoices.void`, `erp.orders.fulfil`, `erp.orders.cancel`), locked posted invoices and atomic order-with-lines writes through `_batch`.
- Schedules for hook services: `hooks[].schedules` in the config call a side service at cron times (UTC) with a signed `hook.schedule.<name>` request.
- Synchronous hooks: a content type can ask a side service before a change is saved (`hooks: [{on: transition, to: posted, service: ledger}]`; create, update, delete and state steps, also inside `_batch`). The services are defined by the operator under `hooks:` in the config (URL, `secret_env`, timeout up to 10s); the request is signed like webhooks, 2xx allows, 4xx rejects with the service's message, anything else rejects too unless `fail_open`.
- `hydro service-account create|revoke`: identities for side services, a user nobody can sign in as with only the permissions you list and an expiring bearer token (audit shows `service+<name>@hydrogo.invalid`).
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
- Schema baseline (ADR 0009): new databases are built from generated per-module schema scripts under `migrations/baseline/` instead of replaying every migration (SQLite, PostgreSQL, MySQL). The migration ledger reads exactly as if the chain had run, migrations written later still run from the chain, and a database that was upgraded halfway keeps running the chain. Fresh installs are faster (SQLite with every module 158 ms to 38 ms; MySQL 8.4 1.3 s to 0.54 s). `go run ./tools/baselinegen` regenerates it; tests prove it equals the chain on schema, row security policies, seed rows and ledger, and `migrationguard` checks it on every run.

### Fixed
- Saving a record with a `many: true` relation field panicked instead of checking the related records.
- Projects without the `pages` module (such as the `crm` preset) failed to sign in because `audit_events` was owned by `pages`; the audit log is now created by a core migration (no change where the pages migrations already built it).
- A module can be enabled on a database that was created without it, for example `pages` on a `crm` project; `hydro migrate` used to stop at the first pages migration because the audit table already existed. On PostgreSQL the module's tables now also get their tenant row-level security policies, which the old chain skipped for tables created after its RLS migration. Existing migration history is unchanged.
- MySQL 8: the collections sequence migration failed because `last_value` is a reserved word there (MariaDB accepts it), so the collections module could not be migrated; the migration and the sequence queries quote it. If you applied `202610210001_collection_sequences` on MariaDB from a development build, reset that database: its MySQL checksum changed.
- `hydro backup mysql` refuses MariaDB client tools against a MySQL server before writing anything: they dump generated column values that MySQL rejects on restore, so the backup could not have been restored. Use MySQL's `mysqldump` (or `HYDRO_MYSQLDUMP` / `HYDRO_MYSQL`).
- Audit chain: verifying a chain no longer reports "audit events written after the chain began are not in it" for an event that shares the first link's timestamp. Windows clocks are coarse enough for the last event before chaining and the first chained one to carry the same instant, which looked like a missing link; such ties now count as written before the chain, and a really later unlinked event is still reported.
- CI: the arm64 image build runs its Node and Go stages on the build machine (they cross-compile) instead of under emulation, which took more than the job limit; the Windows job now fails when any of its commands fails (only the last one counted) and has a limit that fits its two test passes.
- Security scan: `gosec` runs with a pinned `x/tools` (`tools/gosec`) because the release used by CI could not read Go 1.27's export data and reported nothing useful. With real type information it found 186 more places on top of the 105 it had flagged; each is now fixed or carries a reason. Behaviour changes: the pages import caps the upload at 16 MiB; snapshot and archive reads cannot follow a path swapped for a symlink while a backup or extension package is being read (`os.Root`); the restore marker files and snapshot copies are owner-only (0600, were 0640); `HYDRO_DEBUG_PPROF_ADDR` serves profiling from its own mux instead of `http.DefaultServeMux`; the e-mail brand colour is checked again where it reaches an inline style.
- Admin: the dark-theme bootstrap is an external script, so the content security policy no longer blocks it; the System > Services tab shows each service's schedules (the cron expression was missing); the content list toolbar wraps on phone width instead of running off the screen; the notification bell no longer requests `/api/v1/admin/notifications` where the pages module is off, and the admin loads the schema once after sign-in instead of twice.

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
