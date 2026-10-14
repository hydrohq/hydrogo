# R44 — Generated client parity with headless starter

**Date:** 2026-09-17  
**Status:** Successful (local generator contract)

`client/hydrogo.ts` in the project generator output now carries the same baseline boundaries as the headless starter:

- `baseUrl` must be absolute HTTP(S); embedded credentials and unsupported schemes are rejected pre-request;
- `getPage` and `listNews` accept per-call `{ signal }`;
- mutations preserve the existing `{ idempotencyKey, signal }` contract.

The `internal/projectgen` test verifies these markers and published API routes in generated sources. Real framework/browser acceptance remains gated on Client Edition and production auth topology.
