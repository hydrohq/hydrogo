# Production HTML Output Validation — 2026-09-22

The output boundary has three checks:

- `scripts/html-comment-lint.ps1` scans production-facing theme and generated
  HTML assets.
- `observability.HTMLCommentMiddleware` removes comments from `text/html`
  responses, including HTMX partial responses.
- Go tests verify comment removal while leaving JSON bodies unchanged.

```text
go test ./internal/observability ./internal/server -count=1
PASS: no HTML comments in production-facing theme/generated assets
```
