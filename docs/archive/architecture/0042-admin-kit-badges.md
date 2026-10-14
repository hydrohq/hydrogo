# ADR-0042: Accessible admin capability badges

## Status

Accepted

## Decision

Admin collection headers may render sorted, escaped capability badges from the
kit registry. Badges are informational and do not imply authorization or
provider activation.

## Compatibility and capacity

The helper is additive, has no migration or database impact, and performs only
bounded in-memory sorting/template rendering. It adds no workers, routes, or
cache state.
