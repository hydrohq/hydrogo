# ADR-0012: Content, media and data portability boundaries

- **Status:** Modified (partially) — [ADR-0014](0014-publication-routes-and-erasure-recovery.md) for slug/301 activation time; other provisions remain valid.
- **Date:** 2026-09-05
- **Related decisions:** [ADR-0006](0006-extensible-database-adapters.md), [ADR-0009](0009-site-edition-and-delivery-scope.md), [ADR-0011](0011-consistency-upgrade-and-recovery.md)
- **Implementation status:** Model principles accepted; exact table/Go API design will be validated via prototype.

## Decision and alternatives

The initial modules use typed models, relational tables, and module repositories. A universal content engine moving everything to a single EAV/JSON table is not built. JSON is only used in appropriate fields like versioned rich text/blocks; business fields to be filtered/unique require explicit columns/indexes and dialect migrations. The future schema generator will also produce this model.

This choice requires writing more queries/migrations in some modules; it ensures behavior and indexes are explicit. DB-specific JSON/full-text capabilities do not become a domain contract.

## Content identity and publication

- The opaque identity is generated on the application side; the initial common identity format is UUIDv4. It is separate from the public URL slug. UUID does not provide access authorization to the existence of the object.
- Time is UTC, normalized with microsecond precision in persistence/API comparisons; the scheduler explicitly shows local time and DST interpretation in the UI.
- Logical content ID, locale variant, and revision ID are separate. The edit version is an optimistic-lock token; it is not confused with the published revision ID.
- A draft save does not change the live revision. Publish is atomic with the revision pointer/audit/outbox. A scheduled job is tied to a static revision and schedule generation; an old canceled job cannot publish a new draft. During execution, state and permission policies are re-verified.
- Translation variants are related but published independently from each other. The default public behavior for missing translation is a 404; fallback is only enabled by an explicit edition policy, correct canonical, and cache key. There is no fallback to draft.
- Route identity is site + locale + normalized path. The new slug/301 record becomes active within the publish transaction, with reserved path and loop checks; draft saving does not alter the live route or its SEO projection. Details are in ADR-0014. Route normalization must be identical in both DBs using Unicode, case, trailing slash, and percent-encoding fixtures; the algorithm is fixed at the prototype gate.
- API/UI reading is a published and authorized projection; the DB row is not directly serialized. There are field-based read/write allowlists, public/private settings separation, and relation filters.

## Rich text

- The initial editor candidate is Quill; Delta input validation and sanitizer rules are preserved. The stored body carries a `format`, `format_version`, `document` envelope; for example, `quill-delta` v1. This alone is not a guarantee of lossless conversion among all editors.
- The renderer/sanitizer version is in the derived HTML metadata. If the editor changes, the old document renderer remains throughout the support window; the converter reports potentially lost elements, and the original document is preserved. Format change enters the release compatibility check. An unknown format is not silently converted to empty HTML causing data loss.

## Media

- Media ID and storage key are separate. Revision references use a static Media ID; a permanent provider URL is not written into the content.
- Local adapter in the first slice; S3-compatible adapter implemented in Standard/Distributed delivery. An endpoint resembling S3 is not automatic certification: multipart, range, metadata, signature, and error behaviors are tested.
- A single transaction between DB and object store is not assumed. Upload first creates a `pending` record and a temporary/quarantine object. Once validation/transform is complete, the final immutable key is written; then `ready` and references are committed in the DB. Erroneous/orphaned objects are cleaned up with a wait time; a non-existent object is not declared ready for public.
- There is a public/private class and reference authorization. Private objects cannot be leaked via public bucket/origin; signed URLs are time-limited and generated after access control. Hash name is not authorization.
- Making media uploaded in a draft public is a separate explicit transition. When content is unpublished, all copies of a previously public file cannot be recalled; media visibility and other content sharing it are considered.
- The variant key contains source digest + transform spec + encoder version. Referenced revision, pending jobs, and backup pins prevent GC deletion. Transform concurrency, byte/pixel, and disk limits are bounded.
- The initial format scope remains JPEG/PNG/WebP; encoder selection and Windows/Linux build suitability will be validated with a technical spike. SVG, remote URL fetch, and executable files are not enabled initially.

## Search and export

Search is not the Core content truth record; it is derived from the module's allowed public projection. The initial DB query offers limited filter/sort. External index can be rebuilt; hits are re-verified by the service for authorization and current publication state. The same relevance ranking is not promised across the two DBs; the guaranteed filter/pagination semantics are defined separately.

The DB-independent Hydro Export consists of a versioned manifest + type-based NDJSON + referenced media checksums. Identities, locale relations, revisions, routes, and module schema versions are preserved. Password hashes, sessions, secrets, license keys, and job/outbox execution states do not enter the default export; identity migration is a separate secure transition.

Export is not confused with production restore. Import first validates in an empty staging target, establishes references in two passes, reports unknown modules/formats, and generates a measurement report. Automatic merge into a populated DB or seamless transition between DBs is not a v1 promise. The portable exit path is proven with a round-trip fixture before the first public v1; performant bulk migration can be a subsequent development.

## Impact and evidence

SQLite and PostgreSQL pass the identity/time/locale/slug/unique/revision/relation tests together. A new adapter must provide the same semantics. Local media is sufficient for Compact; Distributed requires common object storage. The cache observes the revision+locale+visibility boundary, bypassing admin/private media.

Format and schema changes enter the migration metadata. In blue/green, if the old reader cannot read the newly written content/media format, an online release cannot be declared. Session, asset, job, and backup continuity are subject to ADR-0011.

Fixtures: two editor conflict, draft changes not leaking to live page, missing translation, schedule cancellation, old slug loop, field authorization, private media URL guessing, crash between object write and DB commit, protecting referenced object from GC, export/import between two DBs, and rendering of supported old content format.
