# ADR-0118: Expose standalone frontend manifest validation

- **Status:** Accepted
- **Date:** 2026-09-16

Frontend build manifests are a contract input, not merely a side effect of
project generation. The CLI therefore exposes `hydro frontend check
--manifest PATH [--json]`, reusing the same fail-closed normalization used by
project generation. It reports stable route ordering and rejects unsafe,
duplicate or missing route fields without creating files or contacting a
runtime service.
