# ADR-0278: Indexed taxonomy recommendations for optional content

- **Status:** Accepted for News and Portfolio; Pages behavior is superseded by [ADR-0279](0279-indexed-related-pages.md)
- **Date:** 2026-09-28
- **Related:** [ADR-0273](0273-sectoral-content-modules.md), [ADR-0039](0039-conditional-cross-module-migrations.md)

## Context

Related-content recommendations must use bounded indexed lookups, remain
tenant-isolated, and avoid startup or schema cost for Editions that do not
enable the owning content module.

## Decision

Portfolio retains its canonical tag list and industry category in each typed
payload and mirrors namespaced, normalized term keys into the optional
`sectoral_portfolio_tags` relation table. The module migration backfills the
relation table from existing payloads. Repository create/update synchronizes
both representations inside the caller's transaction; deletion cascades from
the portfolio record. Recommendation lookup joins the tag index, excludes the
source record and drafts, applies tenant and locale scope, ranks by the number
of shared tags/categories, and caps the caller limit at ten. Public Templ output receives
the bounded projection from the service and owns only presentation.

The module-specific table and index are created only when Portfolio is
enabled. SQLite and PostgreSQL use the same logical relation and
tenant-scoping contract. Page recommendations use the separately versioned
page taxonomy snapshot defined by ADR-0279.

## Verification and operational impact

SQLite tests cover migration selection, tag synchronization on writes,
ranking, current/draft/tenant exclusion, and Templ output. PostgreSQL SQL has
matching expand-only DDL and RLS, with production execution gated on the
configured PostgreSQL contract environment. Storage grows by one relation row
per normalized tag/category plus one lookup index entry; request work is limited by the
caller cap and uses the `(tenant_id, tag, content_id)` index. No worker or
cache is introduced.

Rollback is forward-only by default. After a backup, removing the optional
relation table does not lose tags because payloads remain canonical.
