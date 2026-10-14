# ADR-0088: Isolate routes for inactive modules

## Status

Accepted

## Context

HydroGo supports compact profiles where optional modules are disabled. Registering
an optional module's admin route unconditionally violates the module boundary,
exposes dead UI entry points, and creates avoidable routing overhead.

## Decision

The core route assembly registers `/admin/site` only when the site module is
enabled and `/admin/media` only when the media module is enabled. Core admin
surfaces remain available independently. The route pattern helper is covered by
a regression test for both disabled and enabled module sets.

## Consequences

Compact profiles no longer expose optional module routes. Enabling a module adds
its route set deterministically during startup; no runtime route mutation is
needed.

## Validation

See `cmd/hydro/main_test.go` and the full local verification hook.
