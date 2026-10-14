# ADR-0069: Keep module lifecycle hooks optional and Core-orchestrated

## Status

Accepted

## Decision

`extension/v1` publishes separate optional contracts for install, enable,
disable, upgrade, and migration preflight. Hooks receive a versioned request;
Core owns dependency resolution, migration execution, draining, and data
deletion policy. A module is not forced to implement irrelevant lifecycle
methods.

## Consequences

Client Editions can describe lifecycle behavior without importing internal
orchestration or running DDL from an HTTP request. Plans reject duplicate or
empty migration IDs before Core evaluates them.
