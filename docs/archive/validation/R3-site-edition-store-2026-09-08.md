# R3 Site Edition data contract validation

**Date:** 2026-09-08  
**Status:** Settings/navigation/redirect/form inbox store gate succeeded; public settings/navigation projection, canonical/OG helper, published-only sitemap HTTP route, admin-only mutation routes, allowlist/rate-limited contact submit, inbox status/purge, and in-transaction audit operations are ready.

Migration `202609080009_site_edition` adds site settings, ordered navigation, 301/308 redirect, and bounded form inbox tables for both dialects. The `modules/site` validation layer limits key/path/URL/email/payload sizes; the SQL store reads visible settings and navigation solely from explicit public/visible fields. Form submission records expect a hash of the source IP instead of the raw value.

Forward migration `202609080011_navigation_position_unique` also makes the navigation position value within a locale unique at the DB level; concurrent/direct writes bypassing service validation are rejected in the contract test.

Executed check:

```text
go test ./tests/contract ./modules/site/...
go test ./...
```

The SQLite contract passed; the PostgreSQL test is skipped if `HYDRO_TEST_POSTGRES_URL` is absent. Canonical/OG values and the sitemap XML helper are unit tested; the sitemap route is fed only by the `ListPublishedRoutes` projection. Admin mutation routes accept only an active admin context. Contact submit uses only the `contact` allowlist, a 64 KiB body limit, and bounded IP/form limiters. Admin inbox status transitions are allowlisted; retention purge and site mutation audit events are validated in the contract.
