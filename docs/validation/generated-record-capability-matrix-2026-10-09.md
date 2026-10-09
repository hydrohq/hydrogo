# Generated Admin record-level capability matrix — 2026-10-09

## Coverage

The independent Edition test server now seeds a tenant-owned article whose ID
has an Edition-policy-defined read-only capability. The editor can read the
record in the generated list and open its detail page, but the detail has no
edit or delete forms. Direct valid-CSRF Admin update/delete requests and direct
API update/delete requests all return 403. API readback confirms the original
title is unchanged. The same editor still creates and updates an ordinary
record; the existing Viewer read-only route/API assertions remain in the same
browser journey.

Both the generated Admin `ResourceAccess` callback and the generated API
`CanUpdate`/`CanDelete` policy enforce the test Edition's record-level rule.
This confirms the shared panel reflects Edition policy while the Edition API
remains authoritative for direct callers.

## Verification

- External Edition Chromium Playwright file: **2/2 passed on SQLite**.
- External Edition Chromium Playwright file: **2/2 passed on PostgreSQL 18.6**.
- `go test ./... -count=1 -p 1` from `tests/external-edition` — passed.
- `go vet ./...` from `tests/external-edition` — passed.
- Root `go run ./tools/importguard -root tests/external-edition` — passed.

The browser fixture uses test-only identity headers and loopback listeners.
Production identity resolution and deployment policy remain external release
gates. This closes one record-level role/capability cell; it does not close the
complete route × role × mutation matrix.
