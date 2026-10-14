# ADR-0061: Compile-time assertions for extension contracts

## Status

Accepted

## Decision

The versioned SDK package keeps compile-time fixtures for Module, ModuleRoute,
Theme, and BlockTheme. The experimental v0 module and theme contracts carry
deprecation guidance to `extension/v1` while remaining source-compatible.

## Consequences

Public signature changes fail during package tests instead of surfacing only
when an edition is built. Fixtures have no runtime registration or resource
cost. Deprecation remains advisory until a future major release removes v0.
