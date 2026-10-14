# ADR-0085: Verify embedded vendor asset reachability

## Status

Accepted

## Context

HydroGo embeds editor/browser libraries in the binary. An asset can remain in
the release tree after its feature is removed, increasing payload and
maintenance cost without serving a runtime surface.

## Decision

`tools/assetguard` scans `static/vendor` runtime `.js` and `.css` files and
requires each to be referenced by a Go, HTML, JS, or CSS source outside the
vendor tree. License files and placeholders are excluded. The guard detects
unused assets but does not remove files automatically.

## Consequences

Release builds fail early when a vendor runtime asset loses its consumer.
Intentional dynamic consumers must retain an explicit `/static/vendor/...`
reference in the serving surface. Deletion remains a reviewed change.

## Verification

See `tools/assetguard/main_test.go` and
`docs/validation/R5-release-asset-hygiene.md`.
