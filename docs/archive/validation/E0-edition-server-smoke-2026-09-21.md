# Client Edition Server Smoke — 2026-09-21

`scripts/verify-edition-servers.ps1` built both standalone Edition binaries,
ran their explicit operator migration command, started each server against an
isolated SQLite database, and completed health plus generated CRUD requests:

```text
PASS: project Edition migration, healthz, POST and GET
PASS: article Edition migration, healthz, POST and GET
```

The server command does not apply DDL. `migrate` is a separate subcommand and
uses the generated, reviewable `MigrationSQL` artifact. The policy in these
examples is intentionally a local demo policy; production Editions must bind
authenticated member/tenant/RBAC policy before exposing the generated route.
