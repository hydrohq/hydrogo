# ADR-0016: Structured page blocks and theme slots

- **Status:** Superseded by [ADR-0266](0266-design-system-blocks-and-interactions.md)
- **Date:** 2026-09-10
- **Related decisions:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0012](0012-content-media-and-portability.md), [ADR-0015](0015-versioned-http-api.md)
- **Implementation status:** Block contract, SQLite/PostgreSQL migrations, store round-trip validation, admin visual editor, and safe renderer have been implemented.

## Decision

Pages can carry an ordered and versioned list of `Block`s in addition to the rich text body. Block payloads are stored as JSON; however, the runtime only accepts declared block types. The contract covers `hero`, `faq`, `cta`, `pricing_table`, and `testimonials` types.

- A maximum of 64 blocks and 64 KiB payload per block are accepted.
- The payload must be a JSON object and must pass type-based mandatory fields.
- `javascript:` and `data:` schemes are rejected in URL fields.
- Draft, immutable revision, and published projection carry the same block list within the transaction.
- A new block type does not require a DB migration; but it requires a versioned contract, renderer, and export/import testing.
- An unknown block type is rejected fail-closed; it is not silently dropped with empty content.

The theme system does not break the existing `Theme` contract. Themes supporting blocks implement the optional `BlockTheme` contract. Theme selection is done via build/edition composition; arbitrary Go code or untrusted themes are not loaded at runtime.

## Data and compatibility

The `pages.blocks`, `pages.published_blocks`, and `page_revisions.blocks` fields are stored as bounded TEXT in SQLite and JSONB in PostgreSQL. The domain and service layers do not know SQL dialects. Legacy records are read with an empty block list. A format change requires a new content format/version and an expand-migrate-contract transition.

Public and headless outputs use the same published projection and authorization policy. The block renderer receives a safely validated model, not the raw DB record. Media IDs are resolved only via the media service; permanent provider URLs are not written into the payload.

## Next gates

Completed by admin block editor, field-based authorization and sanitizer behavior for each block; HTML renderer; JSON/OpenAPI examples; two dialect migration contracts; export/import; theme asset inventory and blue/green retained asset testing.
