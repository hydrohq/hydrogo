# ADR-0261: Apply Blueprint defaults only to omitted create fields

- **Status:** Accepted
- **Date:** 2026-09-23
- **Related:** ADR-0226, ADR-0257, ADR-0259, ADR-0260

## Context

Blueprint defaults need to distinguish an omitted input from an explicitly
submitted zero value, `false`, or empty string. Decoding directly into a Go
struct loses that distinction and can silently overwrite intentional client
input. Defaults must also be checked against the declared field type and its
existing constraints before generation.

## Decision

Blueprint validation defaults are optional string values in the source
contract, preserving the distinction between absent and explicitly empty
values. The generator validates supported scalar defaults at generation time
against field type, length, regex, enum options, format, and numeric bounds.
Unsupported or invalid defaults reject the Blueprint.

Generated create transports decode bounded JSON objects field-by-field,
reject unknown keys and trailing JSON, and record which fields were present.
Generated services apply defaults only to omitted fields before validation and
persistence. Explicit `0`, `false`, and empty values are preserved and remain
subject to the field's normal validation. Tenant identity remains resolver-owned
and is never accepted as an input field. Update behavior is unchanged.

## Consequences

- Defaults are deterministic and do not require schema changes or runtime DDL.
- Presence tracking adds a bounded map per create request; it adds no worker,
  route, or persistent resource to the Compact profile.
- Create transports intentionally retain strict unknown-field rejection.
- This does not introduce database-level defaults; migrations and database
  defaults remain operator-reviewed Edition responsibilities.

## Verification

Generator fixtures cover typed OpenAPI/form defaults, malformed Blueprint
defaults, and generated create behavior for omitted versus explicitly
submitted integer, boolean, and nullable string values. Generated JSON decode
remains bounded to 1 MiB and uses standard-library streaming decode. Database
schema behavior is unchanged; live PostgreSQL deployment certification is not
claimed by this decision.
