# ADR-0245: Blueprint relations emit bounded resolver metadata

- **Durum:** Accepted
- **Tarih:** 2026-09-22
- **Kapsam:** Blueprint relation decoding, Admin form metadata and `relations.json`
- **İlgili:** ADR-0029, ADR-0230, ADR-0235

## Karar

Blueprint relation fields carry an explicit target collection and `one` or
`many` cardinality. `many` may set `max_items` from 1 to 100; omitted means 50
in generated metadata. One-valued references use a scalar reference and
inherit field requiredness. Generated forms mark the field as a relation
widget, and each resource emits sorted, versioned `relations.json` metadata.

The manifest is a bounded resolver contract, not a promise that generated CRUD
persists many-valued relations. `one` currently maps to the generated scalar
field. `many` remains Edition-managed until a normalized relation-table
contract is implemented and certified for SQLite/PostgreSQL. Edition resolvers
must enforce target authorization, tenant scope, and item bounds; the generator
does not create joins, cascade behavior, or runtime DDL.

## Consequences

- Blueprint relation metadata now reaches validation, generated UI metadata,
  and compatibility checks instead of being silently dropped.
- Unbounded many-valued relations fail before generation.
- Relation persistence and eager loading remain explicit future capability
  work and cannot be mistaken for generated scalar CRUD behavior.

## Verification

Loader tests cover target/cardinality mapping, requiredness, Admin selector
metadata, generated item bounds, and rejection above the hard limit. Generated
contract tests require `relations.json` and its lock checksum.
