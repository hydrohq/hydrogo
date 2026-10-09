# Failed Admin login accessibility — 2026-10-09

The browser test submits invalid credentials through the real `/auth/login`
endpoint and verifies HTTP 401, the localized generic alert copy, and its
`role="alert"` semantics. It runs in English and Turkish, each in light and
dark color modes, then checks WCAG 2.1 A/AA with axe: four combinations.

The first light-mode scan found the error text `#ef4444` on a pale red alert
surface at 3.08:1, below the 4.5:1 AA target. Light mode now uses `#b91c1c`
for the error token. All four focused cells passed.

The complete seven-test `custom/admin-management.spec.ts` Chromium file passed
**7/7** on SQLite (2.8 minutes) and **7/7** on PostgreSQL 18.6 (2.9 minutes).
PostgreSQL used a `NOSUPERUSER NOBYPASSRLS` app role and a separate `BYPASSRLS`
backup operator; its disposable container was removed. `go test
./internal/ui -count=1` passed, and Impeccable's detector returned no findings
for the modified login UI and browser test.

```sh
cd e2e
npm test -- --project=chromium --workers=1 custom/admin-management.spec.ts

HYDRO_E2E_DATABASE_DIALECT=postgres \
HYDRO_E2E_DATABASE_DSN='postgres://<app-role>:<password>@127.0.0.1:<port>/<database>?sslmode=disable' \
HYDRO_PG_PRIVILEGED_ROLE='<backup-operator-role>' \
npm test -- --project=chromium --workers=1 custom/admin-management.spec.ts

cd ..
go test ./internal/ui -count=1
```

This closes invalid-password feedback only. MFA challenge and incorrect-code,
rate-limit/lockout, service-unavailable, physical-device, and deployed Edition
identity behavior remain separate checks.
