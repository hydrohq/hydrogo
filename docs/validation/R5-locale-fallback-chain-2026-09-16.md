# R5 locale fallback chain — 2026-09-16

Added a bounded deterministic locale fallback contract. Exact locale, primary
language, and supported default are selected in order; duplicate and unsafe
configuration fails closed.

Verification:

```text
go test ./internal/i18n -count=1
PASS
```
