# ADR-0299: Optional form geolocation

Status: Accepted

## Context

Form operators asked for coarse country and city metadata on the private inquiry
inbox. Capturing full addresses or consulting a remote geolocation service would
increase privacy and availability costs.

## Decision

When Forms is active, deployments may provide an operator-managed offline MMDB
database. Lookup is performed in process with the Pure Go MaxMind reader. A
deployment may also configure trusted proxy CIDRs; only requests whose direct
peer matches those CIDRs may supply Cloudflare `CF-Connecting-IP` or
`CF-IPCountry` values. Untrusted forwarding headers are ignored. Private,
loopback, and link-local addresses are stored as a local-network marker.

Only ISO country code, bounded city name, and the local-network marker are
stored with the existing form submission. Raw client IP storage and external
lookup calls are not added. Empty configuration disables database lookup.
Database files and their update/license lifecycle remain operator managed.
The reader is opened only when Forms is enabled and a path is configured.

## Compatibility and operations

Migration `202609290008_form_geolocation` is additive with empty/default values,
so old binaries can ignore the new columns during a rolling deployment. SQLite,
PostgreSQL, and MySQL have explicit migration variants; rollback may preserve the
columns, while removal of the collected data requires restoring a pre-migration
backup. No cache, worker, route, or background job is introduced. The lookup
uses one shared read-only database handle and bounded per-request output.

## Verification

Resolver tests cover local addresses, trusted and untrusted proxy headers,
country normalization, and malformed ranges. Form transport tests verify that
resolved metadata is persisted and rendered. Migration contract tests certify
the additive schema across supported dialects.
