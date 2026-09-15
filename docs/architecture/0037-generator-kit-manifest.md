# ADR-0037: Generator kit manifest

## Status

Accepted

## Decision

Resource specs may declare optional capability kits. The generator validates the
known vocabulary and carries the selection into the admin manifest as
metadata; it does not import or activate implementations automatically.

## Compatibility and capacity

The field is additive and migration-free for SQLite/PostgreSQL. Unknown kits
fail at generation time, while omitted kits preserve existing output and the
Compact profile's zero-overhead behavior.
