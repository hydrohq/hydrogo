# R102 — Tenant-scoped media metadata

Date: 2026-09-19

## Evidence

The media migration adds tenant discriminators and lookup indexes to assets and
variants for both SQL dialects. The SQL store automatically scopes metadata
reads and transitions when context contains a validated tenant, and explicit
fail-closed mode rejects missing scope. SQLite tests prove another tenant cannot
read or transition an asset.

Commands:

- `go test ./internal/media/... ./internal/database ./internal/tenant -count=1`
- `go vet ./internal/media/... ./internal/database ./internal/tenant`

Result: passed.

This is metadata-boundary evidence only. Object-store namespace isolation,
signed URL authorization, tenant backfill and production PostgreSQL/RLS tests
remain open.
