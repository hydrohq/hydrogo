# R5 generated bulk request adapter — 2026-09-16

Generated `bulk.ts` now mirrors the Admin Studio bulk envelope with a typed
request builder, bounded selection, duplicate rejection, safe action validation,
and deterministic ID ordering. It has no runtime dependency or credentials.

Verification:

```text
go test ./internal/generator -count=1
PASS
```
