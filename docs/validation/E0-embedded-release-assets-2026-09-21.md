# Embedded release assets — 2026-09-21

`static/embed.go` embeds only the release-owned `static/vendor` tree. The
server serves it through `fs.Sub(staticassets.FS, "vendor")`; themes, uploaded
media, tenant files, and backups are not part of the embedded filesystem.

The existing asset manifest/release verification tools record byte size and
SHA-256 for declared assets. This preserves single-binary distribution while
keeping mutable customer data external.

Production asset size budgets and per-Edition inactive-module asset pruning
remain release-operator checks.

