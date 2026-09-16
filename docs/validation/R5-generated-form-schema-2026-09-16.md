# R5 generated form schema — 2026-09-16

`go test ./internal/generator -count=1` PASS.

Generated resources now include deterministic `form.schema.json`; the artifact
set and compatibility checker require it. Existing Go fixture compilation and
generator smoke remain green.

Business validation, consent, inbox retention, spam controls, and persistence
remain module/Client Edition responsibilities.
