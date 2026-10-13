# Admin Studio MFA Self-Service Validation

**Date:** 2026-10-07  
**Status:** Completed  
**Scope:** MFA setup, verification, and deactivation flow for single Edition administrator and account owner.

## User Journey

- `/admin/account` displays MFA status; if master key is not configured,
  the setup action is not presented.
- Setup reveals the TOTP secret, authenticator URI, and single-use recovery codes.
  The user cannot enable MFA without confirming that they stored the codes.
- MFA activation requires a current TOTP code. Deactivation requires a current TOTP or
  recovery code.
- While an active factor exists, new enrollment is rejected with 409; the factor cannot
  be silently changed. Re-enrollment rotates recovery code hashes.
- Status, enrollment, and account view inline scripts are served with `private, no-store`
  caching behavior. CSP inline asset hashes are merged into the existing strict CSP.
- The UI uses typed Templ and Admin Studio shared `studio-card`, `studio-form`,
  `studio-form-field`, and `button` styles. Browser rendering was inspected;
  heading, description, card, and actions render with the uniform Admin visual language.

## Validation

Successful commands and scenarios:

```sh
go test ./modules/designsystem ./internal/auth ./internal/auth/sqlstore ./internal/ui ./cmd/hydro -count=1
go test ./tests/contract -run '^TestSQLiteAuthContract$' -count=1
go test ./tests/contract -run '^TestPostgresAuthContract$' -count=1
HYDRO_TEST_POSTGRES_URL=<isolated PostgreSQL 18 test URL> go test -p 1 ./... -count=1 -timeout=5m
HYDRO_TEST_POSTGRES_URL=<isolated PostgreSQL 18 test URL> go test ./tests/contract -run '^TestPostgres' -count=1 -p 1
go vet ./modules/designsystem ./internal/auth ./internal/auth/sqlstore ./internal/ui ./cmd/hydro
go run ./tools/actionguard
go run ./tools/importguard -root examples/client-edition
go run ./tools/importguard -root examples/news-edition
go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json
go run ./tools/migrationguard -root migrations
go run ./tools/importguard -root examples/construction-edition
```

The full PostgreSQL run completed on PostgreSQL 18.6 using the `NOBYPASSRLS` test role in a
clean isolated database. The dedicated PostgreSQL auth contract suite passed again in a separate
clean database. The auth contract covers enabling MFA, incorrect codes, deactivating via TOTP,
deactivating via recovery codes, rejecting re-enrollment with an active factor, and code rotation.
No migration was added.

```sh
cd e2e
npx playwright test custom/admin-mfa-settings.spec.ts --project=chromium --workers=1 --reporter=line
npx playwright test custom/admin-management.spec.ts custom/admin-password-reset.spec.ts custom/admin-mfa-settings.spec.ts --project=chromium --workers=1 --reporter=line
```

The standalone MFA browser test passed **1/1**, and the combined admin management, invitation, password reset,
and MFA suite passed **5/5**. The expanded Chromium flow logs out after TOTP activation and logs back in,
prompts for an MFA challenge following the password, and confirms that the `/auth/me` session is not established
until the code is verified. After logging in with the valid TOTP, 15 Edition management routes are accessed;
security IP ban is added and removed. On the account MFA screen, rejecting deactivation with an incorrect code
and successful deactivation with the correct code are verified. On a 390px mobile viewport, axe-core WCAG 2.1 A/AA
scans and horizontal overflow checks passed. CSP/no-store headers and onboarding visibility for the MFA account view
are also verified in browser tests.

Within the same MFA-verified administrator session, creating a page draft and downloading a signed backup
after re-verifying password also passed. The administrator toggled the editor's `admin.content.write` permission
to deny/allow; the deny change revoked the editor's active session and the permission value was confirmed in the
refreshed list. Disabling the viewer account also revoked the active viewer session. With this additional flow,
the combined Chromium group passed **5/5**.

In auth unit tests, the role/permission default matrix was expanded: 27 combinations for three built-in roles
x nine built-in permissions; in each cell, default decision, user override reversing the decision, and rejection
of inactive users despite overrides are validated. Verifying that Edition-specific permissions existing solely
in the catalog does not grant editor/viewer access by default is also checked.
Command: `go test ./internal/auth -run 'TestPermissionCatalog(RoleAndOverrideMatrix|AllowsOnlyRegisteredPermissionsAndExplicitOverrides|RejectsUnknownWritesAndFiltersRemovedPermissions)' -count=1`.

Across 15 management routes, the SHA-256 hash of each rendered inline script is compared one-to-one
with the allowlist in the response `Content-Security-Policy` header; Playwright additionally collects CSP console
violations. News and media Admin script hashes are cataloged for machine-generated code, with the News catalog
appended only when the module is enabled. The Control page does not embed the CSRF token into the script body;
it reads from the cookie, ensuring the script hash does not depend on user/session data. MFA E2E passed 1/1,
and the combined admin management/invite/password reset/MFA group passed 5/5. `go test ./... -count=1 -p 1 -timeout=5m`,
`go vet ./...`, and action, module, migration, and Edition import guards passed as well.

The MFA mobile screenshot was generated as a Playwright artifact under `e2e/artifacts/admin-mfa-mobile.png`
and visually inspected. Artifacts are not committed to git.

## Boundaries

This effort introduces no new cross-tenant Super Admin or new auth routes/schemas.
The TOTP/recovery API operates on top of the existing Edition auth contract.
Unit tests cover the built-in role/permission decision matrix; the complete combinatorial matrix
for all management HTTP routes x roles x mutation types, Edition-specific permission mutations, and
database round-trips remains open. Other Phase 1/2/3/5 open acceptance items are tracked in the main product plan.
CSP script hashes across 15 management routes have been verified; full acceptance evidence across all theme
variants and all admin mutations is not considered satisfied.
