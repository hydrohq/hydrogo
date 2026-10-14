# ADR-0303: Admin-Generated Password Reset Links

- Status: Accepted
- Date: 2026-09-29

## Context

Administrators sometimes need to share a password recovery link directly when
mail delivery is unavailable or unsuitable.

## Decision

An active administrator can request a single-use reset link for an active
staff account. The endpoint is protected, rate-limited, audited, returns a
`no-store` response, and uses HydroGo's existing random token hash and reset
contract with a one-hour expiry. The user list offers one-click copy. If the
clipboard API is unavailable, the link remains available in an accessible
status output. The public reset form posts the token in the request body and
removes it from browser history after loading.

## Compatibility and operations

No schema changes are required. The feature is available only when recovery
storage and a canonical public origin are configured. SQLite, PostgreSQL, and
MySQL use the existing auth recovery token contract.
