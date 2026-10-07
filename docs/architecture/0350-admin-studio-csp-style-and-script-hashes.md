# ADR-0350: Allow verified Admin Studio inline assets through CSP hashes

- **Status:** Accepted
- **Date:** 2026-10-07
- **Supersedes:** None
- **Superseded by:** None

## Context

Admin Studio emits embedded, deterministic CSS and JavaScript in server-rendered
pages. Hash-generation helpers already exist in `internal/ui/csp.go`, but the
runtime did not pass those hashes to the security header middleware. The strict
CSP therefore blocked Studio layout and interactions in Chromium even though
the HTML was present. Login appearance CSS can also vary by Edition setting and
must be authorized from the exact rendered response.

## Decision

1. `server.Config` may receive explicit script and style SHA-256 source hashes.
   The server passes them to the existing central security-header middleware.
2. The HydroGo Admin host supplies only the hashes returned by its typed,
   machine-owned `ui.AdminScriptHashes()` and `ui.AdminStyleHashes()` catalogs.
   This keeps executable inline scripts and embedded style blocks blocked by
   default unless their exact source is cataloged.
3. The login route renders the page into a bounded in-memory buffer, computes
   hashes for its actual inline script/style blocks, and merges those sources
   into the already-established CSP before writing the response body. It does
   not enable `unsafe-inline` for scripts or style blocks.
4. Member login/registration computes the same exact-body hashes after
   redacting and encoding the response. Redirect paths embedded in inline
   JavaScript are JSON encoded before hashing so an attacker-controlled path
   cannot become executable code.
5. No client Edition can supply arbitrary hashes through generated metadata.
   Any future extension-owned inline code requires an explicit reviewed host
   integration and a corresponding security decision.

## Alternatives

- Allow `unsafe-inline`: rejected because it permits injected scripts and
  embedded styles without source integrity.
- Move all Admin styling and scripts to separate static assets now: rejected
  as a larger deployment and asset-lifecycle change; embedded assets are an
  intentional self-contained Studio design.
- Hash only the login page: rejected because the browser showed the same
  blocking on existing Admin area CSS and scripts.

## Consequences

### Benefits

- The verified Admin shell CSS and scripts execute under the strict policy.
- The server keeps a single policy builder and exposes no route-level CSP
  weakening switch.
- The login background may continue to vary while only the exact rendered
  CSS receives a hash source. Customer member sign-in and registration scripts
  work without weakening CSP.

### Costs and risks

- Changing any embedded Admin source requires its hash catalog to remain
  synchronized; live browser and middleware tests guard this contract.
- Login and member auth HTML are buffered before output. Their sizes are
  bounded by fixed templates and validated settings.

## Scale impact

- Compact: fixed CSP hash strings and one small login response buffer; no
  service, asset, worker, connection pool, or runtime dependency is added.
- Standard: same request-local behavior.
- Distributed: no shared state or coordination is added.

## Database impact

- SQLite: unchanged; existing appearance reads are unchanged.
- PostgreSQL: unchanged; no SQL behavior changes.
- Future adapters: no database contract dependency.

## Cache impact

No cache keys or TTLs change. Login remains request-local and carries CSRF
state; no new cache or invalidation behavior is introduced. CSP hash catalog
construction occurs once from compiled strings, not per request.

## Deployment and compatibility

No migrations, assets, workers, sessions, or background jobs are introduced.
Old binaries continue to serve pages but retain the prior CSP rendering bug;
new binaries add the optional hash slices to the internal server configuration.
Rollback is a binary switch. Blue/Green instances need no shared-state change.

## Validation

- Middleware and member response tests verify configured script/style sources
  are included while strict defaults remain in place.
- Login renderer tests verify generated hashes correspond to its inline
  blocks.
- Chromium Admin acceptance verifies login, Admin layout, and management
  routes with computed styles applied.
- `go test ./...`, moduleguard, migrationguard, and all Edition importguards
  remain required release checks.

## References

- [Security headers middleware](../../internal/security/headers.go)
- [Admin CSP hash catalog](../../internal/ui/csp.go)
- [Admin Studio product audit plan](../planning/ADMIN_STUDIO_PRODUCT_AUDIT_AND_PLAN.md)
