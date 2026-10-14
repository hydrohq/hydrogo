# R38 — Headless starter smoke gate added to local verification hook

**Date:** 2026-09-17  
**Status:** Successful (local Windows)

`scripts/verify-local.ps1` appends `examples/headless-starter/npm test` to the full Go verification flow whenever `npm` is present on the machine. In minimal environments without Node/npm, Go verification does not fail; a controlled warning is emitted. The CI browser-e2e job installs Node and runs this check mandatorily.

Verified: TypeScript build, idempotency key, typed API error/request-id, 2 MiB response limit, timeout configuration and abort behavior.
