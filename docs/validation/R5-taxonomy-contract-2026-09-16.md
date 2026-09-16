# R5 taxonomy contract — 2026-09-16

## Scope

Added a bounded provider-neutral taxonomy contract for localized categories,
tags, parent relationships, and content-to-term relations.

## Command

```text
go test ./internal/taxonomy -count=1
```

## Result

PASS. Duplicate IDs, unknown parents, parent cycles, unsafe slugs, duplicate
relations, unknown term references, and deterministic ordering are covered.

## Limits

No module persistence, migration, admin UI, or search indexing is claimed by
this contract-only phase. Those require product acceptance and certified
SQLite/PostgreSQL behavior in the adopting module.
