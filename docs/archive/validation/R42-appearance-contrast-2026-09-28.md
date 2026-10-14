# R42 / R104 — Site appearance and contrast

The Site appearance page stores validated light/dark color tokens, an
allowlisted font family, and same-origin media/theme asset paths. It updates
the color contrast result while editing and rejects values below 4.5:1 on the
server. Public page rendering applies the CSS tokens with its CSP nonce,
switches dark tokens with `prefers-color-scheme`, and emits configured favicon
and social image metadata.

The SQLite integration test covers saving all fields in one transaction and
rejecting an invalid palette without changing the prior settings. Existing
PostgreSQL `site_settings` contract tests exercise the same key/value store;
they require `HYDRO_TEST_POSTGRES_URL` to run.

Verification:

```text
go test ./modules/site/... ./modules/pages/transport ./cmd/hydro ./internal/ui
PASS
```

No migration, worker, cache, or additional per-request database query was
added. Asset uploads still use the existing Media surface; the appearance page
accepts its same-origin asset paths.
