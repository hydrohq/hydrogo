# R5 typed frontend query adapter — 2026-09-16

## Scope

The resource generator now emits typed, framework-neutral query options and
stable query-key helpers in `react.ts`.

## Command

```text
go test ./internal/generator -count=1
```

## Result

PASS. The generated fixture contains `QueryOptions`, `create<Name>Query`, and
`<Name>QueryKey`, and the existing generated Go fixture compilation remains
green.

## Limits

This validates generation and compilation only. React Query/SWR browser
integration, authenticated cache invalidation, and production bundle budgets
remain Client Edition/release gates.
