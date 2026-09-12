# ADR-0026: Bounded media derivatives

## Decision

HydroGo keeps the uploaded original immutable and creates optional WebP image and WebM video derivatives in `internal/media`. Derivative work runs asynchronously through a bounded in-process worker. The original upload is never blocked by a missing encoder or a full derivative queue.

The encoder is an optional external `ffmpeg` process, configured with `HYDRO_MEDIA_FFMPEG`. This keeps Compact free of a permanently resident codec service and allows deployments to choose a packaged/system encoder. If it is unavailable, the original remains usable and the derivative is recorded as failed with a bounded diagnostic.

## Resource policy

- Upload bytes are streamed and capped by the existing media budget.
- Derivative concurrency is bounded by the capacity profile and defaults to one in Compact.
- Each job has a bounded queue, timeout, output-size limit, and ffmpeg is restricted to one codec thread.
- Source files are never loaded into an unbounded byte slice.
- `media_variants` is the status and lookup contract; original `media_assets` metadata is unchanged.

## Consequences

Public media requests may ask for `?variant=card` or `?variant=video`; they fall back to the original while a derivative is processing or failed. RabbitMQ/BullMQ is intentionally not introduced: the existing bounded job model remains the platform boundary, and a durable media-job migration can be added when cross-process processing is actually required.
