# R5 — Sanitized theme content validation

**Status:** Local contract succeeded.

The generated Client Edition and News Edition examples now escape untrusted
theme chrome while rendering the Core-produced, already-sanitized
`HTMLContent` in the content slot. This restores rich-text markup without
weakening the Core sanitizer boundary.

Evidence: `go test ./...` passed independently in
`examples/client-edition` and `examples/news-edition` on 2026-09-14. Full
repository pre-push verification is required before delivery.
