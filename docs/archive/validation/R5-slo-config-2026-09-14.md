# R5 SLO/SLI configuration validation — 2026-09-14

- `go test ./internal/ops` passes.
- YAML parsing and validation reject malformed names, windows, duplicates, and
  unsafe indicator values.
- Targets are optional; no availability, latency, or error-rate objective is
  invented by the platform.
