# ADR-0276: Site appearance tokens and contrast validation

- **Status:** Accepted (color, typography, and existing media references)
- **Date:** 2026-09-28
- **Related:** [ADR-0273: Optional sectoral content modules](0273-optional-sectoral-content-modules.md), [ADR-0197: Admin UI locale preference and local catalog](0197-admin-ui-locale-preference-and-local-catalog.md)

## Context

Site owners need to change the public theme's brand colors, typeface, and
existing logo assets without editing theme source. Color choices must remain
legible in the provided white-text button treatment.

## Decision

The opt-in Site module stores a bounded appearance contract as public keys in
the existing `site_settings` repository. The Admin Studio editor writes all
appearance keys in one transaction and audits the update. Core accepts a
small font allowlist and same-origin `/media/` or `/theme/static/` asset paths
only. Remote asset URLs are rejected to avoid adding tracking and external
runtime dependencies.

Primary and accent colors are normalized as six-digit hex. White-on-color
contrast must be at least 4.5:1 for light and dark token sets. The browser
reports the WCAG AA result immediately; the service repeats validation before
persisting. Validated tokens are emitted in a CSP nonce-bearing style block,
with dark tokens selected through `prefers-color-scheme`. Existing theme
templates may continue to use their standard CSS tokens.

## Compatibility and operational impact

- No migration or new table is needed. SQLite and PostgreSQL use the existing
  tenant-scoped `site_settings` contract. Old binaries ignore the new keys;
  new binaries fall back to product defaults when no appearance keys exist.
- The existing per-request public settings read supplies the tokens. No cache,
  worker, background service, or additional connection is introduced.
- The feature creates no runtime work when the Site module is inactive.
- Saved asset paths reference the existing Media or Theme asset surfaces; this
  change does not create a second upload store.
- Rollback is compatible: an older binary ignores the added key/value rows,
  which remain available if a newer binary is restored.

## Consequences

Appearance changes apply on the next public render. The Admin surface covers
brand colors, supported font families, light/dark logo paths, favicon, and
social sharing image. Theme-specific controls remain in Edition-owned theme
contracts rather than becoming arbitrary CSS injection.
