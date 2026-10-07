# Admin route role and navigation smoke — 2026-10-08

## Coverage added

`e2e/custom/admin-management.spec.ts` now reads the enabled Edition Admin navigation from an authenticated Admin session and opens every unique `/admin` navigation path, requiring HTTP 200. It then signs in as editor and viewer, reads each role's visible navigation, and verifies every visible path also returns HTTP 200. Existing explicit assertions continue to require HTTP 403 from protected account-management, security, audit, backup, control, operations, and developer routes; they also exercise CSRF-less writes and valid-CSRF editor/viewer attempts to change user roles, permissions, accounts, and page content.

The test intentionally does not infer denial from a hidden navigation link: the viewer can read some content routes that are omitted from its menu. Authorization is checked from the HTTP response on explicitly protected paths.

## Result

Command:

```sh
cd e2e && npx playwright test custom/admin-management.spec.ts --project chromium --workers=1 --reporter=line --output=/tmp/hydro-admin-route-matrix
```

Result: Chromium 5/5 passed, including role boundaries, all active Admin navigation paths, Admin surface accessibility, responsive checks, and invitation flow. The test server was stopped after the run; no HydroGo test server process remained.

## Remaining scope

The test covers currently active navigation paths for this Edition fixture, not every module-specific deep link or every mutation form under each role and permission override. Full route and mutation permutation coverage remains a separate expansion.
