# R5 validation: CSP report endpoint

Date: 2026-09-14

- `go test ./internal/security ./internal/server` passes.
- Legacy `csp-report` envelopes and modern direct payloads parse within the
  16 KiB body cap.
- CRLF/negative numeric fields and oversized/malformed bodies fail with 400.
- The public telemetry route bypasses CSRF only for `/csp-report`; no report
  content is persisted or logged.
