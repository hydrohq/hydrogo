# R5 generated kit dependency artifacts — 2026-09-16

Generated admin metadata, `kits.ts`, and `RUNTIME_WIRING.md` now include the
deterministically resolved kit set. A Commerce resource consequently exposes
both `commerce` and its required `forms` dependency. Unknown and duplicate
kits remain fail-closed, and no kit implementation is loaded by generation.

Verification:

```text
go test ./internal/generator ./internal/adminstudio -count=1
PASS
```
