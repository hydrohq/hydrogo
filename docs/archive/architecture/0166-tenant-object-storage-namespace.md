# ADR-0166: Tenant object-storage namespace

- Status: Accepted (upload/key foundation; public routing and derivative propagation open)
- Date: 2026-09-19
- Related decisions: [ADR-0165](0165-tenant-scoped-media-metadata.md), [ADR-0154](0154-tenant-scope-primitive.md)

## Context

Tenant-filtered metadata alone does not protect an object bucket if two tenants
share opaque keys or if a signed URL can be generated for an unscoped object
lookup. Object keys need a deterministic namespace at the application storage
boundary.

## Decision

`media.StorageKeyForContext` prefixes uploaded object keys with
`tenants/{tenant_id}/` whenever a validated tenant scope is present. The media
service stores and sends that key to the configured BlobStore; legacy contexts
retain the basename path for single-customer compatibility. Existing object-key
validation continues to reject traversal/control characters.

## Consequences

- New scoped uploads cannot collide across tenants in S3-compatible storage.
- Signed URL requests already resolve metadata through the tenant-scoped media
  repository before asking the provider for a key.
- Public route tenant resolution, derivative optimizer scope propagation,
  object migration/backfill, and independent provider authorization remain open.
