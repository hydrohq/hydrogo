# ADR-0056: Canonical module names in migration filtering

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0039](0039-conditional-cross-module-migrations.md), [ADR-0037](0037-strict-modular-monolith-boundaries.md)

Database migration filtering canonicalizes enabled module names by trimming
whitespace and lowercasing them, matching the runtime registry contract. Empty
or disabled entries are ignored; migration metadata remains authoritative.
