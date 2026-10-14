# Admin Studio list validation — 2026-09-10

Related plan: [HEADLESS_ADMIN_STUDIO_PLAN.md](../HEADLESS_ADMIN_STUDIO_PLAN.md). The shared metadata list renderer has been validated on the current seeded SQLite demo binary.

## Automated evidence

- `go test ./...`
- `internal/adminstudio` form/list/registry tests
- Pages service contract: published summary list does not carry draft body and blocks.

## Local HTTP smoke

After opening a session with demo credentials:

| Route | Result |
|---|---|
| `GET /admin/news` | `200`; shared `<table>` and `HydroGo Demo Release Notes` are visible |
| `GET /admin/pages` | `200`; shared `<table>` and `HydroGo Showcase` are visible |

In the expanded current binary smoke, `GET /admin/commerce/products/demo-shirt/variants` also returned `200` and used the shared table/form renderer. `GET /admin/pages/demo-showcase` presented the block editor, while `GET /admin/news/demo-news` presented the metadata textarea along with schedule/publish workflow actions.

The `Edit` action of each row navigates to the module's existing detail route. Mutation and publish/schedule rules still belong to the respective application services; the list renderer carries no DB access or authorization decisions.
