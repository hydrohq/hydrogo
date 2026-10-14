# ADR-0262: Pure Go image derivatives

- **Status:** Accepted
- **Date:** 2026-09-27
- **Supersedes:** Image encoding portion of [ADR-0026](0026-bounded-media-derivatives.md)

## Context

The optional external encoder made image derivatives unavailable on shared
hosting without FFmpeg. Image derivative work already runs on a bounded,
profile-sized worker queue, so in-process encoding can preserve the original
upload and queue contracts without adding a resident service.

## Decision

JPEG, PNG, GIF, and WebP images are decoded, resized to at most 1920 pixels
wide, and encoded as WebP in Go. Decoding rejects images above 40 million
pixels. Output is written to a temporary file and renamed only after encoding
succeeds. The original remains immutable. The existing optional FFmpeg path is
retained only for video derivatives and is never required for image uploads.

The existing `media_variants` state and keys remain unchanged. There is no
schema migration. Editions without the media module do not start derivative
workers or acquire the WebP package at runtime beyond the binary's normal code
footprint.

## Consequences

- Image derivatives work in self-contained CGI and binary deployments without
  a system encoder.
- The pure Go WebP encoder is included in builds that include media support.
- The 40-million-pixel guard bounds decoded image memory; worker concurrency
  remains constrained by the selected capacity profile.
- Video derivatives still require an explicitly configured FFmpeg executable.
- The storage key, derivative MIME type, and fallback-to-original behavior are
  compatible with previous releases; no migration or deployment fence is
  required.

## Verification

- SQLite and PostgreSQL share the unchanged `media_variants` repository
  contract; database contract tests remain applicable.
- Pure Go image encode/decode round-trip tests cover the new path.
- The worker queue remains bounded at eight queued assets, with concurrency
  clamped to four and to the runtime capacity profile.
