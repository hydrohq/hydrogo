# ADR-0352: Public staff invitation acceptance form

- **Status:** Accepted
- **Date:** 2026-10-07
- **Supersedes:** None
- **Superseded by:** None

## Context

ADR-0305 issues one-time staff invitation links at
`/auth/invites/accept?token=...`; the application exposed only the POST API,
so a recipient following the link had no page to finish account creation.
ADR-0179 already defines the tenant-scoped, hashed, expiring token boundary.

## Decision

- When the `auth` module is active, serve a public, bilingual GET acceptance
  form at the issued URL using typed Design System Templ components.
- Keep acceptance and password creation inside the existing transactional
  `InvitationService`; retain hashed, single-use, expiring token behavior.
- For browser form requests (`Accept: text/html`), validate confirmation and
  redirect with allowlisted error codes or to login after success. Preserve
  the existing JSON API response and status contract for non-HTML clients.
- Mark page and POST responses `no-store` and `no-referrer`; remove the token
  from browser history after rendering. Keep request body and invitation
  attempts bounded by existing limits.
- Do not add schema, migration, background work, or a new service. Do not
  disclose whether an invitee already has an account.

## Alternatives

- Keep the POST-only API and require Editions to supply their own page: rejected
  because Core itself issues links to this route.
- Add a frontend runtime: rejected because server-rendered Templ and the
  existing Design System cover the flow without adding an Edition dependency.

## Consequences

### Benefits

- The issued invite URL completes account setup in the same Edition.
- API clients remain compatible; users receive localized, recoverable errors.

### Costs and risks

- Mixed old/new binaries may route the new GET page to an old instance during a
  rolling deployment. All serving instances must be upgraded before issuing
  links that rely on the page.

## Scale impact

- Compact: one bounded request and existing auth/database components only.
- Standard: no additional pools, workers, or persistent resources.
- Distributed: existing shared database and rate-limit configuration apply;
  no new cross-instance state is introduced.

## Database impact

- SQLite: reuses the existing transactional invite consume and user create.
- PostgreSQL: same existing transaction contract and tenant scope; no SQL or
  migration changes.
- Future adapters: no new adapter requirement.

## Cache impact

The page and acceptance responses are not cacheable. No cache keys or
invalidation behavior change.

## Deployment and compatibility

No migration is introduced, so there is no migration metadata or rollback.
The existing POST JSON contract remains available to old and new clients.
Deploy the GET route to every serving instance before generating invitations
for the browser flow. Rollback removes the page while leaving the prior API
behavior intact.

## Validation

- Chromium acceptance using an invitation issued and consumed through the
  SQLite-backed service; verify mismatch recovery and subsequent editor login.
- Auth HTTP tests for HTML redirect safety and unchanged JSON API behavior.
- Design System and auth store tests; migrationguard and moduleguard.
- axe-core WCAG 2.1 A/AA scan of the acceptance form.

## References

- [ADR-0179: Tenant-scoped auth security tokens and MFA](0179-tenant-scoped-auth-security-tokens.md)
- [ADR-0305: Staff invitation share message](0305-staff-invitation-share-message.md)
