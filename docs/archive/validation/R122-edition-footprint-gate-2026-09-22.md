# R122 — Generated Edition footprint gate

Date: 2026-09-22

## Evidence

Initial dual-driver baseline (the gate was later extended under ADR-0248; see
R123 for the current four-variant results):

Command: `pwsh -NoProfile -File scripts/verify-edition-footprint.ps1`

- `project-edition`: PASS — 15,782,400 bytes; 89 non-standard packages.
- `article-edition`: PASS — 15,782,400 bytes; 89 non-standard packages.
- Both builds exclude the Workspace CLI, generator/project/release tooling,
  GraphQL, MinIO, Redis, and OTLP/gRPC exporter dependency trees.
- Binary cap: 18 MiB. Import-closure cap: 100 non-standard packages.
- `git diff --check` and the Edition server integration script remain separate
  gates; this report records a Windows amd64 Go 1.27.0 reproducible baseline.

## Limits

These historical measurements predate the SQLite-only variant. The example servers include one generated resource, a reviewed migration,
and health route. They are not a full CMS or Construction Client Edition and
do not prove auth/Admin/Site/tenant/profile budgets. The universal
`cmd/hydro` executable remains 36.98 MiB stripped on this host. R120 tracks the
remaining P0 work; this gate prevents regression only for the narrow examples.
