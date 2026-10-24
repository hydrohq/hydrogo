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
  generic fallbacks. The same false `HYDRO_*` secret-override claim fixed
  in `config.example.yaml` and `internal/config/config.go` was also found
  and fixed in `modules/commerce/providers/paytr/paytr.go` and
  `modules/commerce/providers/yurticikargo/yurticikargo.go`'s own doc
  comments.
- A new end-to-end test, `modules/commerce/providers/checkout_to_shipment_test.go`,
  chains checkout, `StartPayment` and a PayTR callback, and `CreateShipment`
  through the real `commerce.Service` against mock PayTR and Yurtiçi Kargo
  HTTP servers in one run -- the two providers' own unit tests each mock
  only their own call; nothing previously exercised the full hand-off
  between them. It also covers a carrier refusal correctly releasing the
  shipment claim so a retry can still succeed. Both provider packages
  gained unparsable-response, connection-failure and already-canceled-
  context tests, none of which existed before (only the documented-happy-
  path and documented-refusal shapes were covered).

## Consequences

- `go run ./tools/check` passes after every phase. Package-level tests
  were additionally run against a real PostgreSQL instance (a dedicated
  test database, not the default `postgres` database, which carries stale
  migration-ledger rows that break schema-diff tests), and
  `tests/contract`'s RLS-dependent tests were run as a non-superuser role,
  since `NOBYPASSRLS` is required for those tests to mean anything.
- **MySQL and `-race`, previously listed as "not verified" in an earlier
  draft of this record, have since been verified.** Docker was installed
  into this machine's existing WSL2 Ubuntu distribution (which already had
  Go 1.27 and a C toolchain, unused until now), a MySQL 8 container was
  started there, and the full suite ran clean against it
  (`HYDRO_TEST_MYSQL_URL`, zero failures across the whole module tree,
  plus the two tests that need a literally empty target database run
  separately against a dedicated empty one). The full suite also ran
  clean under `go test ./... -race` in the same WSL environment, and the
  new end-to-end commerce/provider tests were additionally run under
  `-race` on their own. Still not verified, because nothing in this
  environment can: the real PayTR and Yurtiçi Kargo services themselves
  (see below).
- Deliberately deferred, named rather than hidden: aggregate SQL pushdown
  for a couple of reporting queries, multi-instance cache invalidation
  (the sqlite cache-invalidation monitor assumes one process), signed
  release manifests for more than the existing migration-ledger use case,
  switching order/payment amounts to a wider integer type, and an outbox
  topic filter. None of these were part of the six critical findings.
- Needs a human, not further autonomous work: merging this six-PR stack
  (with merge commits, not squash, so each phase's history survives),
  re-enabling GitHub Actions if it is currently off, closing or resolving
  stale PRs #325-327, and obtaining real PayTR/Yurtiçi Kargo merchant/
  carrier credentials to verify those two integrations against the actual
  services -- a mock HTTP/SOAP server, however thorough, can only prove
  this code does what it claims to do with the shapes it was told to
  expect, never that those shapes match what the real services send.
