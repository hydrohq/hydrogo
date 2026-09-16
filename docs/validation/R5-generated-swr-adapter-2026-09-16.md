# R5 generated SWR adapter — 2026-09-16

`go test ./internal/generator -count=1` PASS.

Every generated resource now contains `swr.ts` with typed keys and fetcher
wiring. The compatibility checker requires the artifact and generator smoke
continues to pass.

Actual SWR browser integration, auth, and cache invalidation remain Client
Edition acceptance gates.
