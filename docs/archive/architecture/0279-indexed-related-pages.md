# ADR-0279: Indexed related-page recommendations

- **Status:** Accepted
- **Date:** 2026-09-28
- **Related:** [ADR-0278](0278-indexed-related-content.md), [ADR-0014](0014-publication-routes-and-erasure-recovery.md)

## Context

Pages need tag-based recommendations without making draft edits affect the
currently published page. Their content, publication state, and revision
history already have separate repository contracts.

## Decision

Page tags are bounded, normalized inputs in the Page domain model. The Pages
module stores current tags in `page_tags` and copies them into
`page_published_tags` only during publication. Draft creation/update, publish,
and the relation changes share the existing page transaction. Public lookup
joins the published-tag table, excludes the current page and anything whose
current state is not published, filters by locale and tenant, ranks shared
tags, and caps results at ten. Page detail rendering requests four summaries
and emits escaped, responsive related-page cards. This migration is owned by
the Pages module and creates no schema when Pages is inactive.

## Verification and operational impact

SQLite contract tests cover tag snapshots, draft isolation, ranking, locale
and tenant filtering, and index selection. PostgreSQL DDL applies tenant RLS
to both relation tables and has a matching opt-in contract test. Storage is
bounded to 50 normalized tags per page in each relation. Each lookup uses the
`(tenant_id, tag, page_id)` index and the four-card public limit; no workers or
cache tiers are introduced. Old page revisions remain readable because tag
relations are additive and are not required to deserialize prior snapshots.

Rollback is forward-only by default. Back up the Pages module before dropping
the current and published tag tables; page body and revision data remain
untouched.
