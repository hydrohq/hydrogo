# R5 generated frontend index — 2026-09-16

Each generated resource now includes a dependency-free `index.ts` namespace
barrel for all generated API and presentation adapters. This reduces import
friction without coupling Core to React, SWR, HTMX, or another design system.

Verification:

```text
go test ./internal/generator -count=1
PASS
```
