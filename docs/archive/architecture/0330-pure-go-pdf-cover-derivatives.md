# ADR-0330: Pure Go PDF cover derivatives

- **Status:** Accepted
- **Date:** 2026-09-29
- **Related:** [ADR-0026](0026-bounded-media-derivatives.md), [ADR-0262](0262-pure-go-image-derivatives.md)

## Decision

The optional media optimizer may render the first page of PDF assets into a
bounded PNG derivative. Rendering uses `github.com/tucats/pdf-viewer` (MIT), a
Pure Go parser/rasterizer; it does not invoke a native executable or C library.
The source PDF remains authoritative. A failed or unsupported render marks only
the derivative as failed and never makes the source unavailable.

PDF thumbnails are limited to 10 MiB of source data, a 320-pixel maximum page
dimension, and a 20-second render deadline. Work shares the existing bounded
derivative queue. Local deployments retain the derivative beside the source;
object-storage deployments upload the derivative under its tenant-scoped key.
The admin library displays a cover only after the derivative is ready. Private
PDFs remain attachment downloads; their authorized thumbnail variant is served
as an image.

The renderer intentionally targets common PDF content and may reject or
partially omit unsupported features. This is a preview, not a faithful or
print-safe reproduction. The original file remains available for download.

## Compatibility and capacity

No schema change is needed: `media_variants` already stores typed-by-kind
derivative metadata and keys. SQLite and PostgreSQL use the same existing
repository contract. Blue/green readers ignore the new `pdf-cover` kind. The
feature adds code to the media-enabled binary only; inactive media editions do
not create a worker. Render concurrency is bounded by the existing worker and
queue budgets. The 320-pixel output bounds raster memory independently of PDF
page dimensions.

## Recovery and validation

The PDF and its source asset have independent lifecycles. Derivative failure is
visible in existing variant status and can be retried by re-enqueuing the asset;
deleting the source follows the existing variant cleanup path. Tests render a
minimal valid PDF to PNG, verify output bounds, and reject invalid PDF uploads.
