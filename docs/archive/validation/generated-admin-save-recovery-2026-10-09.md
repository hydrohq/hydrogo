# Generated Admin save error recovery — 2026-10-09

## Change

Generated collection create/update API transport or 5xx failures return HTTP
502 with the original edit form rendered in the shared Admin shell. The error
alert is localized and accessible. User-entered values, CSRF token, and save
button are preserved, so a transient problem can be retried without retyping.
If a delete request fails and the record is still readable, the Admin restores
its edit form with the current record, CSRF token, delete action, and a
localized alert that tells the user to check the list before retrying. If the
record cannot be fetched (including a delete that completed despite an
ambiguous response), the handler returns delete-specific localized guidance
without exposing the API response.

## Verification

- A fake API returns an HTTP 500 containing a PostgreSQL URL, internal host,
  filesystem path, and panic marker.
- Create, update, and delete cases, in both Turkish and English, show the expected
  localized `role="alert"` inside the shared shell.
- Submitted create/update values remain in their forms. Delete restores the
  current record and delete action with CSRF. No internal detail is returned,
  and no success redirect occurs.
- Focused test passed:
  `go test ./internal/adminstudio ./internal/i18n -run '^TestGeneratedAdminLocalizesAndSanitizesResourceFailure$' -count=1 -p 1 -timeout=4m`.
- Delete fallback test passed:
  `go test ./internal/adminstudio ./internal/i18n -run '^TestGeneratedAdminDeleteFailureWithoutReadableRecordUsesDeleteCopy$' -count=1 -p 1 -timeout=4m`.
- Full package tests passed:
  `go test ./internal/adminstudio ./internal/i18n -count=1 -p 1 -timeout=5m`.
- `go vet ./internal/adminstudio ./internal/i18n` and `git diff --check` passed.

## Browser retry coverage added — 2026-10-09

The generated rich-text Chromium suite passed 8/8 on the local Edition fixture.
It covers Quill formatting and API round-trip; 502 recovery with preserved
title/body, localized alert, axe, retry and API readback; lazy block-editor load
and JSON round-trip; stale-edit conflict; relation/media selection; localized
draft publishing; and draft review/discard state separation.

```sh
cd e2e
npx playwright test --project=chromium --workers=1 custom/generated-richtext.spec.ts \
  --reporter=line --trace=off
```

## Limits

The UI journeys use the local in-process Edition fixture. Production session
integration and deployed Edition error mapping remain open. The delete
fallback is covered for Turkish and English when the record is no longer
readable.

## Full SQLite and PostgreSQL browser replay — 2026-10-09

The complete eight-test Chromium suite passed **8/8 on SQLite** (30.5 seconds)
and **8/8 on PostgreSQL 18.6** (29.7 seconds). The PostgreSQL run used a fresh
temporary cluster and application role. The fixture created and dropped an
isolated schema for its editorial tables; the cluster itself was stopped and
removed after the run. Covered journeys include Quill create/edit, block editor
round-trip, failed-save retry, stale-edit conflict, relation/media picker,
localized drafts, review/publish, and draft discard.

This certifies the browser fixture against both database adapters, not the
product's generated migration, real customer authorization/session composition,
or deployed Edition error mapping. `generated-richtext` is an in-process test
Edition with fixture-owned routes and editorial workflow callbacks.

```sh
cd e2e
TMPDIR=/var/tmp/hydro-go-tests npx playwright test custom/generated-richtext.spec.ts \
  --project=chromium --workers=1 --reporter=line --trace=off
HYDRO_TEST_POSTGRES_URL='postgres://<app-role>@127.0.0.1:<port>/<empty-test-db>?sslmode=disable' \
  TMPDIR=/var/tmp/hydro-go-tests npx playwright test custom/generated-richtext.spec.ts \
  --project=chromium --workers=1 --reporter=line --trace=off
```
