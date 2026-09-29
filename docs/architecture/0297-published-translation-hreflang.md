# ADR-0297: Published translation hreflang links

- Status: Accepted
- Date: 2026-09-29
- Related: [ADR-0012](0012-content-media-and-portability.md), [ADR-0014](0014-publication-routes-and-erasure-recovery.md), [ADR-0268](0268-seo-performance-and-discovery-runtime.md)

## Context

Search crawlers need reciprocal locale alternates for translated public pages.
HydroGo publishes each translation independently, and a draft translation must
not become discoverable through another locale's metadata.

## Decision

Page HTML metadata may emit an `hreflang` link only for a page with the same
published slug and a non-null published revision, within the current tenant
scope. The URL uses the existing public HTML route. Missing locales have no
fallback link; unpublished translations never appear. The lookup is one bounded
query capped at 100 locale variants and selects only ID, locale, and slug.

## Compatibility and operations

No schema, migration, worker, cache, or public content contract changes. The
query uses the repository dialect's positional placeholders and tenant filter,
so SQLite and PostgreSQL preserve identical visibility. It adds one indexed
route lookup while rendering public page metadata; it performs no per-locale
queries. If a repository adapter does not implement the optional translation
projection, page output remains valid and omits alternates. Rollback is a code
revert; there is no stored data to recover.

## Verification

SQLite repository tests cover published-only filtering, bounded single-query
behavior, and tenant isolation. The page transport test covers escaped emitted
alternate routes and rejects a different-slug page. PostgreSQL uses the same
adapter query contract when `HYDRO_TEST_POSTGRES_URL` is configured.
