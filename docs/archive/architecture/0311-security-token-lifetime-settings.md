# ADR-0311: Security token lifetime settings

## Status

Accepted

## Decision

Password reset and invitation lifetimes are configurable from the protected
Admin Studio security page. The accepted choices are 15 minutes, 1 hour, and
24 hours for password reset links, and 1, 3, and 7 days for invitations. A
singleton policy row is shared by all tenants and application instances so a
setting change immediately affects subsequently issued links everywhere.
Existing links retain their original expiry; reset and invitation stores reject
them after that persisted timestamp.

When no policy row exists, bounded operator configuration supplies defaults.
The table stores only the two numeric durations and update time. SQLite,
PostgreSQL, and MySQL share the same constraints and upsert contract. No worker,
cache, or additional process state is introduced. Rollback drops the policy row
and returns to configured defaults; migration is additive and independently
reversible by that loss of the setting.

## Security and verification

Only active administrators can change or view the policy. Forms use the
existing CSRF boundary, inputs accept only enumerated durations, and updates
are audit-recorded. Database constraints and contract tests reject unsupported
values. Password reset links are capped at 24 hours; invitation links at 7
days.
