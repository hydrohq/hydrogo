# ADR-0268: SEO, performance, and public discovery runtime

- **Status:** Accepted
- **Date:** 2026-09-28
- **Related:** [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0014](0014-publication-routes-and-erasure-recovery.md), [ADR-0125](0125-versioned-analytics-consent-contract.md), [ADR-0131](0131-durable-analytics-consent-store.md), [ADR-0217](0217-consent-gated-content-view-analytics.md), [ADR-0262](0262-pure-go-image-derivatives.md), [ADR-0264](0264-cgi-hosting-runtime.md), [ADR-0265](0265-fastcgi-and-persistent-html-cache.md), [ADR-0269](0269-optional-no-cookie-visitor-analytics.md)

## Context

Public editions need bounded social previews, sitemap discovery, redirects,
AI-readable page indexes, local font assets, and HTTP transfer optimizations.
These must work in the self-contained Go runtime and must not impose workers or
storage on editions that do not enable the relevant module.

Issue #226 proposes unique-visitor calculation from IP and user-agent hashes.
The optional implementation is governed by ADR-0269. Consent-based event
contracts in ADR-0125, ADR-0131, and ADR-0217 remain unchanged.

## Decision

- Pages may serve generated OpenGraph PNGs using the bundled Go font and a
  bounded in-memory cache. An optional disk cache is configured explicitly.
- Site and News sitemap output is split under the standard per-file URL and
  byte caps. SQL repositories stream only public route columns in stable order;
  a request holds at most the current 50 MB protocol file, not the full catalog.
  Active route sources get separate `pages` and `articles` files. A new content
  module must register its own public route stream before it can appear in the
  index; deferred Commerce does not add product routes today.
- Redirect rules are tenant-scoped, validated internal destinations. Exact,
  wildcard, and bounded Go regular-expression matches use an in-memory
  snapshot. SQLite and PostgreSQL persist the rules; hit counts are runtime
  aggregates until a separately bounded persistence policy is adopted.
- `/llms.txt` and `/llms-full.txt` expose only published page projections,
  apply bounded text/output limits, and cache output briefly. Page mutations
  invalidate the discovery cache.
- `hydro fonts add` and the Admin Studio font picker may contact the fixed
  Google Fonts CSS endpoint after an operator action. They write bounded,
  hashed WOFF2 assets locally and publish the chosen local stylesheet; the
  runtime serves only those local files and does not contact Google.
- Critical design tokens are inlined in server-rendered heads. Remaining CSS
  is preloaded and activated by the existing deferred vanilla script.
- ETag and compression middleware apply only to bounded, cacheable responses.
  Content with cookies, private/no-store policy, CSP nonces, ranges, or already
  encoded bodies is not transformed. Buffers are capped at 2 MiB and returned
  to a `sync.Pool`.
- Existing consent-based analytics remains cookie-free and consent-gated.
  Optional IP/user-agent-derived daily unique visitors follow ADR-0269 and are
  disabled unless the operator enables the Analytics module and configures the
  feature and deployment-shared hash key.

## Data and deployment compatibility

The redirect table is an additive expand migration with SQLite/PostgreSQL
definitions, tenant RLS in PostgreSQL, checksums, a rollback note, and a
SQLite/PostgreSQL repository contract. Old binaries ignore the table. Deploy
the migration before enabling redirect-rule writes; rollback keeps the table
and its operator-managed data. The Site module owns the migration, so inactive
Site editions do not install it.

All runtime caches are bounded and process-local. No new resident worker is
introduced. Sitemap/LLMS route responses are short-lived public cache entries;
content mutations must invalidate any local discovery cache. Blue/Green
instances can safely serve the same read-only public formats. Redirect rule
changes become visible on the editing instance immediately and on other
instances after restart; shared multi-instance invalidation remains outside
the current tenant-sharing support boundary.

Pure Go OpenGraph generation trades CPU for avoiding an external image process.
Its memory cache is limited to 64 entries by default and the optional disk
cache requires an operator-provided directory with adequate capacity. Font
localization is an explicit CLI action and has no build or runtime network
dependency.

## Verification

SQLite redirect CRUD and tenant isolation are covered by repository contract
tests. PostgreSQL executes the same contract when `HYDRO_TEST_POSTGRES_URL` is
configured. Sitemap limits, SEO rendering, compression negotiation, ETag
conditions, font localization, and discovery output have unit tests. Production
capacity and cache-hit claims remain subject to workload benchmarks.
