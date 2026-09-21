# ADR-0212: Embedded release-owned assets

- **Status:** Accepted
- **Date:** 2026-09-21

## Decision

Release-owned browser libraries are embedded through `static/embed.go` and
served from the binary under `/vendor/`. User uploads, tenant media, themes,
and backups remain external runtime resources. Asset manifests and release
verification provide deterministic size/hash evidence for shipped artifacts.

The embed boundary must remain small and module-aware: an inactive module must
not cause its assets to be embedded or served. No development watcher or
frontend package manager is required by the production binary.

## Consequences

- Compact deployments can ship one binary without a vendor directory.
- Mutable customer data is not baked into releases.
- Large or frequently changing media stays in the configured storage adapter.
- Asset changes require a release artifact and hash verification rather than a
  runtime download.

