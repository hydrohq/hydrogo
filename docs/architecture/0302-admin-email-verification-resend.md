# ADR-0302: Admin Resend for Staff Email Verification

- Status: Accepted
- Date: 2026-09-29

## Context

Administrators need a safe way to resend a verification message for an
unverified staff account when mail delivery is configured.

## Decision

The staff user list shows the verification state and exposes a resend action
only for unverified users and only when the mail sender is configured. The
protected resend route is admin-only, rate-limited, audited, and creates a new
two-hour single-use verification token through the existing auth token
contract. Delivery failures return generic user-facing guidance.

No schema changes are required. Member verification uses a separate data
contract and is not modified by this staff-account action.

## Compatibility and operations

The route is registered only when the auth and mail composition enables the
existing email verification flow. SQLite, PostgreSQL, and MySQL token
contracts remain unchanged.
