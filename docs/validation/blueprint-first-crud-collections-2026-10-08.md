# Blueprint first CRUD through optional Collections — 2026-10-08

## Verified workflow

A fresh project generated with `hydro project new edition --preset compact-site --no-demo --with-collections` now opts into `auth`, `members`, and `collections`. The scaffold includes `data/.gitkeep`, so the configured SQLite database can be created from the project root. Its README documents the complete sequence:

1. `hydro resource create --name article --field title:text --out .`
2. `hydro migrate --config config.yaml`
3. `hydro collections sync --config config.yaml --generated generated --confirm-admin-crud`
4. `hydro create-admin --config config.yaml --email admin@example.test --password-stdin`
5. `hydro serve --config config.yaml`

The Playwright fixture now follows this flow from a freshly generated Edition, builds and invokes the local `hydro` binary from the Edition root, and completes the Collections Admin CRUD scenario in Chromium. The browser scenario passed 1/1. Project generator and CLI tests verify opt-in module selection and the SQLite data directory.

## Product boundary

Collections is an explicit bridge to its existing `/admin/collections` route and Admin CRUD contract. Sync requires `--confirm-admin-crud`, runs no DDL, does not translate generated policy callbacks, and leaves generated SQL as a reviewable proposal. This path does not activate the Blueprint's generated API/Admin registry, Edition authorization policy, drafts, revisions, workflow, or localization. Those still need a runnable, policy-bound Edition composition. Without `--with-collections`, the generated project does not include the optional Collections module.
