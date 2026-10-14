# ADR-0356: Revision-bound News draft preview

- **Status:** Accepted
- **Date:** 2026-10-08
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0013](0013-identity-and-security-boundaries.md), [ADR-0014](0014-publication-routes-and-erasure-recovery.md), [ADR-0346](0346-blueprint-admin-studio-workflows.md)

## Context

The News editor's preview action currently returns the ordinary public article
URL. Draft and scheduled articles are deliberately absent from public reads, so
the editor's preview frame receives a 404. News needs the same revision-bound,
short-lived preview behavior already used by Pages without changing published
route, sitemap, or cache behavior.

## Decision

- The protected `POST /admin/news/{id}/preview` endpoint issues a signed token
  only for a non-published News item. The token binds `resource_type=news`, item
  ID, current revision, optional tenant ID, and expiry. Default lifetime is 15
  minutes.
- The endpoint returns the existing public article path with a
  `preview_token` query parameter. `GET /public/news/{locale}/{slug}/html`
  accepts this query only when the signer validates the token, the item is
  still non-published, the revision and tenant still match, and locale and slug
  resolve to the same item.
- Published articles keep the ordinary public URL. Missing, expired, altered,
  wrong-resource, wrong-tenant, changed-revision, or newly published tokens
  return 404 without revealing which check failed.
- Preview responses set `Cache-Control: private, no-store`,
  `X-Robots-Tag: noindex, nofollow`, and `Referrer-Policy: no-referrer`.
  Query-bearing public requests already bypass process and static HTML caches;
  content-view analytics also skips token previews.
- The token-signing key is shared across Blue/Green slots. An old slot that
  does not understand `preview_token` returns the usual unpublished 404; it
  never exposes draft content. Preview links are short lived and may be
  regenerated after a rollout.
- No SQL, migration, worker, or background service is added. The shared token
  signer remains optional; draft preview is unavailable when no signing key is
  configured, and published preview continues to use the public URL.

## Alternatives

- Keep the ordinary public URL: rejected because an unpublished draft cannot
  render there.
- Add a separate `/preview/news/{id}` route: rejected because the existing
  locale/slug renderer should remain the only article presentation path; the
  signed query naturally bypasses public caches.
- Store preview tokens in the database: rejected because stateless revision
  binding needs no new table, cleanup worker, or state synchronization.

## Consequences

### Benefits

- Editors can review the actual News theme and article layout before publish.
- Saving a new revision or publishing/retracting an article invalidates an old
  preview token without a separate revocation table.
- Existing public article paths and SQL adapters do not change.

### Costs and risks

- The configured signing key must be at least 32 bytes and identical across
  active deployment slots.
- A copied preview URL can be used by anyone who receives it until expiry or a
  revision/status change; no-store, no-referrer, and the short lifetime limit
  that exposure.

## Scale impact

- **Compact:** one bounded HMAC verification and existing item lookup per
  preview; no cache, worker, or persistent token state is created.
- **Standard:** uses the existing DB pool and token signer; query previews
  bypass shared HTML caching.
- **Distributed:** all slots share the configured signing key; database
  revision/status checks make tokens consistent across instances.

## Database impact

- **SQLite:** uses the existing News item lookup; contract tests verify the
  draft, revision, status, and public-route behavior.
- **PostgreSQL:** same contract tests run against an isolated schema when
  `HYDRO_TEST_POSTGRES_URL` is configured.
- **Future adapters:** implementations must provide the existing `Get` and
  published-route semantics; no dialect-specific query is added.

No migration is required. Existing SQLite/PostgreSQL migration metadata,
rollback instructions, and migrationguard outputs are unchanged because no
schema or persisted token is introduced.

## Cache impact

The query string bypasses the process public-HTML cache and static disk cache.
The response is `private, no-store`; no preview HTML, token, or tokenized URL is
stored. No cache key or invalidation path changes.

## Deployment and compatibility

Configuration and schema remain backward compatible. New binaries accept
revision-bound tokens; old binaries continue to serve published articles and
return 404 for draft URLs. Rollback requires no data change. Blue/Green slots
must share the preview signing key for links to work across slots; all slots
remain fail-closed for draft reads.

## Validation

- Unit tests prove resource binding, expiry, and revision/status invalidation.
- SQLite and PostgreSQL News contract tests prove a draft is hidden without a
  token, rendered with a valid token, hidden after its revision changes, and
  never cached as public HTML.
- Chromium Admin acceptance opens a real draft preview, checks the article
  content and private/no-store headers, then exercises schedule, publish,
  retract, and automatic scheduled publication.
- `go test ./... -count=1 -p 1 -timeout=5m`, `go vet ./...`, moduleguard,
  migrationguard, and `git diff --check` passed. The News preview contract
  passed independently on SQLite and PostgreSQL 18.
- Chromium `admin-management.spec.ts` passed 5/5, including the draft preview
  dialog, rendered draft content, private response headers, mobile sizing,
  editorial permissions, live scheduled publication, responsive layouts, and
  accessibility checks across themes and color modes.

## References

- [ADR-0014: Publication routes and safe recovery](0014-publication-routes-and-erasure-recovery.md)
- [ADR-0346: Blueprint Admin Studio and editorial workflows](0346-blueprint-admin-studio-workflows.md)
- `internal/preview/token.go`
- `modules/pages/transport/http.go` (`createPreviewLink`, `publicDraftPreview`)
- `modules/news/transport/http.go` (`adminPreview`, `publicGetHTML`)
