# R5 form contract — 2026-09-16

`go test ./internal/forms -count=1` PASS.

The bounded form contract validates metadata and normalizes submissions without
persisting or exposing input. Unknown fields, oversized values, invalid select
options, duplicate definitions, and missing required values are covered.

Inbox retention, consent, spam protection, delivery, persistence, and browser
UX remain module/product acceptance gates.
