# Blueprint Client Editions — 2026-09-21

The Workspace scaffold flow was exercised against two checked-in examples:

- `examples/project-edition/blueprint.yaml`
- `examples/article-edition/blueprint.yaml`

For each edition:

1. `hydro generate scaffold --blueprint ... --out ...` produced 29 reviewable
   artifacts under `generated/<collection>` and a `custom/` policy boundary.
2. `hydro resource check --root ...` passed with the generated lock and SHA-256
   compatibility evidence.
3. Generated `migration.yaml` records SQLite/PostgreSQL class, transactional
   behavior, checksums, compatibility, and rollback metadata next to the SQL.
   The generated Go contract test embeds both files and rejects missing table
   declarations or `DROP TABLE`/`DELETE FROM` statements; both example
   packages pass `go test`. The SQL remains an artifact for operator review;
   no command in this validation executed DDL or registered runtime
   workers/routes.

The examples intentionally leave workflow, tenant ownership, and authorization
in `custom/` so the Core/Edition boundary remains explicit.
