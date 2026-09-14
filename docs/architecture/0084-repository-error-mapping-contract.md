# ADR-0084: Stable repository error mapping across SQL dialects

## Status

Accepted

## Context

Repository callers must not depend on driver-specific SQL error text. A
duplicate member registration can race the service pre-check and reach the
database, where SQLite and PostgreSQL report different errors.

## Decision

The members SQL repository maps the database uniqueness condition to the
stable `members.ErrMemberExists` contract. PostgreSQL SQLSTATE `23505` and
the equivalent SQLite uniqueness messages are recognized only at the
repository boundary; domain and transport packages remain dialect-agnostic.
Unrelated driver failures are returned unchanged.

## Consequences

Concurrent duplicate registrations produce the same safe application error on
both supported dialects. New adapters must preserve this contract and add
their driver-specific recognition in the adapter layer. Real PostgreSQL
execution remains covered by the external database gate.

## Verification

See `modules/members/sqlstore/error_mapping_test.go`,
`modules/members/sqlstore/store_test.go`, and
`docs/validation/R5-repository-error-mapping.md`.
