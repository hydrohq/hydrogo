# R6 validation: public news locale fallback

Date: 2026-09-16

`go test ./modules/news/...` passes. The service contract test proves a
`tr-TR` request tries `tr`, then serves the published `en` item when the exact
translation is absent. Non-not-found repository errors are not swallowed.

This is local contract evidence only; edition-level multilingual fixtures and
real production content acceptance remain open.
