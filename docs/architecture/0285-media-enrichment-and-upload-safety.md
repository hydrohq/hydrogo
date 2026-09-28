# ADR-0285: Pure Go media enrichment and upload safety

- **Status:** Accepted
- **Date:** 2026-09-28
- **Related decisions:** [ADR-0012](0012-content-media-and-portability.md), [ADR-0026](0026-bounded-media-derivatives.md), [ADR-0262](0262-pure-go-image-derivatives.md), [ADR-0110](0110-bounded-private-media-urls.md), [ADR-0111](0111-protected-media-url-endpoint.md)

## Decision

Image metadata generation uses Go image decoding and a fixed 32×32 sample grid to bound CPU cost. BlurHash and dominant colors are stored with the media asset. Focal coordinates use normalized values with a centered default; crop and watermark operations return new images and do not modify immutable masters. SEO display filenames are normalized independently from content-addressed storage keys.

Virtual folders and tags are editor metadata. They never affect object keys or authorization. Folder and asset records carry tenant scope. Private media remains governed by the existing authorization and signed URL boundary. Referer filtering is an optional edge guard and treats an empty Referer as a direct visit; it is not an authorization mechanism.

Admin Studio media controls persist as one private, tenant-scoped value in the existing `site_settings` store, so they require no additional schema or worker. Hotlink settings are cached per tenant in-process and take effect on the next media request. Watermark options (PNG logo asset, text, opacity, placement, margin, and selected folders) are captured when an upload is promoted to ready; generated public derivatives receive the overlay while the source master remains unchanged. A watermarked public image is served from its derivative; while generation is pending, the response asks the client to retry instead of exposing the master. Editors retain the existing authorized master access.

Documents are bounded and validated by extension plus file signature before protected storage. PDF previews are not rendered by the server. Video embeds accept only HTTPS YouTube/Vimeo identifiers and use provider embed hosts; rendering adds no JavaScript.

## Compatibility and operations

`202609290001_media_enrichment` is additive. Old binaries ignore the extra columns; the new media store needs the migration before use. There is no new worker, service, or runtime executable. The Compact profile pays only bounded image sampling on image upload. Folder indexes remain local DB metadata. No cache is introduced.

SQLite and PostgreSQL use the same repository contract; MySQL migration parity is kept because the current repository certifies that adapter. Rollback is forward-only because removing the metadata would lose editor choices. Blue/green deployment remains compatible while old code ignores additive columns; media metadata writes begin only after migration.

## Validation

Pure Go unit tests cover hash output, focal cropping, watermarking, filename normalization, document signature rejection, host allowlisting, and embed URL validation. SQLite migration tests cover the additive schema. The media lifecycle contract also covers folder metadata and tenant-scoped Admin Studio settings on SQLite and PostgreSQL.

## Limits

BlurHash is computed from a fixed sample grid; it is not a perceptual quality or performance guarantee. Referer headers can be omitted or forged by non-browser clients. Document signature checks do not certify that a document is malware-free; private authorization and operational scanning policy remain separate controls.
