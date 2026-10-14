# ADR-0221: Tenant and Module Dimensions in Endpoint Reporting

## Status

Accepted — 2026-09-22.

## Decision

In addition to route-normalized path statistics, endpoint telemetry retains at most
128 bounded `(tenant_id, module)` dimensions. The tenant is resolved exclusively from
a verified tenant scope, and the module is extracted from the bounded first path segment;
if no tenant scope is present, `default` is used. Query parameters, slugs, and user-controlled
unbounded tags are strictly excluded from measurement.

## Verification

`Metrics.DimensionStats`, the Admin JSON/CSV export, and
`TestMetricsExposeTenantAndModuleDimensions` verify the bounded dimension contract.
