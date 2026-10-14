# ADR-0260: Enforce generated immutable fields atomically

- **Status:** Accepted
- **Date:** 2026-09-23
- **Related:** ADR-0226, ADR-0257, ADR-0258, ADR-0259

## Context

Blueprint `validation.immutable` was previously metadata only. A service-side
read/compare followed by an unconditional update is vulnerable to concurrent
changes: the value can change between the comparison and write. The generated
resource model also does not preserve nullable field presence, and many-valued
relations are persisted outside the resource row.

## Decision

For each scalar field marked `validation.immutable`, generated SQLite and
PostgreSQL repository updates include the submitted immutable value in the
same `UPDATE ... WHERE` predicate as the record identity (and tenant identity
when applicable). This is an atomic compare-and-write boundary: a caller can
update other fields only when the immutable value still matches the stored
value. An immutable-aware update affecting zero rows returns the generated
`ErrResourceConflict` sentinel; the generated transport maps that sentinel to
the safe HTTP 409 contract in ADR-0259.

The generator rejects immutable nullable fields and many-valued relations.
Their presence/null and relation-set baselines cannot be represented by the
current scalar CRUD model. No schema migration, runtime DDL, service, or worker
is added. Repository implementations remain reviewable generated artifacts.

## Consequences

- Immutable values are enforced by one atomic write, avoiding a read/write TOCTOU
  window without adding a row version or migration.
- The check is only as strong as the repository boundary: direct SQL writes
  that bypass the generated repository are outside this guarantee.
- A missing row and a mismatched immutable value both produce a conflict from
  this primitive; an Edition may perform an authorized scoped read when it
  needs a distinct not-found response.
- Nullable immutable values and immutable many-relations remain unsupported
  until field presence and relation transaction semantics are versioned.

## Verification

Generator integration fixtures compile and execute the emitted SQLite
repositories for both tenant-scoped and global resources. They prove changed
immutable values are rejected without changing any column and unrelated fields
remain updateable. A validation test rejects unsupported declarations. The
generated SQL uses placeholders and portable equality/AND syntax; a live
PostgreSQL contract run remains a release gate.
