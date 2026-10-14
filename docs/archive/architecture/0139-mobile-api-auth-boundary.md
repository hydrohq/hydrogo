# ADR-0139: Versioned mobile API authentication boundary

- **Status:** Accepted (contract); implementation is a separate product slice

## Decision

- Mobile clients use a versioned JSON API and short-lived access tokens with refresh-token rotation; browser cookies are not accepted as mobile credentials.
- Every resource operation resolves an application actor and authorization scope before repository access.
- Tokens carry an opaque subject and server-side revocation family; raw tokens never enter logs, analytics, or cache keys.
- API version, pagination, idempotency, error envelope, and deprecation policy are explicit before a public mobile SDK is shipped.

## Release gate

Threat-model review, abuse/rate-limit tests, device/session revoke tests, SQLite/PostgreSQL contract tests, and a documented migration window are required before enabling a public mobile API.
