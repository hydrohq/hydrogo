# Dynamic Collections Admin pagination — 2026-10-08

## Behavior

The Collections Admin record page now reads a bounded `offset` query parameter, requests 51 rows from the existing limit/offset service contract, renders at most 50, and exposes accessible Previous/Next links. The extra row determines whether a next page exists without a total-count query. Malformed, negative, non-page-aligned, and out-of-range offsets return HTTP 400. No route, schema, migration, permission, or database contract changed.

## Verification

`internal/ui`, `internal/collections/transport`, and `internal/collections` Go tests passed. Chromium E2E created 51 records against a fresh generated SQLite Edition and verified 50 rows on page 1, one row on page 2, previous/next navigation, disabled-state semantics, invalid offset rejection (malformed, negative, unaligned, and out-of-range), and axe WCAG 2.1 A/AA on both pages. The full dynamic Collections browser journey passed 1/1. Desktop/mobile captures were inspected; the Impeccable detector returned no findings.

```sh
TMPDIR=/var/tmp/hydro-go-tests go test ./internal/ui ./internal/collections/transport ./internal/collections -count=1 -p 1 -timeout=5m
cd e2e && npx playwright test custom/dynamic-collections.spec.ts --project chromium --workers=1 --reporter=line --output=/tmp/hydro-collections-pagination-axe
```

## Capacity and storage

Each request reads at most 51 records, below the existing service maximum of 100. Pagination uses the existing database adapter and tenant scope; it adds no count query, cache, worker, migration, or retained state.
