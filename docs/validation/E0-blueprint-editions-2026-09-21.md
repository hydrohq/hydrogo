# Blueprint Client Editions — 2026-09-21

The Workspace scaffold flow was exercised against two checked-in examples:

- `examples/project-edition/blueprint.yaml`
- `examples/article-edition/blueprint.yaml`

For each edition:

1. `hydro generate scaffold --blueprint ... --out ...` produced 28 reviewable
   artifacts under `generated/<collection>` and a `custom/` policy boundary.
2. `hydro resource check --root ...` passed with the generated lock and SHA-256
   compatibility evidence.
3. Generated migration SQL is an artifact for operator review; no command in
   this validation executed DDL or registered runtime workers/routes.

The examples intentionally leave workflow, tenant ownership, and authorization
in `custom/` so the Core/Edition boundary remains explicit.

