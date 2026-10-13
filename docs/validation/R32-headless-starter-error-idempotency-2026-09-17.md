# R32 — Headless starter error and idempotency parity

**Date:** 2026-09-17  
**Status:** Successful (local TypeScript contract)

## Changes

The standalone `examples/headless-starter` client was brought into parity with the generated client's baseline security and retry contract:

- checkout idempotency key is validated fail-closed with 16–128 visible ASCII characters;
- API errors are now carried with typed `HydroAPIErrorBody` instead of raw text;
- `code`, user-appropriate `message`, and `x-request-id` fields are preserved;
- known `Content-Length` exceeding 2 MiB is rejected without reading the body;
- chunked/streaming responses are also read under a 2 MiB byte limit and the reader is aborted if the limit is exceeded;
- timeout, abort, and existing error body limits are preserved.

## Verification

```powershell
npm run check
npm run build
```

Result: **PASS**. `tsc --noEmit` and production TypeScript build succeeded.

## Scope & Boundaries

This evidence does not constitute real browser/CORS/preview-token or payment provider acceptance. Those remain Client Edition and production environment gates.
