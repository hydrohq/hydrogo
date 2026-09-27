# ADR-0266: Design-system blocks and lightweight interactions

- **Status:** Accepted
- **Date:** 2026-09-27
- **Supersedes:** [ADR-0016](0016-structured-page-blocks-and-theme-slots.md)
- **Related:** [ADR-0209](0209-typed-server-ui-contract.md), [ADR-0216](0216-production-html-output-boundary.md), [ADR-0217](0217-consent-gated-content-view-analytics.md)

## Context

Client Editions need reusable public page blocks without a browser framework.
Existing page blocks are bounded JSON payloads stored in the existing
`pages.blocks`, `pages.published_blocks`, and revision fields. The public HTML
renderer and the Admin editor must share their validation contract. Existing
custom themes may not have templates for a newer block.

## Decision

The existing block contract remains capped at 64 blocks and 64 KiB per
payload. Registered payloads are validated before writes. Registered design
blocks now include `before_after`, `stats`, `contact_cta`, `video`, `masonry`,
and `hotspots`, alongside `hero`, `faq`, `cta`, `pricing_table`, and
`testimonials`. Render output uses typed Templ components from
`modules/designsystem`; `internal/ui` exposes the same typed components to
first-party views. Caller-provided text is escaped, links and media URLs are
allowlisted, and media IDs are resolved only to ready public assets.

Readers preserve an older unknown block only when its payload is a bounded JSON
object. Public rendering emits a short unavailable-content message for that
block; it never inserts its payload as markup. New writes still reject unknown
types. This allows content from an older edition to remain readable without
weakening the write boundary.

The design system uses self-hosted CSS custom properties named as Tailwind
semantic color tokens and small utility classes. Its optional JavaScript is
plain, deferred, and loaded only when an active Pages or Forms module serves
the design-system assets. Native details, range inputs, video controls, and
keyboard behavior remain available when JavaScript is disabled. Motion stops
for `prefers-reduced-motion`.

FAQ output uses native `details` and emits Schema.org `FAQPage` JSON-LD in the
document head. Contact call/WhatsApp clicks emit daily aggregate event types
only when the visitor has granted the existing analytics purpose. The event
contract stores no page URL, phone number, account, IP, or visitor identifier.

## Data and deployment compatibility

No database migration is needed: the new typed payloads use the existing
bounded JSON fields. SQLite and PostgreSQL persistence contracts are unchanged.
The existing store round-trip tests continue to cover the serialized block
envelope, and validation tests cover each registered payload. There are no new
workers, per-request browser dependencies, or mandatory resources when Pages
and Forms are inactive. The aggregate contact event uses the existing
`analytics_daily` table and its `event_type` dimension.

Blue/Green rollout rule: old records can be served by the new binary; do not
publish a new block type until all old binaries have drained. After a new block
has been written, rollback requires removing or converting those blocks before
an older binary serves the database, because that older binary does not know
the new type. No DDL or asset-manifest migration is required.

## Consequences

- Components add no third-party JavaScript or CSS runtime dependency.
- Public widgets receive the same escaping, URL checks, and accessible names
  in custom-theme fallback rendering.
- Missing or malformed legacy blocks fail safely without exposing stored JSON.
- Design assets add one bounded CSS file and one small script only when the
  Pages or Forms module is active.
- Contact analytics is opt-in and aggregated; it does not create lead records
  or identify individual visitors.
