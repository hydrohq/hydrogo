# ADR-0089: Validate route conflicts before startup

## Status

Accepted

## Context

Route assembly combines Core, module, and compatibility alias patterns. A
duplicate pattern previously reached `http.ServeMux`, whose registration panic
could terminate the process without an actionable configuration error.

## Decision

`internal/router.New` expands compatibility aliases, tracks every resulting
pattern, and returns a descriptive duplicate-pattern error before registering
handlers. This applies equally to direct routes and generated `/api/v1` aliases.

## Consequences

Module integration mistakes fail deterministically during startup and are
testable without launching a server. Valid route registration behavior is
unchanged.

## Validation

`internal/router/router_test.go` covers a duplicate generated API alias. The
full local verification hook covers the assembled application route set.
