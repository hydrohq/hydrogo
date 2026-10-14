# ADR-0251: Optional S3 media provider build capability

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related:** ADR-0007, ADR-0010, ADR-0012, ADR-0022, ADR-0247, ADR-0248

## Context

ADR-0022 defines filesystem storage as the default and S3-compatible object
storage as an explicit provider choice. The MinIO SDK was nevertheless linked
into every Workspace build. This increases artifact and dependency surface
for local/Compact Editions that never configure remote object storage.

## Decision

The default build continues to include the S3-compatible provider. An Edition
may exclude it with `hydro_no_s3`; customer package builders expose this as
`HYDRO_S3=enabled|disabled` (default `enabled`) and record the selection in
`PACKAGE-MANIFEST.md`.

In an S3-free build, `media.NewS3BlobStore` returns `media.ErrS3Unavailable`
and config validation rejects `media.storage_driver: s3` before serving.
Local filesystem media, media validation/quarantine lifecycle, database
metadata, and provider-neutral contracts remain intact. Provider selection is
build-time composition, not runtime plugin loading.

## Alternatives

- Keep MinIO linked but select local at runtime: preserves flexibility but
  does not reduce the unused dependency or binary surface.
- Silently fall back from configured S3 to local storage: rejected because it
  can split media state and cause data loss or broken URLs.
- Remove S3 globally: rejected because Standard/Distributed Editions and the
  direct-upload workflow depend on it.

## Consequences

### Benefits

- Local-media-only Editions exclude MinIO and its transitive packages.
- Misconfigured S3 use fails closed during config validation and at the
  provider constructor boundary.
- Media domain interfaces and local behavior are unchanged.

### Costs and risks

- S3 capability is fixed in the binary; deployments using remote storage must
  use an artifact with the provider linked.
- Existing tests that exercise S3 remain in the default build; the opt-out
  build has explicit tests for local storage and fail-closed rejection.
- This removes compiled dependencies but does not remove MinIO from `go.mod`
  or all source/test supply-chain scans.

## Scale impact

- Compact: local media remains; S3 can be omitted for single-node deployments.
- Standard: choose S3-enabled artifacts when shared object storage is needed.
- Distributed: shared S3-compatible media remains required by the deployment;
  use the provider-enabled artifact.

## Database impact

None. Database asset metadata, migrations, pending-to-ready lifecycle,
transactions, and adapter contracts remain unchanged.

## Cache impact

None; cache keys, TTL, invalidation, stampede protection, and fallback are
unchanged.

## Deployment and compatibility

No schema migration. `storage_driver: s3` with a no-S3 artifact fails config
validation before startup. Blue/green versions must use a provider-enabled
artifact wherever remote objects must remain readable. Rollback requires the
previous binary and its matching storage configuration.

## Validation

CI tests default S3 contracts and no-S3 fail-closed behavior, ensures MinIO is
absent from the Compact no-S3 import closure, and enforces the measured
Workspace artifact/package budget. Results are in R126.

## References

- [Provider-neutral media storage](0022-provider-neutral-media-storage.md)
- [SQLite-only customer runtime build variant](0248-sqlite-only-customer-runtime-build.md)
- [Compact runtime dependency measurements](../validation/R126-no-s3-runtime-variant-2026-09-22.md)
