# ADR-0250: Optional OTLP exporter build capability

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related:** ADR-0007, ADR-0010, ADR-0247, ADR-0248, ADR-0249

## Context

The Compact runtime disables tracing by default, but the OTLP HTTP exporter,
OTLP protobuf, and gRPC dependencies were still statically linked into the
Workspace binary. An Edition that does not export traces should not pay this
binary and dependency-closure cost. Conversely, ignoring an explicitly
configured tracing endpoint in an exporter-free artifact would hide an
operational mistake.

## Decision

The default build continues to include OTLP support. An Edition can exclude
the provider with the `hydro_no_otlp` build tag. Customer package builders
accept `HYDRO_OTLP=enabled|disabled`, defaulting to `enabled`, and record the
selection in `PACKAGE-MANIFEST.md`.

When OTLP is excluded, `InstallOTLPProvider` accepts only a disabled tracing
profile and returns a no-op shutdown function. If the profile enables tracing
or contains an endpoint, it returns `ErrOTLPUnavailable`; it never silently
drops configured traces. Shared telemetry APIs and local in-memory metrics
remain available. This is build-time composition, not runtime plugin loading.

## Alternatives

- Keep exporter linked but disabled by profile: preserves flexibility but has
  no artifact/import-closure reduction.
- Silently no-op when a customer configures OTLP: rejected because operators
  would believe that spans are being exported when they are not.
- Remove tracing globally: rejected; it would break currently supported
  deployments and tests.

## Consequences

### Benefits

- OTLP exporter, OTLP protobuf, and gRPC package trees are absent from an
  explicitly exporter-free artifact.
- Default deployments preserve existing OTLP support.
- Misconfigured no-OTLP artifacts fail closed with a typed error.

### Costs and risks

- Export support is fixed at build time and must match deployment config.
- This tag removes exporter packages from the compiled artifact, not module
  requirements from `go.mod` or the repository's development/test supply
  chain.
- Other optional adapters remain to be independently composed and measured.

## Scale impact

- Compact: disabled tracing adds no exporter/goroutine/network dependency.
- Standard: default build retains OTLP; opt-out is permitted per Edition.
- Distributed: deploy a build with OTLP enabled if the operational profile
  requires external traces.

## Database impact

None; SQLite/PostgreSQL adapters, migrations, transactions, and adapter
certification are unchanged.

## Cache impact

None; cache keying, TTL, invalidation, stampede protection, and fallback are
unchanged.

## Deployment and compatibility

No schema change. If an artifact with OTLP disabled is deployed with
`HYDRO_OTEL_ENABLED=true` or an OTLP endpoint set, startup fails before
serving. Switching artifact capability requires configuration review in
blue/green deployment; rollback is to the previous artifact and supported
configuration pair.

## Validation

CI tests default and no-OTLP behavior, verifies exporter package absence, and
enforces the Compact SQLite/no-GraphQL/no-OTLP binary/import budget. Current
measurements are recorded in R125.

## References

- [Optional GraphQL runtime build capability](0249-optional-graphql-runtime-build.md)
- [Compact runtime dependency measurements](../validation/R125-no-otlp-runtime-variant-2026-09-22.md)
