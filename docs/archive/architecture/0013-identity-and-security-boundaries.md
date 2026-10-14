# ADR-0013: Identity, authorization and security boundaries

- **Status:** Modified (partially) — [ADR-0014](0014-publication-routes-and-erasure-recovery.md) for the source of deletion history during restore; other security provisions remain valid.
- **Date:** 2026-09-05
- **Related documents:** [Threat model](../security/THREAT_MODEL.md), [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0011](0011-consistency-upgrade-and-recovery.md)
- **Implementation status:** Local user/session, role, CSRF, canonical-origin host binding, MFA, reset, invite, limiter, and audit controls have been implemented and verified with contract tests. OIDC/SAML and production provider/alerting are out of scope or an R3/R6 open gate; current status is in [roadmap](../ROADMAP.md).

## Decision

The initial identity method is local user + password + DB-backed opaque session. The identity provider's output is an application actor; authorization remains in the application service. OIDC/SAML, mobile-specific tokens, and API key distribution are not added without their own threat model/ADR. Site Edition's public JSON API does not imply support for identity-requiring mobile APIs.

Cryptographic primitives, CSRF, password hashing, and the TOTP algorithm are not written from scratch; a dependency with evaluated maintenance/license/build compatibility is chosen. Library versions are fixed at the end of the technical spike. The principle of ownership is not a justification to reinvent security algorithms.

## Lifecycle

- The first admin is created by the person with operational access via an explicit bootstrap command; there is no public first-user registration endpoint. Deletions/permission changes that accidentally lock out the last available admin account are prevented.
- The session identity is cryptographically random, the DB record is hashed, the cookie is host-only/HttpOnly/Secure/SameSite=Lax. Upon login and privilege change, the session is renewed; idle/absolute durations are applied separately. HTTPS is mandatory in production; the local development exception cannot be enabled in production mode.
- Sessions are revoked upon user disable, password reset, and critical role change. Registration/domain policies are evaluated on every application service call; a long-lived role cache does not hide the new authorization.
- Invite and reset tokens carry a separate purpose, user binding, short duration, hashed storage, and single-use limit. Consumption and password change are the same transaction. The response does not disclose account existence; the URL is generated from the trusted config origin.
- MFA for admin/publisher is a commercial pilot gate accessible from the internet. The initial method is TOTP and a hashed single-use recovery code; the seed is stored encrypted. The challenge is verified before registration becomes active; reuse of the code within the same time window is tested. Recovery/disable requires new authentication and audit.
- There is a documented CLI recovery path for lost MFA by the operations authority; there is no general web bypass. Recovery and password reset events generate a secure notification to the user.
- Password hashing, login, and MFA attempts carry separate concurrency/rate limit budgets. The cost of a safe hash for a single request does not translate into unbounded memory usage during a mass attack.

## Authorization and network boundary

- Route classification is default-deny. Admin, public API, private JSON endpoints, and background jobs use the same service policies. In addition to the role, record ownership, status, locale, and field allowlist are evaluated. Hooks relaxing the policy are not supported.
- CSRF is mandatory for admin state-changing requests; `SameSite` is a defense in depth. CORS does not open cross-origin access by default. The application does not trust forwarded host headers; if `public_origin` is defined, the incoming `Host` must match the canonical origin host/port. Health/readiness public outputs do not expose version/secret/DB details; detailed diagnostics and metrics are protected.
- Outbound webhooks/remote fetches are not opened to general use initially. When added, allowed targets, private/link-local/metadata network blocking, DNS/redirect re-checking, timeout, and response size limits are mandatory. A mere string check of the URL is not considered sufficient.

## Audit and the limit of trust boundary

Audit is append-only in the application API. The audit UPDATE/DELETE permission is removed for the PostgreSQL runtime role; the migration/maintenance identity is separate. For SQLite, this DB role guarantee does not exist against a person with full access to the file or server. A separate OS user/file permissions is the baseline boundary. A target with external append-only/retention control is separately certified for customers requiring tamper detection; the local log is not presented as "not even root can modify it."

Session/reset/preview tokens, passwords, TOTP seeds, and form bodies are not written to the log/audit. Data deletion/retention rules holistically cover revisions, soft-delete, search, media, and backup retention; restore is reconciled with an independent external deletion log without making deleted data visible again. Durable intent, checkpoint, retention, and isolation rules on missing logs are in ADR-0014.

## Impacts, alternatives and validation

Instead of stateless long-lived JWTs, DB session provides fast revocation and blue/green continuity; it has a DB-bound access cost. A small adapter boundary is chosen in the first phase instead of a general identity framework. MFA does not block the first development slice, but it blocks the commercial pilot.

Single-use token consumption, session revocation, and auth/audit transaction tests are the same in SQLite/PostgreSQL. Session does not enter the shared response cache. Compact auth concurrency is bounded; Distributed rate limit/session revocation passes multi-instance testing. In secret key rotation, the key IDs supported by the old/new slots overlap in a controlled manner; indefinitely old keys are not accepted. Restore security rules are in ADR-0011.

Security test scenarios and residual risks are tracked in the [threat model](../security/THREAT_MODEL.md). The model is not a penetration test or a security certificate.

## References

- [OWASP Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html): Session lifecycle and cookie defenses.
- [OWASP Forgot Password](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html): Single-use recovery token and limiting account enumeration.
- [OWASP SSRF Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html): External target and network validation boundaries.

These references are the rationale for the controls; delivery preferences like the MFA requirement in the pilot are HydroGo design decisions.
