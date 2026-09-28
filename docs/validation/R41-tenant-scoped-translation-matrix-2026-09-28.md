# R41 — Tenant-scoped static UI translation matrix

The Site module exposes `/admin/translations` with a key search, TR/EN
side-by-side editor, per-cell save forms, and a visible marker for missing
locale entries. The editor accepts only keys from the embedded catalogs and
bounded values. A blank value removes the local override and returns to the
embedded locale fallback.

Overrides persist in `site_settings` under the dedicated
`i18n.override.<locale>.<key>` namespace. The settings contract is exercised
by the existing SQLite and opt-in PostgreSQL visibility dialect tests. The
HTTP integration test verifies that a saved value appears in a subsequent
rendered Admin Account view and remains isolated from a different tenant.
At startup, active Site tenants load persisted overrides into a per-tenant,
catalog-bounded in-memory lookup map.

Verification performed:

```text
go test ./internal/i18n ./internal/ui ./modules/site/... ./cmd/hydro
PASS
```

The PostgreSQL-specific test requires `HYDRO_TEST_POSTGRES_URL`; it is skipped
when that environment variable is absent. No migration, worker, external
translation service, or inactive-module query was introduced.
