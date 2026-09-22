# ADR-0241: Blueprint input has an explicit version boundary

- **Durum:** Accepted
- **Tarih:** 2026-09-22
- **Kapsam:** YAML/JSON Blueprint loader and authoring schema
- **İlgili:** ADR-0210, ADR-0219, ADR-0224, ADR-0240

## Karar

Blueprint files may declare `version: v0`; when omitted, the loader preserves
backward compatibility by treating the input as v0. Any other version is
rejected before a `Spec` is built or any generated artifact is written.

The JSON Schema publishes the same allowlist. A future Blueprint version must
be introduced deliberately with a migration/compatibility decision rather than
being inferred from newly added fields.

## Consequences

- A typo or future-version file cannot silently produce an incompatible
  migration, API, or Admin Studio surface.
- Existing Blueprints remain valid and deterministic.
- Version evolution is explicit and can carry schema-diff and rollback
  metadata without changing the meaning of an old Blueprint.

## Verification

Loader tests cover omitted/explicit `v0` and fail-closed rejection of `v1`.
`docs/schemas/blueprint.schema.json` exposes only `v0` until a new ADR and
compatibility implementation are accepted.
