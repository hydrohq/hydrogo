# ADR-0025: Separate frontend member identity

- **Status:** Accepted for MVP implementation
- **Date:** 2026-09-11
- **Related:** [ADR-0013](0013-identity-and-security-boundaries.md), [ADR-0017](0017-commerce-consistency-and-payments.md)

## Decision

Frontend members are a separate identity class from Admin Studio users. They use `members` and
`member_sessions` tables, a separate `hydro_member_session` cookie, and `/member/*` endpoints. A member
session is never accepted by the Admin authorizer and carries no role or panel capability.

The MVP supports registration, login, logout, current-member lookup, and admin active/inactive management.
The member module is opt-in through the module list and uses the same SQLite/PostgreSQL migration discipline.

Email verification, password reset, consent/legal text, deletion/export, and customer analytics remain explicit
follow-up gates; they must not be implied by the MVP session flow.
