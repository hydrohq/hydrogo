# 0022: 2026-10-24 audit remediation

- Status: Accepted
- Date: 2026-10-24

## Context

The six weeks before this date added commerce, authz, and several
operational features in quick succession. A full-codebase audit (spaghetti
code, security, performance, architectural mistakes) was requested and run
in two passes -- an initial commerce-focused pass, then an expanded
whole-codebase pass -- before fixing anything, so the remediation below
could be planned and executed as one ordered sequence instead of
discovered and patched piecemeal.

Six critical findings drove the plan, split into one PR per phase, each
based on the previous phase's branch:

1. Hardcoded tenant-filter placeholder positions in
   `modules/commerce/repository/sqlstore` were wrong in several stock
   functions -- a future added SQL argument could silently shift which
   value bound to which `?`/`$N`, including the tenant filter itself.
2. API routes could declare a `Permission` with no checker enforcing it,
   and API keys inherited their owner's full role instead of being bounded
   to read access.
3. Rich text (news, content-type fields) was stored and served without
   sanitization; public content and forms had no rate limiting or
   abuse-resistant templating.
4. Client IP resolution trusted `X-Forwarded-For` unconditionally in
   several places, letting a direct caller spoof its own address past
   per-IP rate limits and bans.
5. Background workers were not drained on shutdown, admin audit write
   failures were silently dropped, and media assets had no cache headers.
6. `internal/` had no enforcement against importing a module package
   (only module-to-module imports were checked), and a config struct
   field name collided with an unrelated concept.

## Decision

Fix all six, plus everything each fix's own blast radius required, as six
sequential phases (branches `fix/audit-1-commerce` through
`fix/audit-6-docs`, one PR per phase):

- **Phase 1 (commerce integrity):** a `Store.scope()` helper computes the
  tenant-filter placeholder position from the argument list instead of a
  hardcoded number, so this class of bug cannot recur silently. Checkout
  now prices from a `PriceSource` (the content-type catalog) instead of
  trusting a client-sent or stale cart price; payment confirmation checks
  the paid amount against the order total; shipment claiming is a guarded
  atomic update so two concurrent "hand to carrier" calls cannot both call
  the carrier. A genuine, previously undiscovered pricing deadlock (a
  within-transaction price lookup against a one-connection pool) was found
  by live-testing the fix and is covered by a regression test.
- **Phase 2 (authz/identity):** `internal/requestmeta.IPExtractor` makes
  the proxy-trust model explicit (an untrusted hop stops the
  `X-Forwarded-For` walk); `router.NewWithPermissions` makes a declared
  `Permission` with no checker a startup error instead of a silent gap;
  API keys are forced to read-only content/commerce access regardless of
  the issuing admin's role; TOTP codes become single-use via a guarded
  step-claim update.
- **Phase 3 (content safety/abuse):** `internal/richtext.Clean` sanitizes
  rich text on write and on read (for pre-existing dirty records), applied
  to content-type fields and news; public content reads are rate-limited;
  the forms autoresponder only interpolates a fixed, non-free-text set of
  submitted fields into its template.
- **Phase 4 (data/ops):** `serveApp.spawn`/`waitForWorkers` drain
  background goroutines on shutdown instead of severing them; media
  responses carry visibility-appropriate `Cache-Control`; admin audit
  write failures are logged instead of swallowed; HTTP servers got
  explicit read/write timeouts.
- **Phase 5 (code health):** `moduleguard -deny-modules-root` makes
  "`internal/` never imports a module" an enforced rule with a named,
  tested exception list instead of an unchecked convention; the config
  field colliding with `commerce.ShippingProvider` was renamed to
  `ShippingFee`.
- **Phase 6 (docs):** this record; `config.example.yaml` documents every
  commerce/notification config key added across phases 1-5, corrected to
  not claim an `HYDRO_*` environment variable override for secrets that
  have none (PayTR, Yurtiçi Kargo, Telegram -- the same false claim
  existed in `internal/config/config.go`'s own doc comments and is fixed
  there too); `docs/commerce.md` is a new endpoint/CLI/config reference
  (none existed despite ten prior commerce ADRs); AGENTS.md rule 9's claim
  of an OpenAPI-to-TypeScript generation step for
  `admin/src/lib/api/types.ts` was removed -- no such step, `make
  generate` target, or `openapi-typescript` dependency exists; the file
  is and remains hand-maintained; `hydro new --preset commerce`'s module
  list and description were out of step with `config/presets/commerce.yaml`
  (the preset omitted the `commerce` module itself and described cart/
  checkout as not yet built, which stopped being true once phase 1 of the
  original commerce work landed) and are now reconciled; the
  `hydropost-news-starter` example's `bot-meta.php` had a hardcoded origin
  and filesystem DB path, now read from environment variables with
  generic fallbacks.

## Consequences

- `go run ./tools/check` passes after every phase. Package-level tests
  were additionally run against a real PostgreSQL instance (a dedicated
  test database, not the default `postgres` database, which carries stale
  migration-ledger rows that break schema-diff tests), and
  `tests/contract`'s RLS-dependent tests were run as a non-superuser role,
  since `NOBYPASSRLS` is required for those tests to mean anything.
- Not verified, and not claimed as verified: MySQL (no server available in
  this environment) and `-race` (no C toolchain/WSL available on this
  machine for cgo). Anyone merging this work should run both before
  relying on it.
- Deliberately deferred, named rather than hidden: aggregate SQL pushdown
  for a couple of reporting queries, multi-instance cache invalidation
  (the sqlite cache-invalidation monitor assumes one process), signed
  release manifests for more than the existing migration-ledger use case,
  switching order/payment amounts to a wider integer type, and an outbox
  topic filter. None of these were part of the six critical findings.
- Needs a human, not further autonomous work: merging this six-PR stack
  (with merge commits, not squash, so each phase's history survives),
  re-enabling GitHub Actions if it is currently off, installing a C
  toolchain or WSL to ever run `-race` on this machine, closing or
  resolving stale PRs #325-327, obtaining a MySQL instance for dialect
  parity testing, and obtaining real PayTR/Yurtiçi Kargo credentials to
  verify those two integrations end-to-end.
