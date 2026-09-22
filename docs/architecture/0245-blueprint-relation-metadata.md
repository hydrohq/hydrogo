# ADR-0245: Blueprint relations emit bounded resolver metadata

- **Durum:** Accepted; many-valued persistence decision superseded by [ADR-0246](0246-generated-many-relation-storage.md)
- **Tarih:** 2026-09-22
- **Kapsam:** Blueprint relation decoding, Admin form metadata and `relations.json`
- **İlgili:** ADR-0029, ADR-0230, ADR-0235

## Karar

Blueprint relation fields carry an explicit target collection and `one` or
`many` cardinality. `many` may set `max_items` from 1 to 100; omitted means 50
in generated metadata. One-valued references use a scalar reference and
inherit field requiredness. Generated forms mark the field as a relation
widget, and each resource emits sorted, versioned `relations.json` metadata.

The manifest is a bounded resolver contract. `one` currently maps to the
generated scalar field. Generated normalized storage for `many` is specified
by ADR-0246; Edition resolvers still enforce target authorization and tenant
scope. The generator does not create cascade behavior or runtime DDL.

## Consequences

- Blueprint relation metadata now reaches validation, generated UI metadata,
  and compatibility checks instead of being silently dropped.
- Unbounded many-valued relations fail before generation.
- Eager loading and target resolution remain provider/Edition responsibilities
  and cannot be mistaken for generated scalar CRUD behavior.

## Verification

Loader tests cover target/cardinality mapping, requiredness, Admin selector
metadata, generated item bounds, and rejection above the hard limit. Generated
contract tests require `relations.json` and its lock checksum.
