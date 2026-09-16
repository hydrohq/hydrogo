# R5 configurable editorial workflow — 2026-09-16

`go test ./internal/adminstudio -count=1` PASS.

Admin Studio now validates bounded custom states and transitions while keeping
the existing default publication flow backward compatible. Unknown states,
duplicate transitions, invalid identifiers, and oversized definitions fail
closed.

Scheduling, persistence, moderation UI, notifications, and module-specific
business rules remain module/Client Edition responsibilities.
