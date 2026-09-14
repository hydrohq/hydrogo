# ADR-0022: Provider-neutral media storage

- **Status:** Accepted
- **Date:** 2026-09-15
- **Related decisions:** [ADR-0012](0012-content-media-and-portability.md), [ADR-0007](0007-capacity-profiles.md), [ADR-0011](0011-consistency-upgrade-and-recovery.md)

## Decision

Media application code uses the `internal/media.BlobStore` contract. The default
`local` driver keeps the existing filesystem behavior. An explicitly configured
`s3` driver supports AWS S3 and S3-compatible services such as MinIO and R2.
S3 details never cross the media service or module boundaries.

Objects remain immutable and are addressed by content-derived storage keys. The
database remains the source of asset state; object writes and DB transactions
are coordinated by the existing pending/ready lifecycle and orphan cleanup.
CDN delivery is represented by a deployment-level public base URL and is not
required for local or Compact profiles.

## Configuration

`media.storage_driver` is `local` unless explicitly set to `s3`. S3 requires an
endpoint and bucket; credentials are supplied by environment/secret injection.
The local staging directory remains required for bounded validation and image
processing. A CDN URL is optional and must be an HTTPS/HTTP origin.

## Consequences

This permits a zero-change Compact deployment and a shared object store for
multi-instance profiles. MinIO contract tests can run in WSL. Real cloud IAM,
retention, KMS, CDN purge and disaster-recovery behavior remain deployment
acceptance gates and are not implied by the adapter.
