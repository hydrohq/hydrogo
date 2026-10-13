# R46 — Generated client stream lock release cleanup

**Date:** 2026-09-17  
**Status:** Successful (local generator contract)

The bounded response reader in generated `client/hydrogo.ts` now guarantees calling `reader.releaseLock()` in a `finally` block on standard completion, oversize errors, and parse/transport exceptions. The same stream lifecycle contract as headless starter is maintained; response byte limits and reader cancel behavior remain unchanged.

The `internal/projectgen` test verifies the `releaseLock` marker and existing published-client contract in generated sources.
