# ADR-0087: Harden admin blank-target links

## Status

Accepted

## Context

The admin shell opens the public site in a new browsing context. A link opened
with `target="_blank"` can otherwise retain an opener reference and let the
opened page navigate or inspect the originating window.

## Decision

Admin shell links using a blank target receive `rel="noopener noreferrer"` at
render time through the nonce-protected shell script. The contract also applies
to links added by optional admin fragments.

## Validation

`internal/ui/admin_areas_test.go` asserts that the rendered shell contains the
blank-target link and the hardening contract. Rich-text links already emit the
same relation directly.
