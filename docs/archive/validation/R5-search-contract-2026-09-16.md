# R5 search contract — 2026-09-16

`go test ./internal/search -count=1` PASS.

The provider-neutral search boundary rejects oversized queries and pagination,
validates locales, and produces deterministic bounded result ordering.

Provider relevance, indexing, latency, and production capacity remain open
dataset/provider gates.
