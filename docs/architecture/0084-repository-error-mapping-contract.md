# ADR-0084: Stable repository error mapping across SQL dialects

## Status

Accepted

## Context

Repository callers must not depend on driver-specific SQL error text. A
duplicate member registration can race the service pre-check and reach the
database, where SQLite and PostgreSQL report different errors.

## Decision

The adapter-facing `internal/database.IsUniqueViolation` helper recognizes
PostgreSQL SQLSTATE `23505` and equivalent SQLite uniqueness messages. The
members SQL repository maps that condition to the stable
`members.ErrMemberExists` contract, while pages, commerce, and the durable
job store reuse the same helper for their own conflict semantics. Domain and
transport packages remain dialect-agnostic; unrelated driver failures are
returned unchanged.

## Consequences

Concurrent duplicate registrations produce the same safe application error on
both supported dialects, and conflict handling no longer drifts between
repositories. New adapters must preserve this contract and add their
driver-specific recognition in the adapter layer. Real PostgreSQL execution
remains covered by the external database gate.

## Verification

See `modules/members/sqlstore/error_mapping_test.go`,
`modules/members/sqlstore/store_test.go`, and
`docs/validation/R5-repository-error-mapping.md`.
