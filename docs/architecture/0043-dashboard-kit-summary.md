# ADR-0043: Admin dashboard capability summary

## Status

Accepted

## Decision

The Admin Studio dashboard renders an informational, accessible capability
summary for collections that declare kits. Existing collection tiles remain
unchanged; the summary never implies provider activation or authorization.

## Compatibility and capacity

This is additive presentation output with no migration or database impact. It
renders only bounded metadata and adds no workers, routes, cache entries, or
provider dependencies.
